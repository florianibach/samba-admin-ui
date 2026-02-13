package main

import (
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/florianibach/samba-admin-ui/internal/state"
)

func TestUIE2E_DashboardUsersAndShares(t *testing.T) {
	fakeBin := t.TempDir()
	writeFakeCommand(t, fakeBin, "testparm", `#!/usr/bin/env bash
if [[ "$1" == "-s" ]]; then
  cat <<'OUT'
[global]
  workgroup = WORKGROUP

[media]
  path = /tmp
  read only = no

[archive]
  path = /var/empty
  read only = yes
OUT
  exit 0
fi
exit 1
`)
	writeFakeCommand(t, fakeBin, "pidof", `#!/usr/bin/env bash
if [[ "$1" == "smbd" ]]; then
  echo "123"
  exit 0
fi
exit 1
`)
	writeFakeCommand(t, fakeBin, "pdbedit", `#!/usr/bin/env bash
echo "alice:1000:Alice User"
exit 0
`)
	writeFakeCommand(t, fakeBin, "getent", `#!/usr/bin/env bash
if [[ "$1" == "passwd" ]]; then
  if [[ -z "$2" ]]; then
    cat <<'OUT'
root:x:0:0:root:/root:/bin/bash
alice:x:1000:1000::/home/alice:/bin/bash
OUT
    exit 0
  fi

  if [[ "$2" == "alice" ]]; then
    echo "alice:x:1000:1000::/home/alice:/bin/bash"
    exit 0
  fi
  exit 2
fi

if [[ "$1" == "group" ]]; then
  echo "users:x:1000:alice"
  exit 0
fi

exit 2
`)
	writeFakeCommand(t, fakeBin, "id", `#!/usr/bin/env bash
if [[ "$1" == "-G" && "$2" == "alice" ]]; then
  echo "1000 27"
  exit 0
fi
if [[ "$1" == "-nG" && "$2" == "alice" ]]; then
  echo "users sudo"
  exit 0
fi
if [[ "$1" == "-gn" && "$2" == "alice" ]]; then
  echo "users"
  exit 0
fi
exit 1
`)

	t.Setenv("PATH", fakeBin+":"+os.Getenv("PATH"))

	app := newTestApp(t)
	server := httptest.NewServer(withHeaders(newMux(app)))
	defer server.Close()

	assertBodyContains(t, mustGET(t, server.URL+"/"), "testparm:</strong> OK")
	assertBodyContains(t, mustGET(t, server.URL+"/"), "smbd:</strong> running")

	usersBody := mustGET(t, server.URL+"/users")
	assertBodyContains(t, usersBody, "Linux Users")
	assertBodyContains(t, usersBody, "alice")
	assertBodyContains(t, usersBody, "GIDs:")

	sharesBody := mustGET(t, server.URL+"/shares")
	assertBodyContains(t, sharesBody, "/shares/media")
	assertBodyContains(t, sharesBody, "enabled")
	assertBodyContains(t, sharesBody, "archive")
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := state.Open(dbPath)
	if err != nil {
		t.Fatalf("state.Open failed: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	base := template.Must(template.New("").Funcs(template.FuncMap{"now": time.Now}).ParseFS(templatesFS, "templates/layout.html"))

	sharesIndex := filepath.Join(t.TempDir(), "shares.conf")
	if err := os.WriteFile(sharesIndex, []byte("\n; samba-admin-ui:begin media\n[media]\n   include = /tmp/media.conf\n; samba-admin-ui:end media\n\n; samba-admin-ui:begin archive\n[archive]\n   include = /tmp/archive.conf\n   available = no\n   browseable = no\n; samba-admin-ui:end archive\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(shares index) failed: %v", err)
	}
	t.Setenv("UI_SHARES_INDEX", sharesIndex)

	return &App{
		base:       base,
		smbConf:    filepath.Join(t.TempDir(), "smb.conf"),
		shareRoot:  "/shares",
		store:      store,
		lastReload: time.Now(),
	}
}

func newMux(app *App) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", app.dashboard)
	mux.HandleFunc("/users", app.users)
	mux.HandleFunc("/shares", app.shares)
	return mux
}

func writeFakeCommand(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("failed writing fake command %s: %v", name, err)
	}
}

func mustGET(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status = %d", url, resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	return string(b)
}

func assertBodyContains(t *testing.T, body, want string) {
	t.Helper()
	if !strings.Contains(body, want) {
		t.Fatalf("response body did not contain %q\nBody:\n%s", want, body)
	}
}
