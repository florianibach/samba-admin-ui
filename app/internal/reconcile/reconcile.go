package reconcile

import (
	"fmt"
	"time"

	"github.com/florianibach/samba-admin-ui/internal/samba"
	"github.com/florianibach/samba-admin-ui/internal/state"
)

type Result struct {
	Actions []string
}

func Apply(store *state.Store) (*Result, error) {
	res := &Result{}

	groups, err := store.ListGroups()
	if err != nil {
		return nil, err
	}
	users, err := store.ListUsers()
	if err != nil {
		return nil, err
	}
	mems, err := store.ListMemberships()
	if err != nil {
		return nil, err
	}

	// 1) Ensure groups
	for _, g := range groups {
		if samba.LinuxGroupExists(g.Name) {
			continue
		}
		if err := samba.CreateLinuxGroup(g.Name, g.GID); err != nil {
			return nil, fmt.Errorf("create group %s: %w", g.Name, err)
		}
		res.Actions = append(res.Actions, "groupadd "+g.Name)
	}

	// 2) Ensure users
	for _, u := range users {
		if samba.LinuxUserExists(u.Name) {
			continue
		}
		if err := samba.CreateLinuxUser(u.Name, u.UID, u.GID); err != nil {
			return nil, fmt.Errorf("create user %s: %w", u.Name, err)
		}
		res.Actions = append(res.Actions, "useradd "+u.Name)
	}

	// 3) Ensure memberships
	for _, m := range mems {
		ok, err := samba.IsUserInGroup(m.User, m.Group)
		if err != nil {
			return nil, err
		}
		if ok {
			continue
		}

		if err := samba.AddUserToGroup(m.User, m.Group); err != nil {
			return nil, fmt.Errorf("add %s to %s: %w", m.User, m.Group, err)
		}
		res.Actions = append(res.Actions, "usermod -aG "+m.Group+" "+m.User)
	}

	_ = time.Now()
	return res, nil
}
