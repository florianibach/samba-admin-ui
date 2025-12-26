package samba

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

type LinuxUserInfo struct {
	Name string
	UID  int
	GIDs []int
}

func ListLinuxUsersHuman() ([]LinuxUserInfo, error) {
	out, err := exec.Command("getent", "passwd").Output()
	if err != nil {
		return nil, fmt.Errorf("getent passwd failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	users := make([]LinuxUserInfo, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 4 {
			continue
		}

		name := parts[0]
		if name == "nobody" {
			continue
		}

		uid, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}

		if uid < 1000 {
			continue
		}

		gids, _ := userGIDs(name)

		users = append(users, LinuxUserInfo{
			Name: name,
			UID:  uid,
			GIDs: gids,
		})
	}

	sort.Slice(users, func(i, j int) bool {
		return strings.ToLower(users[i].Name) < strings.ToLower(users[j].Name)
	})

	return users, nil
}

func userGIDs(user string) ([]int, error) {
	out, err := exec.Command("id", "-G", user).Output()
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(strings.TrimSpace(string(out)))
	gids := make([]int, 0, len(fields))
	for _, f := range fields {
		n, err := strconv.Atoi(f)
		if err != nil {
			continue
		}
		gids = append(gids, n)
	}
	sort.Ints(gids)
	return gids, nil
}
