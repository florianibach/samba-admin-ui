package state

import "database/sql"

func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.DB.Query(`SELECT name, uid, gid FROM users ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		var uid, gid sql.NullInt64
		if err := rows.Scan(&u.Name, &uid, &gid); err != nil {
			return nil, err
		}
		if uid.Valid {
			v := int(uid.Int64)
			u.UID = &v
		}
		if gid.Valid {
			v := int(gid.Int64)
			u.GID = &v
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) ListGroups() ([]Group, error) {
	rows, err := s.DB.Query(`SELECT name, gid FROM groups ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Group
	for rows.Next() {
		var g Group
		var gid sql.NullInt64
		if err := rows.Scan(&g.Name, &gid); err != nil {
			return nil, err
		}
		if gid.Valid {
			v := int(gid.Int64)
			g.GID = &v
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) ListMemberships() ([]Membership, error) {
	rows, err := s.DB.Query(`SELECT user_name, group_name FROM user_groups ORDER BY user_name, group_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Membership
	for rows.Next() {
		var m Membership
		if err := rows.Scan(&m.User, &m.Group); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
