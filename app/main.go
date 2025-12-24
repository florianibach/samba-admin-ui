package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/florianibach/samba-admin-ui/internal/reconcile"
	"github.com/florianibach/samba-admin-ui/internal/samba"
	"github.com/florianibach/samba-admin-ui/internal/state"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

type View struct {
	Title string
	Data  any
}

type App struct {
	base      *template.Template
	smbConf   string
	shareRoot string
	store     *state.Store

	lastReload time.Time
}

func main() {
	addr := getenv("HTTP_ADDR", ":8080")
	smbConf := getenv("SMB_CONF", "/etc/samba/smb.conf")
	shareRoot := getenv("SHARE_ROOT", "/shares")

	base := template.Must(template.New("").Funcs(template.FuncMap{
		"now": time.Now,
	}).ParseFS(templatesFS, "templates/layout.html"))

	app := &App{
		base:      base,
		smbConf:   smbConf,
		shareRoot: shareRoot,
	}

	dbPath := getenv("APP_DB", "/data/app.db")
	store, err := state.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	app.store = store

	if getenv("RECONCILE_ON_START", "true") == "true" {
		if _, err := reconcile.Apply(store); err != nil {
			log.Printf("reconcile failed: %v", err)
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	mux.HandleFunc("/", app.dashboard)
	mux.HandleFunc("/shares", app.shares)
	mux.HandleFunc("/shares/", app.shareDetail) // /shares/{name}
	mux.HandleFunc("/users", app.users)
	mux.HandleFunc("/reload", app.reload)

	mux.HandleFunc("/users/create", app.userCreate)
	mux.HandleFunc("/users/password", app.userPassword)
	mux.HandleFunc("/users/enable", app.userEnable)
	mux.HandleFunc("/users/disable", app.userDisable)
	mux.HandleFunc("/users/delete", app.userDelete)

	mux.HandleFunc("/shares/create", app.shareCreate)
	mux.HandleFunc("/shares/disable", app.shareDisable)
	mux.HandleFunc("/shares/enable", app.shareEnable)
	mux.HandleFunc("/shares/delete", app.shareDelete)

	log.Printf("samba-admin-ui listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, withHeaders(mux)))
}

func withHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func (a *App) dashboard(w http.ResponseWriter, r *http.Request) {
	type vm struct {
		Now        time.Time
		SmbConf    string
		ConfigOK   bool
		ConfigErr  string
		SmbdUp     bool
		SmbdErr    string
		LastReload *time.Time
	}

	ok, errStr := samba.TestparmOK(a.smbConf)
	smbdUp, smbdErr := samba.IsSmbdRunning()

	var lr *time.Time
	if !a.lastReload.IsZero() {
		t := a.lastReload
		lr = &t
	}

	a.render(w, "dashboard.html", "Dashboard", vm{
		Now:        time.Now(),
		SmbConf:    a.smbConf,
		ConfigOK:   ok,
		ConfigErr:  errStr,
		SmbdUp:     smbdUp,
		SmbdErr:    smbdErr,
		LastReload: lr,
	})
}

func (a *App) shares(w http.ResponseWriter, r *http.Request) {
	sections, raw, err := samba.ReadEffectiveConfig(a.smbConf)
	indexPath := getenv("UI_SHARES_INDEX", "/etc/samba/shares.d/ui/shares.conf")
	managed, err := samba.ReadManagedSharesIndex(indexPath)
	if err != nil {
		// log + treat as empty so UI still works
		managed = map[string]samba.ManagedShareState{}
	}

	type shareRow struct {
		Name      string
		Path      string
		ReadOnly  string
		ManagedBy string
		PathOK    bool
		Perms     string

		Managed  bool
		Disabled bool
	}
	type vm struct {
		SmbConf string
		Error   string
		Raw     string
		Shares  []shareRow
	}

	if err != nil {
		a.render(w, "shares.html", "Shares", vm{SmbConf: a.smbConf, Error: err.Error()})
		return
	}

	var rows []shareRow
	for name, kv := range sections {
		if strings.EqualFold(name, "global") {
			continue
		}
		path := kv["path"]
		if path == "" {
			// Sometimes shares rely on defaults; still list them.
			path = "(not set)"
		}
		ro := kv["read only"]
		if ro == "" {
			ro = kv["readonly"]
		}
		if ro == "" {
			ro = "(unknown)"
		}

		pathOK, perms := samba.PathPerms(path)
		st, ok := managed[name]
		isManaged := ok
		isDisabled := ok && st.Disabled

		rows = append(rows, shareRow{
			Name:     name,
			Path:     path,
			ReadOnly: ro,
			PathOK:   pathOK,
			Perms:    perms,
			Managed:  isManaged,
			Disabled: isDisabled,
		})

	}

	sort.Slice(rows, func(i, j int) bool {
		// UI-managed shares first
		if rows[i].Managed != rows[j].Managed {
			return rows[i].Managed && !rows[j].Managed
		}
		// Within managed: enabled first, then disabled
		if rows[i].Disabled != rows[j].Disabled {
			return !rows[i].Disabled && rows[j].Disabled
		}
		// Finally: alphabetical by name (case-insensitive)
		return strings.ToLower(rows[i].Name) < strings.ToLower(rows[j].Name)
	})

	a.render(w, "shares.html", "Shares", vm{
		SmbConf: a.smbConf,
		Raw:     raw,
		Shares:  rows,
	})
}

func (a *App) shareDetail(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/shares/")
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "/") {
		http.NotFound(w, r)
		return
	}

	sections, _, err := samba.ReadEffectiveConfig(a.smbConf)
	type vm struct {
		Name     string
		SmbConf  string
		Error    string
		KV       map[string]string
		PathOK   bool
		Perms    string
		Resolved string
	}

	if err != nil {
		a.render(w, "share_detail.html", "Share "+name, vm{Name: name, SmbConf: a.smbConf, Error: err.Error()})
		return
	}

	kv, ok := sections[name]
	if !ok {
		a.render(w, "share_detail.html", "Share "+name, vm{Name: name, SmbConf: a.smbConf, Error: "share not found in effective config"})
		return
	}

	path := kv["path"]
	pathOK, perms := samba.PathPerms(path)

	resolved := path
	if strings.HasPrefix(path, a.shareRoot) {
		resolved = path
	} else if path != "" && filepath.IsAbs(path) {
		resolved = path
	}

	a.render(w, "share_detail.html", "Share "+name, vm{
		Name:     name,
		SmbConf:  a.smbConf,
		KV:       kv,
		PathOK:   pathOK,
		Perms:    perms,
		Resolved: resolved,
	})
}

func (a *App) users(w http.ResponseWriter, r *http.Request) {
	type userRow struct {
		Name        string
		LinuxExists bool
	}
	type vm struct {
		Error string
		Users []userRow
	}

	users, err := samba.ListSambaUsers()
	if err != nil {
		a.render(w, "users.html", "Users", vm{Error: err.Error()})
		return
	}

	rows := make([]userRow, 0, len(users))
	for _, u := range users {
		rows = append(rows, userRow{
			Name:        u,
			LinuxExists: samba.LinuxUserExists(u),
		})
	}

	a.render(w, "users.html", "Users", vm{Users: rows})
}

func (a *App) reload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if err := samba.ReloadConfig(); err != nil {
		http.Error(w, "reload failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	a.lastReload = time.Now()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) render(w http.ResponseWriter, pageFile string, title string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Clone base layout and parse exactly one page file which defines {{ define "content" }}
	tpl, err := a.base.Clone()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := tpl.ParseFS(templatesFS, "templates/"+pageFile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tpl.ExecuteTemplate(w, "layout.html", View{
		Title: title,
		Data:  data,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type CreateUserRequest struct {
	Name     string
	Password string
	UID      string
	GID      string
}

func (a *App) userCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name required", 400)
		return
	}

	uid, err := parseOptionalInt(r.FormValue("uid"))
	if err != nil {
		http.Error(w, "invalid uid", 400)
		return
	}
	gid, err := parseOptionalInt(r.FormValue("gid"))
	if err != nil {
		http.Error(w, "invalid gid", 400)
		return
	}

	// 1) Desired state speichern
	if err := a.store.UpsertUser(state.User{
		Name: name,
		UID:  uid,
		GID:  gid,
	}); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// 2) Apply (OS anpassen)
	if _, err := reconcile.Apply(a.store); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func parseOptionalInt(v string) (*int, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (a *App) userPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}
	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	pw := r.FormValue("password")
	if name == "" || pw == "" {
		http.Error(w, "name and password required", 400)
		return
	}

	if err := samba.SetSambaPassword(name, pw); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (a *App) userEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}
	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name required", 400)
		return
	}
	if err := samba.EnableSambaUser(name); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (a *App) userDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}
	_ = r.ParseForm()
	name := (strings.TrimSpace(r.FormValue("name")))
	if name == "" {
		http.Error(w, "name required", 400)
		return
	}
	if err := samba.DisableSambaUser(name); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (a *App) userDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}
	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name required", 400)
		return
	}
	if err := samba.DeleteSambaUser(name); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

type ShareCreateForm struct {
	SmbConf     string
	SnippetDir  string
	Name        string
	Path        string
	ReadOnly    bool
	Browseable  bool
	ValidUsers  string
	Error       string
	IncludeGlob string
}

func (a *App) shareCreate(w http.ResponseWriter, r *http.Request) {
	sharesDir := getenv("UI_SHARES_DIR", "/etc/samba/shares.d/ui")
	indexPath := getenv("UI_SHARES_INDEX", "/etc/samba/shares.d/ui/shares.conf")

	if r.Method == http.MethodGet {
		a.render(w, "share_create.html", "Create Share", ShareCreateForm{
			SmbConf:    a.smbConf,
			SnippetDir: sharesDir,
			Browseable: true,
		})
		return
	}

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/shares", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	form := ShareCreateForm{
		SmbConf:    a.smbConf,
		SnippetDir: sharesDir,
		Name:       strings.TrimSpace(r.FormValue("name")),
		Path:       strings.TrimSpace(r.FormValue("path")),
		ReadOnly:   r.FormValue("readOnly") == "on",
		Browseable: r.FormValue("browseable") != "off",
		ValidUsers: strings.TrimSpace(r.FormValue("validUsers")),
	}

	// 0) smb.conf must include our index file (read-only check)
	if err := samba.CheckSmbConfIncludesIndex(a.smbConf, indexPath); err != nil {
		form.Error = err.Error() + " (smb.conf is read-only; please add it manually)"
		a.render(w, "share_create.html", "Create Share", form)
		return
	}

	// 1) Write share file: /etc/samba/shares.d/ui/<name>.conf
	shareFile := filepath.Join(sharesDir, form.Name+".conf")
	_, err := samba.CreateShareSnippet(sharesDir, samba.CreateShareOptions{
		Name:       form.Name,
		Path:       form.Path,
		ReadOnly:   form.ReadOnly,
		Browseable: form.Browseable,
		ValidUsers: form.ValidUsers,
	})
	if err != nil {
		form.Error = err.Error()
		a.render(w, "share_create.html", "Create Share", form)
		return
	}

	// 2) Ensure shares index references that share file
	if err := samba.EnsureIndexReferencesShare(indexPath, form.Name, shareFile); err != nil {
		form.Error = "failed to update shares index: " + err.Error()
		a.render(w, "share_create.html", "Create Share", form)
		return
	}

	// 3) Reload Samba
	if err := samba.ReloadConfig(); err != nil {
		form.Error = "reload failed: " + err.Error()
		a.render(w, "share_create.html", "Create Share", form)
		return
	}

	http.Redirect(w, r, "/shares", http.StatusSeeOther)
}

func (a *App) shareDisable(w http.ResponseWriter, r *http.Request) {
	a.setShareDisabled(w, r, true)
}

func (a *App) shareEnable(w http.ResponseWriter, r *http.Request) {
	a.setShareDisabled(w, r, false)
}

func (a *App) setShareDisabled(w http.ResponseWriter, r *http.Request, disabled bool) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/shares", http.StatusSeeOther)
		return
	}
	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name required", 400)
		return
	}

	sharesDir := getenv("UI_SHARES_DIR", "/etc/samba/shares.d/ui")
	indexPath := getenv("UI_SHARES_INDEX", "/etc/samba/shares.d/ui/shares.conf")

	// require include exists in smb.conf
	if err := samba.CheckSmbConfIncludesIndex(a.smbConf, indexPath); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	shareFile := filepath.Join(sharesDir, name+".conf")

	// Ensure index entry exists (best effort)
	_ = samba.EnsureIndexReferencesShare(indexPath, name, shareFile)

	if err := samba.SetShareDisabled(indexPath, name, shareFile, disabled); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err := samba.ReloadConfig(); err != nil {
		http.Error(w, "reload failed: "+err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/shares", http.StatusSeeOther)
}

func (a *App) shareDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/shares", http.StatusSeeOther)
		return
	}
	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name required", 400)
		return
	}

	sharesDir := getenv("UI_SHARES_DIR", "/etc/samba/shares.d/ui")
	indexPath := getenv("UI_SHARES_INDEX", "/etc/samba/shares.d/ui/shares.conf")

	if err := samba.CheckSmbConfIncludesIndex(a.smbConf, indexPath); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	shareFile := filepath.Join(sharesDir, name+".conf")

	// Remove from index (only if managed marker exists)
	if err := samba.RemoveShareFromIndex(indexPath, name); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Delete share snippet file (ignore if missing)
	_ = os.Remove(shareFile)

	if err := samba.ReloadConfig(); err != nil {
		http.Error(w, "reload failed: "+err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/shares", http.StatusSeeOther)
}
