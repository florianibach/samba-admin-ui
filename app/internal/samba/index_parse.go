package samba

import (
	"bufio"
	"os"
	"strings"
)

type ManagedShareState struct {
	Name     string
	Disabled bool
}

func ReadManagedSharesIndex(indexPath string) (map[string]ManagedShareState, error) {
	// If index doesn't exist yet -> empty
	f, err := os.Open(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]ManagedShareState{}, nil
		}
		return nil, err
	}
	defer f.Close()

	out := map[string]ManagedShareState{}

	sc := bufio.NewScanner(f)

	var inBlock bool
	var currentName string
	var disabled bool

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		lower := strings.ToLower(line)

		// begin marker
		if strings.HasPrefix(lower, "; samba-admin-ui:begin ") {
			inBlock = true
			currentName = strings.TrimSpace(line[len("; samba-admin-ui:begin "):])
			disabled = false
			continue
		}

		// end marker
		if inBlock && strings.HasPrefix(lower, "; samba-admin-ui:end ") {
			endName := strings.TrimSpace(line[len("; samba-admin-ui:end "):])
			// Only commit if names match (ignore malformed)
			if currentName != "" && endName == currentName {
				out[currentName] = ManagedShareState{Name: currentName, Disabled: disabled}
			}
			inBlock = false
			currentName = ""
			disabled = false
			continue
		}

		if !inBlock {
			continue
		}

		// inside block: detect disabled state
		// We write: "available = no"
		if strings.HasPrefix(lower, "available") && strings.Contains(lower, "= no") {
			disabled = true
		}
	}

	if err := sc.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
