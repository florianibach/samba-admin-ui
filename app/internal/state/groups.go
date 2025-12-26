package state

func (s *Store) ListGroups() ([]Group, error) {
	rows, err := s.DB.Query(`SELECT name, gid FROM groups ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.Name, &g.GID); err != nil {
			return nil, err
		}
		res = append(res, g)
	}
	return res, nil
}

func (s *Store) UpsertGroup(g Group) error {
	_, err := s.DB.Exec(
		`INSERT INTO groups (name, gid)
		 VALUES (?, ?)
		 ON CONFLICT(name) DO UPDATE SET gid = excluded.gid`,
		g.Name, g.GID,
	)
	return err
}

func (s *Store) DeleteGroup(name string) error {
	_, err := s.DB.Exec(`DELETE FROM groups WHERE name = ?`, name)
	return err
}
