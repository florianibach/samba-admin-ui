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

func DeleteLinuxGroup(name string) error {
	_, errStr, code, _ := run(5*time.Second, "groupdel", name)
	if code != 0 {
		return fmt.Errorf("groupdel failed: %s", strings.TrimSpace(errStr))
	}
	return nil
}

// true wenn irgendein User diese GID als Primärgruppe hat
func IsPrimaryGroupGIDUsed(gid int) (bool, error) {
	out, errStr, code, err := run(3*time.Second, "getent", "passwd")
	if err != nil && code == 0 {
		return false, err
	}
	if code != 0 {
		return false, fmt.Errorf("getent passwd failed: %s", strings.TrimSpace(errStr))
	}

	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		// name:x:uid:gid:...
		parts := strings.Split(line, ":")
		if len(parts) < 4 {
			continue
		}
		pgid, convErr := strconv.Atoi(parts[3])
		if convErr != nil {
			continue
		}
		if pgid == gid {
			return true, nil
		}
	}
	return false, nil
}

func GetPrimaryGroupName(user string) (string, error) {
	out, errStr, code, err := run(3*time.Second, "id", "-gn", user)
	if err != nil && code == 0 {
		return "", err
	}
	if code != 0 {
		return "", fmt.Errorf("id -gn failed: %s", strings.TrimSpace(errStr))
	}
	return strings.TrimSpace(out), nil
}

func GetUserGroups(user string) ([]string, error) {
	out, errStr, code, err := run(3*time.Second, "id", "-nG", user)
	if err != nil && code == 0 {
		return nil, err
	}
	if code != 0 {
		return nil, fmt.Errorf("id -nG failed: %s", strings.TrimSpace(errStr))
	}
	return strings.Fields(out), nil
}

// Sets supplementary groups exactly to the given list.
// IMPORTANT: do not include the primary group here.
func SetUserSupplementaryGroups(user string, groups []string) error {
	// usermod -G grp1,grp2 user
	arg := strings.Join(groups, ",")
	_, errStr, code, _ := run(5*time.Second, "usermod", "-G", arg, user)
	if code != 0 {
		return fmt.Errorf("usermod -G failed: %s", strings.TrimSpace(errStr))
	}
	return nil
}

func GetLinuxGroupGID(name string) (*int, error) {
	out, errStr, code, err := run(3*time.Second, "getent", "group", name)
	if err != nil && code == 0 {
		return nil, err
	}
	if code != 0 {
		return nil, fmt.Errorf("getent group failed: %s", strings.TrimSpace(errStr))
	}
	// name:x:gid:members
	parts := strings.Split(strings.TrimSpace(out), ":")
	if len(parts) < 3 {
		return nil, fmt.Errorf("unexpected getent group format")
	}
	gid, convErr := strconv.Atoi(parts[2])
	if convErr != nil {
		return nil, convErr
	}
	return &gid, nil
}
