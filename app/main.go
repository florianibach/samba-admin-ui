package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/florianibach/samba-admin-ui/internal/samba"
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
	type shareRow struct {
		Name      string
		Path      string
		ReadOnly  string
		ManagedBy string
		PathOK    bool
		Perms     string
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

		rows = append(rows, shareRow{
			Name:      name,
			Path:      path,
			ReadOnly:  ro,
			ManagedBy: "manual", // MVP: everything is manual; later add managed/custom tags
			PathOK:    pathOK,
			Perms:     perms,
		})
	}

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

	pw := r.FormValue("password")
	pw2 := r.FormValue("password_confirm")

	if pw != pw2 {
		http.Error(w, "passwords do not match", http.StatusBadRequest)
		return
	}

	req := CreateUserRequest{
		Name:     strings.TrimSpace(r.FormValue("name")),
		Password: r.FormValue("password"),
		UID:      strings.TrimSpace(r.FormValue("uid")),
		GID:      strings.TrimSpace(r.FormValue("gid")),
	}
	if req.Name == "" || req.Password == "" {
		http.Error(w, "name and password required", 400)
		return
	}

	var uidPtr *int
	var gidPtr *int
	if req.UID != "" {
		v, err := strconv.Atoi(req.UID)
		if err != nil {
			http.Error(w, "uid must be numeric", 400)
			return
		}
		uidPtr = &v
	}
	if req.GID != "" {
		v, err := strconv.Atoi(req.GID)
		if err != nil {
			http.Error(w, "gid must be numeric", 400)
			return
		}
		gidPtr = &v
	}

	// Ensure linux user exists
	if !samba.LinuxUserExists(req.Name) {
		if err := samba.CreateLinuxUser(req.Name, uidPtr, gidPtr); err != nil {
			http.Error(w, "create linux user failed: "+err.Error(), 500)
			return
		}
	}

	// Create samba user (or update password if already exists)
	if err := samba.CreateSambaUser(req.Name, req.Password); err != nil {
		// fallback: if already exists, set password
		if err2 := samba.SetSambaPassword(req.Name, req.Password); err2 != nil {
			http.Error(w, "create samba user failed: "+err.Error(), 500)
			return
		}
	}

	http.Redirect(w, r, "/users", http.StatusSeeOther)
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
