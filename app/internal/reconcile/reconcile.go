package reconcile

import (
	"fmt"
	"strings"
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

	// 1) Ensure groups (and persist learned GID)
	for _, g := range groups {
		if !samba.LinuxGroupExists(g.Name) {
			if err := samba.CreateLinuxGroup(g.Name, g.GID); err != nil {
				return nil, fmt.Errorf("create group %s: %w", g.Name, err)
			}
			res.Actions = append(res.Actions, "groupadd "+g.Name)
		}

		// If DB has no GID yet, learn from OS and persist.
		if g.GID == nil {
			gid, err := samba.GetLinuxGroupGID(g.Name)
			if err != nil {
				return nil, fmt.Errorf("read gid for group %s: %w", g.Name, err)
			}
			if gid != nil {
				if err := store.UpdateGroupGID(g.Name, *gid); err != nil {
					return nil, fmt.Errorf("persist gid for group %s: %w", g.Name, err)
				}
				res.Actions = append(res.Actions, fmt.Sprintf("db: set group %s gid=%d", g.Name, *gid))
			}
		}
	}

	// 2) Ensure users (and persist learned UID/GID)
	for _, u := range users {
		created := false
		if !samba.LinuxUserExists(u.Name) {
			if err := samba.CreateLinuxUser(u.Name, u.UID, u.GID); err != nil {
				return nil, fmt.Errorf("create user %s: %w", u.Name, err)
			}
			res.Actions = append(res.Actions, "useradd "+u.Name)
			created = true
		}

		// If DB UID or GID is missing, learn from OS and persist.
		// We do this even if the user already existed, because DB might be empty/new.
		if u.UID == nil || u.GID == nil || created {
			uid, gid, err := samba.GetLinuxUserUIDGID(u.Name)
			if err != nil {
				return nil, fmt.Errorf("read uid/gid for user %s: %w", u.Name, err)
			}

			needPersist := false
			newUID := uid
			newGID := gid

			// Only persist if missing (or created). We do NOT "fix" OS here.
			if u.UID == nil {
				needPersist = true
			} else {
				newUID = *u.UID
			}
			if u.GID == nil {
				needPersist = true
			} else {
				newGID = *u.GID
			}

			// If created, it's safe and useful to persist actual ids to DB.
			// If not created, we still persist missing values, but we never override existing DB values here.
			if created {
				needPersist = true
				newUID = uid
				newGID = gid
			}

			if needPersist {
				if err := store.UpdateUserIDs(u.Name, newUID, newGID); err != nil {
					return nil, fmt.Errorf("persist uid/gid for user %s: %w", u.Name, err)
				}
				res.Actions = append(res.Actions, fmt.Sprintf("db: set user %s uid=%d gid=%d", u.Name, newUID, newGID))
			}
		}
	}

	// 3) Ensure memberships (idempotent)
	for _, m := range mems {
		ok, err := samba.IsUserInGroup(m.User, m.Group)
		if err != nil {
			return nil, fmt.Errorf("check membership %s in %s: %w", m.User, m.Group, err)
		}
		if ok {
			continue
		}

		// best effort: ensure group exists
		if !samba.LinuxGroupExists(m.Group) {
			// if a group membership exists in DB, the group should exist in DB too.
			// but handle gracefully.
			if err := samba.CreateLinuxGroup(m.Group, nil); err != nil {
				return nil, fmt.Errorf("create missing group %s for membership: %w", m.Group, err)
			}
			res.Actions = append(res.Actions, "groupadd "+m.Group)
		}

		if err := samba.AddUserToGroup(m.User, m.Group); err != nil {
			return nil, fmt.Errorf("add %s to %s: %w", m.User, m.Group, err)
		}
		res.Actions = append(res.Actions, "usermod -aG "+m.Group+" "+m.User)
	}

	// Keep for debugging / future timing metrics
	_ = time.Now()

	// Small cleanup: normalize action strings (optional)
	for i := range res.Actions {
		res.Actions[i] = strings.TrimSpace(res.Actions[i])
	}

	return res, nil
}
