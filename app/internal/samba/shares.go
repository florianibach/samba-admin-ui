package samba

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var shareNameRx = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

func CheckSmbConfIncludesIndex(smbConfPath string, indexPath string) error {
	b, err := os.ReadFile(smbConfPath)
	if err != nil {
		return err
	}
	txt := string(b)

	// Accept include with either absolute path or with different spacing
	needle := "include"
	if !strings.Contains(strings.ToLower(txt), needle) {
		return fmt.Errorf("smb.conf has no include statements; expected include = %s", indexPath)
	}

	// Very MVP: require the indexPath string to appear somewhere
	if !strings.Contains(txt, indexPath) {
		return fmt.Errorf("missing required include in smb.conf: include = %s", indexPath)
	}
	return nil
}

type CreateShareOptions struct {
	Name       string
	Path       string
	ReadOnly   bool
	Browseable bool
	ValidUsers string // e.g. "vater, @eltern"
}

func CreateShareSnippet(snippetDir string, opt CreateShareOptions) (string, error) {
	name := strings.TrimSpace(opt.Name)
	if name == "" || !shareNameRx.MatchString(name) {
		return "", fmt.Errorf("invalid share name (use letters, numbers, . _ -)")
	}

	path := strings.TrimSpace(opt.Path)
	if path == "" || !strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("path must be an absolute path")
	}

	file := filepath.Join(snippetDir, name+".conf")
	if err := os.MkdirAll(snippetDir, 0755); err != nil {
		return "", err
	}

	ro := "no"
	if opt.ReadOnly {
		ro = "yes"
	}
	br := "yes"
	if !opt.Browseable {
		br = "no"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("path = %s\n", path))
	b.WriteString(fmt.Sprintf("read only = %s\n", ro))
	b.WriteString(fmt.Sprintf("browseable = %s\n", br))
	b.WriteString("guest ok = no\n")

	vu := strings.TrimSpace(opt.ValidUsers)
	if vu != "" {
		parts := strings.Split(vu, ",")
		var cleaned []string
		for _, p := range parts {
			if c := strings.TrimSpace(p); c != "" {
				cleaned = append(cleaned, c)
			}
		}
		if len(cleaned) > 0 {
			b.WriteString(fmt.Sprintf("valid users = %s\n", strings.Join(cleaned, ", ")))
		}
	}

	// Optional Defaults (wenn du willst – passt zu deinen anderen Shares)
	b.WriteString("create mask = 0660\n")
	b.WriteString("directory mask = 0770\n")

	// Wichtig: Datei endet mit Newline
	b.WriteString("\n")

	if err := os.WriteFile(file, []byte(b.String()), 0644); err != nil {
		return "", err
	}
	return file, nil
}
