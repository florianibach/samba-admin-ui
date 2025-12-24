package state

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	s := &Store{DB: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error { return s.DB.Close() }

func (s *Store) migrate() error {
	_, err := s.DB.Exec(`
CREATE TABLE IF NOT EXISTS users (
  name TEXT PRIMARY KEY,
  uid INTEGER NULL,
  gid INTEGER NULL,
  created_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS groups (
  name TEXT PRIMARY KEY,
  gid INTEGER NULL,
  created_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS user_groups (
  user_name TEXT NOT NULL,
  group_name TEXT NOT NULL,
  PRIMARY KEY (user_name, group_name),
  FOREIGN KEY (user_name) REFERENCES users(name) ON DELETE CASCADE,
  FOREIGN KEY (group_name) REFERENCES groups(name) ON DELETE CASCADE
);
`)
	return err
}
