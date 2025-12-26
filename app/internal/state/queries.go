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
