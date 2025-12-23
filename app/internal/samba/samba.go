package samba

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func run(timeout time.Duration, name string, args ...string) (string, string, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exitCode = ee.ExitCode()
		} else if errors.Is(err, context.DeadlineExceeded) {
			return out.String(), errb.String(), 124, fmt.Errorf("timeout running %s", name)
		} else {
			return out.String(), errb.String(), 1, err
		}
	}

	return out.String(), errb.String(), exitCode, nil
}

func runWithStdin(timeout time.Duration, stdin string, name string, args ...string) (string, string, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	cmd.Stdin = strings.NewReader(stdin)

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exitCode = ee.ExitCode()
		} else if errors.Is(err, context.DeadlineExceeded) {
			return out.String(), errb.String(), 124, fmt.Errorf("timeout running %s", name)
		} else {
			return out.String(), errb.String(), 1, err
		}
	}
	return out.String(), errb.String(), exitCode, nil
}

func TestparmOK(smbConf string) (bool, string) {
	_, errStr, code, err := run(5*time.Second, "testparm", "-s", smbConf)
	if err != nil && code == 0 {
		return false, err.Error()
	}
	if code != 0 {
		trim := strings.TrimSpace(errStr)
		if trim == "" && err != nil {
			trim = err.Error()
		}
		return false, trim
	}
	return true, ""
}

// ReadEffectiveConfig returns INI-like sections from `testparm -s`.
// It does NOT try to preserve formatting; it's for display and health checks.
func ReadEffectiveConfig(smbConf string) (map[string]map[string]string, string, error) {
	out, errStr, code, err := run(8*time.Second, "testparm", "-s", smbConf)
	if code != 0 {
		if errStr == "" && err != nil {
			errStr = err.Error()
		}
		return nil, "", fmt.Errorf("testparm failed: %s", strings.TrimSpace(errStr))
	}

	sections := make(map[string]map[string]string)
	var current string
	lines := strings.Split(out, "\n")
	for _, ln := range lines {
		line := strings.TrimSpace(ln)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			current = strings.TrimSpace(current)
			if _, ok := sections[current]; !ok {
				sections[current] = map[string]string{}
			}
			continue
		}
		if current == "" {
			continue
		}
		// key = value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(parts[0]))
		v := strings.TrimSpace(parts[1])
		sections[current][k] = v
	}

	return sections, out, nil
}

func IsSmbdRunning() (bool, string) {
	// pidof returns exit 0 if running
	_, errStr, code, err := run(2*time.Second, "pidof", "smbd")
	if err != nil && code == 0 {
		return false, err.Error()
	}
	if code != 0 {
		msg := strings.TrimSpace(errStr)
		if msg == "" {
			msg = "smbd not running"
		}
		return false, msg
	}
	return true, ""
}

func ReloadConfig() error {
	// Works without systemd inside containers.
	_, errStr, code, err := run(5*time.Second, "smbcontrol", "all", "reload-config")
	if err != nil && code == 0 {
		return err
	}
	if code != 0 {
		return fmt.Errorf("smbcontrol failed: %s", strings.TrimSpace(errStr))
	}
	return nil
}

func ListSambaUsers() ([]string, error) {
	out, errStr, code, err := run(5*time.Second, "pdbedit", "-L")
	if err != nil && code == 0 {
		return nil, err
	}
	if code != 0 {
		return nil, fmt.Errorf("pdbedit failed: %s", strings.TrimSpace(errStr))
	}
	var users []string
	for _, ln := range strings.Split(out, "\n") {
		line := strings.TrimSpace(ln)
		if line == "" {
			continue
		}
		// format: user:UID:... (varies)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) >= 1 && parts[0] != "" {
			users = append(users, parts[0])
		}
	}
	return users, nil
}

func LinuxUserExists(user string) bool {
	_, _, code, _ := run(2*time.Second, "getent", "passwd", user)
	return code == 0
}

func PathPerms(path string) (bool, string) {
	if path == "" || path == "(not set)" {
		return false, "(no path)"
	}
	fi, err := os.Stat(path)
	if err != nil {
		return false, err.Error()
	}
	mode := fi.Mode()
	perms := mode.Perm().String()

	uid := "?"
	gid := "?"
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		uid = fmt.Sprintf("%d", st.Uid)
		gid = fmt.Sprintf("%d", st.Gid)
	}
	return true, fmt.Sprintf("uid=%s gid=%s mode=%s", uid, gid, perms)
}

func CreateLinuxUser(name string, uid *int, gid *int) error {
	if gid != nil {
		if err := EnsureGroupExists(*gid); err != nil {
			return err
		}
	}

	args := []string{"-M", "-s", "/usr/sbin/nologin"}
	if uid != nil {
		args = append(args, "-u", strconv.Itoa(*uid))
	}
	if gid != nil {
		args = append(args, "-g", strconv.Itoa(*gid))
	}
	args = append(args, name)

	_, errStr, code, _ := run(5*time.Second, "useradd", args...)
	if code != 0 {
		return fmt.Errorf("useradd failed: %s", strings.TrimSpace(errStr))
	}
	return nil
}

func CreateSambaUser(name string, password string) error {
	// smbpasswd reads password twice from stdin
	// Use: printf "pw\npw\n" | smbpasswd -a -s name
	in := fmt.Sprintf("%s\n%s\n", password, password)
	_, errStr, code, err := runWithStdin(8*time.Second, in, "smbpasswd", "-a", "-s", name)
	if err != nil && code == 0 {
		return err
	}
	if code != 0 {
		return fmt.Errorf("smbpasswd -a failed: %s", strings.TrimSpace(errStr))
	}
	return nil
}

func EnsureGroupExists(gid int) error {
	// check by gid
	_, _, code, _ := run(2*time.Second, "getent", "group", strconv.Itoa(gid))
	if code == 0 {
		return nil
	}

	// create group with same numeric gid and name = gid
	_, errStr, code, _ := run(5*time.Second, "groupadd", "-g", strconv.Itoa(gid), strconv.Itoa(gid))
	if code != 0 {
		return fmt.Errorf("groupadd failed: %s", strings.TrimSpace(errStr))
	}
	return nil
}

func SetSambaPassword(name, password string) error {
	in := fmt.Sprintf("%s\n%s\n", password, password)
	_, errStr, code, err := runWithStdin(8*time.Second, in, "smbpasswd", "-s", name)
	if err != nil && code == 0 {
		return err
	}
	if code != 0 {
		return fmt.Errorf("smbpasswd set password failed: %s", strings.TrimSpace(errStr))
	}
	return nil
}

func EnableSambaUser(name string) error {
	_, errStr, code, _ := run(5*time.Second, "smbpasswd", "-e", name)
	if code != 0 {
		return fmt.Errorf("smbpasswd -e failed: %s", strings.TrimSpace(errStr))
	}
	return nil
}

func DisableSambaUser(name string) error {
	_, errStr, code, _ := run(5*time.Second, "smbpasswd", "-d", name)
	if code != 0 {
		return fmt.Errorf("smbpasswd -d failed: %s", strings.TrimSpace(errStr))
	}
	return nil
}

func DeleteSambaUser(name string) error {
	_, errStr, code, _ := run(5*time.Second, "smbpasswd", "-x", name)
	if code != 0 {
		return fmt.Errorf("smbpasswd -x failed: %s", strings.TrimSpace(errStr))
	}
	return nil
}
