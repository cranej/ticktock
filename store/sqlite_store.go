package store

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

type Sqlite struct {
	db *sql.DB
}

func (s *Sqlite) Start(entry *UnfinishedEntry) error {
	return nil
}

func (s *Sqlite) StartTitle(title, notes string) error {
	return nil
}

func newSqlite(db string) (Sqlite, error) {
	pool, err := sql.Open("sqlite3", db)
	if err != nil {
		return Sqlite{}, err
	}

	initSql := `CREATE TABLE IF NOT EXISTS clocking (
                id INTEGER PRIMARY KEY,
                title TEXT NOT NULL,
                start TEXT NOT NULL,
                end TEXT NULL,
                notes TEXT NULL
             )`

	if _, err := pool.Exec(initSql); err != nil {
		return Sqlite{}, err
	}

	return Sqlite{pool}, nil
}
