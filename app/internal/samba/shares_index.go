package samba

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func indexBlock(shareName, shareFile string, disabled bool) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n; samba-admin-ui:begin %s\n", shareName))
	b.WriteString(fmt.Sprintf("[%s]\n", shareName))
	b.WriteString(fmt.Sprintf("   include = %s\n", shareFile))
	if disabled {
		b.WriteString("   available = no\n")
		b.WriteString("   browseable = no\n")
	}
	b.WriteString(fmt.Sprintf("; samba-admin-ui:end %s\n", shareName))
	return b.String()
}

func EnsureIndexReferencesShare(indexPath string, shareName string, shareFilePath string) error {
	if err := os.MkdirAll(filepath.Dir(indexPath), 0755); err != nil {
		return err
	}

	existing := ""
	if b, err := os.ReadFile(indexPath); err == nil {
		existing = string(b)
	}

	// If already referenced (either marker-block or plain include), do nothing.
	if strings.Contains(existing, shareFilePath) || strings.Contains(existing, fmt.Sprintf("[%s]", shareName)) {
		return nil
	}

	newContent := existing + indexBlock(shareName, shareFilePath, false)
	return os.WriteFile(indexPath, []byte(newContent), 0644)
}

func SetShareDisabled(indexPath, shareName, shareFilePath string, disabled bool) error {
	if err := os.MkdirAll(filepath.Dir(indexPath), 0755); err != nil {
		return err
	}

	existing := ""
	if b, err := os.ReadFile(indexPath); err == nil {
		existing = string(b)
	}

	// Prefer marker-based replace
	re := regexp.MustCompile(`(?s)\n; samba-admin-ui:begin ` + regexp.QuoteMeta(shareName) + `\n.*?; samba-admin-ui:end ` + regexp.QuoteMeta(shareName) + `\n`)
	if re.MatchString(existing) {
		repl := indexBlock(shareName, shareFilePath, disabled)
		out := re.ReplaceAllString(existing, repl)
		return os.WriteFile(indexPath, []byte(out), 0644)
	}

	// Fallback: if no marker exists, append a managed marker block and keep old one
	// (MVP-safe: avoids accidentally deleting user-managed content)
	out := existing + indexBlock(shareName, shareFilePath, disabled)
	return os.WriteFile(indexPath, []byte(out), 0644)
}

func RemoveShareFromIndex(indexPath, shareName string) error {
	b, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}
	existing := string(b)

	// Remove marker block if present
	re := regexp.MustCompile(`(?s)\n; samba-admin-ui:begin ` + regexp.QuoteMeta(shareName) + `\n.*?; samba-admin-ui:end ` + regexp.QuoteMeta(shareName) + `\n`)
	if re.MatchString(existing) {
		out := re.ReplaceAllString(existing, "\n")
		return os.WriteFile(indexPath, []byte(out), 0644)
	}

	// No marker -> do not attempt risky deletes in MVP
	return fmt.Errorf("share block for %s not managed by UI (no markers found)", shareName)
}
