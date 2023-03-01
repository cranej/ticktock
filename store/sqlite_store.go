package store

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

type Sqlite struct {
	db *sql.DB
}

func (s *Sqlite) Start(entry *UnfinishedEntry) error {
	_, err := s.db.Exec(`INSERT INTO clocking (title, start, notes)
	VALUES(?,?,?)`,
		entry.Title,
		entry.Start.Format(time.RFC3339),
		entry.Notes)

	return err
}

func (s *Sqlite) StartTitle(title, notes string) error {
	return s.Start(&UnfinishedEntry{
		Title: title,
		Start: time.Now().UTC(),
		Notes: notes,
	})
}

func (s *Sqlite) Finish(title string) error {
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
