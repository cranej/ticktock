package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type sqlite struct {
	db *sql.DB
}

func (s *sqlite) Start(activity *OpenActivity) error {
	start := activity.Start.Format(time.RFC3339)

	var count uint
	row := s.db.QueryRow(`select count(1) from clocking
		where end is null`,
		activity.Title)
	if err := row.Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return ErrOngoingExists
	}

	var exists uint
	row = s.db.QueryRow(`select count(1) from clocking
		where title = ? and start = ?`,
		activity.Title,
		start)
	if err := row.Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		return ErrDuplicateActivity
	}

	_, err := s.db.Exec(`INSERT INTO clocking (title, start, notes)
	VALUES(?,?,?)`,
		activity.Title,
		start,
		activity.Notes)

	return err
}

func (s *sqlite) StartTitle(title, notes string) error {
	return s.Start(&OpenActivity{
		Title: title,
		Start: time.Now().UTC(),
		Notes: notes,
	})
}

func (s *sqlite) CloseActivity(notes string) (string, error) {
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

func (s *sqlite) Ongoing() (*OpenActivity, error) {
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

	return &OpenActivity{title, startTime, notes}, nil
}

func (s *sqlite) LastClosed(title string) (*ClosedActivity, error) {
	query := `SELECT title, start, end, notes
		FROM clocking
		WHERE id in (
			SELECT max(id) FROM clocking
			WHERE %s end IS NOT NULL)`
	params := []any{}
	if title == "" {
		query = fmt.Sprintf(query, "")
	} else {
		query = fmt.Sprintf(query, "title = ? and ")
		params = append(params, title)
	}

	row := s.db.QueryRow(query, params...)

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

	result := ClosedActivity{&OpenActivity{title_, startTime, notes}, endTime}
	return &result, nil
}

var errTimeShouldBeUTC = errors.New("parameters should be in UTC")

func (s *sqlite) Closed(start, end time.Time, filter *QueryArg) ([]ClosedActivity, error) {
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

	if filter.Empty() {
		query = fmt.Sprintf(query, "")
	} else {
		marks := make([]string, 0, len(filter.Values()))
		var mark string
		var valueF func(string) string
		if filter.IsTag() {
			mark = `title like ?`
			valueF = func(t string) string { return t + ": %" }
		} else {
			mark = "title = ?"
			valueF = func(t string) string { return t }
		}
		for _, t := range filter.Values() {
			marks = append(marks, mark)
			params = append(params, valueF(t))
		}
		query = fmt.Sprintf(query, "and ("+strings.Join(marks, " or ")+")")
	}

	rows, err := s.db.Query(query, params...)
	if err != nil {
		return nil, err
	}

	activities := make([]ClosedActivity, 0)
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

		activities = append(activities, ClosedActivity{
			&OpenActivity{title, startTime, notes},
			endTime,
		})
	}

	return activities, nil
}

func (s *sqlite) Add(activity *ClosedActivity) error {
	start := activity.Start.Format(time.RFC3339)
	var exists uint
	row := s.db.QueryRow(`select count(1) from clocking
		where title = ? and start = ?`,
		activity.Title,
		start)
	if err := row.Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		return ErrDuplicateActivity
	}

	end := activity.End.Format(time.RFC3339)
	_, err := s.db.Exec(`INSERT INTO clocking (title, start, end, notes)
	VALUES(?,?,?,?)`,
		activity.Title,
		start,
		end,
		activity.Notes)

	return err
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
