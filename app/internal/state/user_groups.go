package state

func (s *Store) ListUserGroups(user string) ([]string, error) {
	rows, err := s.DB.Query(
		`SELECT group_name FROM user_groups WHERE user_name = ? ORDER BY group_name`,
		user,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []string
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, err
		}
		res = append(res, g)
	}
	return res, nil
}

func (s *Store) SetUserGroups(user string, groups []string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`DELETE FROM user_groups WHERE user_name = ?`, user,
	); err != nil {
		return err
	}

	for _, g := range groups {
		if _, err := tx.Exec(
			`INSERT INTO user_groups (user_name, group_name) VALUES (?, ?)`,
			user, g,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
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

func (s *Store) CountGroupAssignments(group string) (int, error) {
	row := s.DB.QueryRow(`SELECT COUNT(*) FROM user_groups WHERE group_name = ?`, group)
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
