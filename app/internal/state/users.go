package state

import (
	"database/sql"
)

func (s *Store) UpsertUser(u User) error {
	_, err := s.DB.Exec(`
INSERT INTO users(name, uid, gid)
VALUES(?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
  uid = excluded.uid,
  gid = excluded.gid
`, u.Name, nullableInt(u.UID), nullableInt(u.GID))
	return err
}

func nullableInt(v *int) any {
	if v == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: int64(*v), Valid: true}
}
