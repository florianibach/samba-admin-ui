package samba

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func GetLinuxUserUIDGID(name string) (uid int, gid int, err error) {
	out, errStr, code, e := run(3*time.Second, "getent", "passwd", name)
	if e != nil && code == 0 {
		return 0, 0, e
	}
	if code != 0 {
		return 0, 0, fmt.Errorf("getent passwd failed: %s", strings.TrimSpace(errStr))
	}
	parts := strings.Split(strings.TrimSpace(out), ":")
	if len(parts) < 4 {
		return 0, 0, fmt.Errorf("unexpected getent passwd format")
	}
	uid, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, err
	}
	gid, err = strconv.Atoi(parts[3])
	if err != nil {
		return 0, 0, err
	}
	return uid, gid, nil
}
