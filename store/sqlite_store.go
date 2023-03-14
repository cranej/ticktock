package store

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"strings"
	"time"
)

type sqlite struct {
	db *sql.DB
}

func (s *sqlite) Start(entry *UnfinishedEntry) error {
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

func (s *sqlite) StartTitle(title, notes string) error {
	return s.Start(&UnfinishedEntry{
		Title: title,
		Start: time.Now().UTC(),
		Notes: notes,
	})
}

func (s *sqlite) Finish(notes string) (string, error) {
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

func (s *sqlite) RecentTitles(limit uint8) ([]string, error) {
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

func (s *sqlite) Ongoing() (*UnfinishedEntry, error) {
	row := s.db.QueryRow(`SELECT title, start, notes
		from clocking
		where end is null`)

	var title, start, notes string
	if err := row.Scan(&title, &start, &notes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	startTime, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return nil, err
	}

	return &UnfinishedEntry{title, startTime, notes}, nil
}

func (s *sqlite) LastFinished(title string) (*FinishedEntry, error) {
	row := s.db.QueryRow(`SELECT title, start, end, notes
		FROM clocking
		WHERE id in (
			SELECT max(id) FROM clocking
			WHERE title = ? and end IS NOT NULL
		)`, title)

	var title_, start, end, notes string
	if err := row.Scan(&title_, &start, &end, &notes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		} else {
			return nil, err
		}
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

var errTimeShouldBeUTC = errors.New("parameters should be in UTC")

func (s *sqlite) Finished(start, end time.Time, titles []string) ([]FinishedEntry, error) {
	_, soffset := start.Zone()
	_, eoffset := end.Zone()
	if soffset != 0 || eoffset != 0 {
		return nil, errTimeShouldBeUTC
	}

	query := `select title, start, end, notes
		from clocking
		where end is not null
		and start >= ? and start <= ?
		%s
		order by start`
	params := []any{start.Format(time.RFC3339), end.Format(time.RFC3339)}

	if len(titles) > 0 {
		marks := make([]string, 0, len(titles))
		for _, t := range titles {
			marks = append(marks, "?")
			params = append(params, t)
		}
		query = fmt.Sprintf(query, "and title in ("+strings.Join(marks, ",")+")")
	} else {
		query = fmt.Sprintf(query, "")
	}

	rows, err := s.db.Query(query, params...)
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

func newSqlite(db string) (sqlite, error) {
	pool, err := sql.Open("sqlite3", db)
	if err != nil {
		return sqlite{}, err
	}

	initSql := `CREATE TABLE IF NOT EXISTS clocking (
                id INTEGER PRIMARY KEY,
                title TEXT NOT NULL,
                start TEXT NOT NULL,
                end TEXT NULL,
                notes TEXT NULL
             )`

	if _, err := pool.Exec(initSql); err != nil {
		return sqlite{}, err
	}

	return sqlite{pool}, nil
}
