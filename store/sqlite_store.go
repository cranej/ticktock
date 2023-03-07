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

func (s *Sqlite) Start(entry *UnfinishedEntry) error {
	start := entry.Start.Format(time.RFC3339)

	var count uint
	row := s.db.QueryRow(`select count(1) from clocking
		where end is null`,
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
	err := row.Scan(&title)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}

	return title, err
}

func (s *Sqlite) RecentTitles(limit uint8) ([]string, error) {
	rows, err := s.db.Query(`SELECT title, max(start)
		FROM clocking
		where end is not null
		group by title
		order by max(start) desc limit ?`,
		limit)
	if err != nil {
		return nil, err
	}

	titles := make([]string, 0)
	for rows.Next() {
		var title, start string
		if err := rows.Scan(&title, &start); err != nil {
			return nil, err
		}

		titles = append(titles, title)
	}

	return titles, nil
}

func (s *Sqlite) Ongoing() (string, time.Duration, error) {
	row := s.db.QueryRow(`SELECT title, start
		from clocking
		where end is null`)

	dur0 := time.Duration(0)

	var title, start string
	if err := row.Scan(&title, &start); err != nil {
		return "", dur0, err
	}

	startTime, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return "", dur0, err
	}

	return title, time.Now().Sub(startTime), nil
}

func (s *Sqlite) LastFinished(title string) (*FinishedEntry, error) {
	row := s.db.QueryRow(`SELECT title, start, end, notes
		FROM clocking
		WHERE id in (
			SELECT max(id) FROM clocking
			WHERE title = ? and end IS NOT NULL
		)`, title)

	var title_, start, end, notes string
	if err := row.Scan(&title_, &start, &end, &notes); err != nil {
		return nil, err
	}

	startTime, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return nil, err
	}
	endTime, err := time.Parse(time.RFC3339, end)
	if err != nil {
		return nil, err
	}

	result := FinishedEntry{&UnfinishedEntry{title_, startTime, notes}, endTime}
	return &result, nil
}

func (s *Sqlite) Finished(start, end time.Time) ([]FinishedEntry, error) {
	_, soffset := start.Zone()
	_, eoffset := end.Zone()
	if soffset != 0 || eoffset != 0 {
		return nil, errors.New("Parameters should be in UTC.")
	}

	rows, err := s.db.Query(`select title, start, end, notes
		from clocking
		where end is not null
		and start >= ? and start <= ?
		order by start`,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
	)

	if err != nil {
		return nil, err
	}

	entries := make([]FinishedEntry, 0)
	for rows.Next() {
		var title, start, end, notes string
		if err := rows.Scan(&title, &start, &end, &notes); err != nil {
			return nil, err
		}

		startTime, err := time.Parse(time.RFC3339, start)
		if err != nil {
			return nil, err
		}

		endTime, err := time.Parse(time.RFC3339, end)
		if err != nil {
			return nil, err
		}

		entries = append(entries, FinishedEntry{
			&UnfinishedEntry{title, startTime, notes},
			endTime,
		})
	}

	return entries, nil
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
