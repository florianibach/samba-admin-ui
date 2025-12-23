package samba

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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

func EnsureIndexReferencesShare(indexPath string, shareName string, shareFilePath string) error {
	if err := os.MkdirAll(filepath.Dir(indexPath), 0755); err != nil {
		return err
	}

	var existing string
	if b, err := os.ReadFile(indexPath); err == nil {
		existing = string(b)
	}

	// If already referenced, do nothing
	if strings.Contains(existing, shareFilePath) {
		return nil
	}

	// Append new section that includes the share file
	block := fmt.Sprintf("\n[%s]\n   include = %s\n", shareName, shareFilePath)
	newContent := existing + block

	return os.WriteFile(indexPath, []byte(newContent), 0644)
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

	// prevent path traversal
	file := filepath.Join(snippetDir, name+".conf")
	if !strings.HasPrefix(filepath.Clean(file), filepath.Clean(snippetDir)+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid target path")
	}

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
	b.WriteString(fmt.Sprintf("[%s]\n", name))
	b.WriteString(fmt.Sprintf("path = %s\n", path))
	b.WriteString(fmt.Sprintf("read only = %s\n", ro))
	b.WriteString(fmt.Sprintf("browseable = %s\n", br))
	b.WriteString("guest ok = no\n")

	vu := strings.TrimSpace(opt.ValidUsers)
	if vu != "" {
		// normalize: allow "a,b" -> "a, b"
		parts := strings.Split(vu, ",")
		var cleaned []string
		for _, p := range parts {
			c := strings.TrimSpace(p)
			if c == "" {
				continue
			}
			cleaned = append(cleaned, c)
		}
		if len(cleaned) > 0 {
			b.WriteString(fmt.Sprintf("valid users = %s\n", strings.Join(cleaned, ", ")))
		}
	}

	b.WriteString("\n")

	if err := os.WriteFile(file, []byte(b.String()), 0644); err != nil {
		return "", err
	}

	// quick sanity check: reload is separate, but nice to ensure file written
	_ = time.Now()
	return file, nil
}
