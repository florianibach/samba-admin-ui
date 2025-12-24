package samba

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func LinuxGroupExists(name string) bool {
	_, _, code, _ := run(2*time.Second, "getent", "group", name)
	return code == 0
}

func CreateLinuxGroup(name string, gid *int) error {
	args := []string{}
	if gid != nil {
		args = append(args, "-g", strconv.Itoa(*gid))
	}
	args = append(args, name)
	_, errStr, code, _ := run(5*time.Second, "groupadd", args...)
	if code != 0 {
		return fmt.Errorf("groupadd failed: %s", strings.TrimSpace(errStr))
	}
	return nil
}

func IsUserInGroup(user, group string) (bool, error) {
	// id -nG user -> "grp1 grp2 ..."
	out, errStr, code, err := run(3*time.Second, "id", "-nG", user)
	if err != nil && code == 0 {
		return false, err
	}
	if code != 0 {
		return false, fmt.Errorf("id -nG failed: %s", strings.TrimSpace(errStr))
	}
	for _, g := range strings.Fields(out) {
		if g == group {
			return true, nil
		}
	}
	return false, nil
}

func AddUserToGroup(user, group string) error {
	_, errStr, code, _ := run(5*time.Second, "usermod", "-a", "-G", group, user)
	if code != 0 {
		return fmt.Errorf("usermod failed: %s", strings.TrimSpace(errStr))
	}
	return nil
}
