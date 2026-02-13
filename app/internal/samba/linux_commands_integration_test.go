package samba

import (
	"os/user"
	"strconv"
	"testing"
)

func TestGetLinuxUserUIDGID_UsesRealSystemCommand(t *testing.T) {
	t.Parallel()

	cur, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current failed: %v", err)
	}

	uid, gid, err := GetLinuxUserUIDGID(cur.Username)
	if err != nil {
		t.Fatalf("GetLinuxUserUIDGID failed: %v", err)
	}

	expectedUID, err := strconv.Atoi(cur.Uid)
	if err != nil {
		t.Fatalf("invalid current uid %q: %v", cur.Uid, err)
	}
	expectedGID, err := strconv.Atoi(cur.Gid)
	if err != nil {
		t.Fatalf("invalid current gid %q: %v", cur.Gid, err)
	}

	if uid != expectedUID {
		t.Fatalf("uid mismatch: got %d want %d", uid, expectedUID)
	}
	if gid != expectedGID {
		t.Fatalf("gid mismatch: got %d want %d", gid, expectedGID)
	}
}

func TestCurrentUserPrimaryGroupAndMembership_UsesRealSystemCommands(t *testing.T) {
	t.Parallel()

	cur, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current failed: %v", err)
	}

	primaryGroup, err := GetPrimaryGroupName(cur.Username)
	if err != nil {
		t.Fatalf("GetPrimaryGroupName failed: %v", err)
	}
	if primaryGroup == "" {
		t.Fatalf("expected non-empty primary group")
	}

	if !LinuxGroupExists(primaryGroup) {
		t.Fatalf("expected LinuxGroupExists(%q) to be true", primaryGroup)
	}

	inGroup, err := IsUserInGroup(cur.Username, primaryGroup)
	if err != nil {
		t.Fatalf("IsUserInGroup failed: %v", err)
	}
	if !inGroup {
		t.Fatalf("expected user %q to be in group %q", cur.Username, primaryGroup)
	}

	groups, err := GetUserGroups(cur.Username)
	if err != nil {
		t.Fatalf("GetUserGroups failed: %v", err)
	}
	if len(groups) == 0 {
		t.Fatalf("expected at least one group for %q", cur.Username)
	}
}
