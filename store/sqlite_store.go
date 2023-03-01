package store

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

type Sqlite struct {
	db *sql.DB
}

var ErrOngoingExists = errors.New("Ongoing entry eixsts")
var ErrDuplicateEntry = errors.New("Entry already started")

func (s *Sqlite) Start(entry *UnfinishedEntry) error {
	start := entry.Start.Format(time.RFC3339)

	var count uint
	row := s.db.QueryRow(`select count(1) from clocking
		where title = ? and end is null`,
		entry.Title)
	if err := row.Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return ErrOngoingExists
	}

	var exists uint
	row = s.db.QueryRow(`select count(1) from clocking
		where title = ? and start = ?`,
		entry.Title,
		start)
	if err := row.Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		return ErrDuplicateEntry
	}

	_, err := s.db.Exec(`INSERT INTO clocking (title, start, notes)
	VALUES(?,?,?)`,
		entry.Title,
		start,
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

func (s *Sqlite) FinishLatest(notes string) (string, error) {
	row := s.db.QueryRow(`update clocking
		set end = ?, notes = IFNULL(notes, '')||?
		where id in (
			select max(id) from clocking
			where end is null
		) returning title`,
		time.Now().UTC().Format(time.RFC3339),
		notes)

	var title string
	if err := row.Scan(&title); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		} else {
			return "", err
		}
	}

	return title, nil
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
