package store

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"testing"
	"time"
)

var dbConn = "file::memory:?cache=shared"

func assertStoreSetup(t *testing.T) Store {
	t.Helper()

	ss, err := NewSqliteStore(dbConn)
	if err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite3", dbConn)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec("delete from clocking"); err != nil {
		t.Fatalf("Error while cleanup db: %v", err)
	}

	return ss
}

func TestStoreStart(t *testing.T) {
	ss := assertStoreSetup(t)
	activity := OpenActivity{
		Title: "test title",
		Start: time.Now().UTC(),
		Notes: "test notes",
	}

	if err := ss.Start(&activity); err != nil {
		t.Fatal(err)
	}

	var title, notes, start string
	var end sql.NullString

	db, err := sql.Open("sqlite3", dbConn)
	if err != nil {
		t.Fatal(err)
	}

	rows, err := db.Query(`select title, notes, start, end from clocking`)

	if err != nil {
		t.Fatal(err)
	}

	if !rows.Next() {
		t.Fatal("Expects 1 row but got nothing")
	}

	if err := rows.Scan(&title, &notes, &start, &end); err != nil {
		t.Fatal(err)
	}

	if rows.Next() {
		t.Fatal("Does not expect more than 1 rows")
	}

	if title != activity.Title || notes != activity.Notes ||
		start != activity.Start.Format(time.RFC3339) || end.Valid {
		t.Fatalf("Values do not match, title: %s, start: %s, notes: %s, end: %v",
			title, start, notes, end)
	}

}

func TestStoreStartOngoingExists(t *testing.T) {
	ss := assertStoreSetup(t)
	activity := OpenActivity{
		Title: "test title",
		Start: time.Now().UTC(),
		Notes: "test notes",
	}

	if err := ss.Start(&activity); err != nil {
		t.Fatal(err)
	}

	activity = OpenActivity{
		Title: "test new title",
		Start: time.Now().UTC(),
		Notes: "test new notes",
	}

	err := ss.Start(&activity)
	if err == nil || !errors.Is(err, ErrOngoingExists) {
		t.Fatalf("Expects ErrOngoingExists, got: %v", err)
	}
}

func TestStoreStartDuplicate(t *testing.T) {
	ss := assertStoreSetup(t)
	activity := OpenActivity{
		Title: "test title",
		Start: time.Now().UTC(),
		Notes: "test notes",
	}

	if err := ss.Start(&activity); err != nil {
		t.Fatal(err)
	}

	if _, err := ss.CloseActivity(""); err != nil {
		t.Fatal(err)
	}

	err := ss.Start(&activity)
	if err == nil || !errors.Is(err, ErrDuplicateActivity) {
		t.Fatalf("Expects ErrDuplicateActivity, got: %v", err)
	}
}

func TestCloseActivityNoOpenActivity(t *testing.T) {
	ss := assertStoreSetup(t)
	title, err := ss.CloseActivity("")

	if err != nil || title != "" {
		t.Fatalf("Should return (\"\", nil), but got: (%s, %v)", title, err)
	}
}

func TestNoOngingActivity(t *testing.T) {
	ss := assertStoreSetup(t)
	activity, err := ss.Ongoing()

	if err != nil || activity != nil {
		t.Fatalf("Should return (nil, nil), but got: (%v, %v)", activity, err)
	}
}

func TestNoLastClosed(t *testing.T) {
	ss := assertStoreSetup(t)
	activity, err := ss.LastClosed("test")

	if err != nil || activity != nil {
		t.Fatalf("Should return (nil, nil), but got: (%v, %v)", activity, err)
	}
}

func TestClosedQueryByNoneUTCTime(t *testing.T) {
	ss := assertStoreSetup(t)

	loc := time.FixedZone("UTC-8", -8*60*60)
	start := time.Date(2009, time.November, 10, 23, 0, 0, 0, loc)
	end := time.Date(2009, time.November, 11, 23, 0, 0, 0, loc)

	activities, err := ss.Closed(start, end.UTC(), nil)
	if !errors.Is(err, errTimeShouldBeUTC) || activities != nil {
		t.Fatal("Should fail on none UTC time query.")
	}

	activities, err = ss.Closed(start.UTC(), end, nil)
	if !errors.Is(err, errTimeShouldBeUTC) || activities != nil {
		t.Fatal("Should fail on none UTC time query.")
	}

	activities, err = ss.Closed(start, end, nil)
	if !errors.Is(err, errTimeShouldBeUTC) || activities != nil {
		t.Fatal("Should fail on none UTC time query.")
	}
}
