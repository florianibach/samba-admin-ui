package samba

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type LinuxGroupInfo struct {
	Name    string
	GID     int
	Members []string
}

func ListLinuxGroups() ([]LinuxGroupInfo, error) {
	out, errStr, code, _ := run(3*time.Second, "getent", "group")
	if code != 0 {
		return nil, fmt.Errorf("getent group failed: %s", strings.TrimSpace(errStr))
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	res := make([]LinuxGroupInfo, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// name:x:gid:member1,member2
		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			continue
		}

		name := parts[0]
		gid, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}

		members := []string{}
		if len(parts) >= 4 && strings.TrimSpace(parts[3]) != "" {
			members = strings.Split(parts[3], ",")
		}

		res = append(res, LinuxGroupInfo{Name: name, GID: gid, Members: members})
	}

	sort.Slice(res, func(i, j int) bool {
		return strings.ToLower(res[i].Name) < strings.ToLower(res[j].Name)
	})

	return res, nil
}
