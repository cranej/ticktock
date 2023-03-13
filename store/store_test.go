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
	entry := UnfinishedEntry{
		Title: "test title",
		Start: time.Now().UTC(),
		Notes: "test notes",
	}

	if err := ss.Start(&entry); err != nil {
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
		t.Fatal("Expect 1 row but got nothing")
	}

	if err := rows.Scan(&title, &notes, &start, &end); err != nil {
		t.Fatal(err)
	}

	if rows.Next() {
		t.Fatal("Does not expect more than 1 rows")
	}

	if title != entry.Title || notes != entry.Notes ||
		start != entry.Start.Format(time.RFC3339) || end.Valid {
		t.Fatalf("Values does not match, title: %s, start: %s, notes: %s, end: %v",
			title, start, notes, end)
	}

}

func TestStoreStartOngoingExists(t *testing.T) {
	ss := assertStoreSetup(t)
	entry := UnfinishedEntry{
		Title: "test title",
		Start: time.Now().UTC(),
		Notes: "test notes",
	}

	if err := ss.Start(&entry); err != nil {
		t.Fatal(err)
	}

	entry = UnfinishedEntry{
		Title: "test new title",
		Start: time.Now().UTC(),
		Notes: "test new notes",
	}

	err := ss.Start(&entry)
	if err == nil || !errors.Is(err, ErrOngoingExists) {
		t.Fatalf("Expect ErrOngoingExists, got: %v", err)
	}
}

func TestStoreStartDuplicate(t *testing.T) {
	ss := assertStoreSetup(t)
	entry := UnfinishedEntry{
		Title: "test title",
		Start: time.Now().UTC(),
		Notes: "test notes",
	}

	if err := ss.Start(&entry); err != nil {
		t.Fatal(err)
	}

	if _, err := ss.Finish(""); err != nil {
		t.Fatal(err)
	}

	err := ss.Start(&entry)
	if err == nil || !errors.Is(err, ErrDuplicateEntry) {
		t.Fatalf("Expect ErrDuplicateEntry, got: %v", err)
	}
}

func TestFinishNoUnfinishedEntry(t *testing.T) {
	ss := assertStoreSetup(t)
	title, err := ss.Finish("")

	if err != nil || title != "" {
		t.Fatalf("Should return (\"\", nil), but got: (%s, %v)", title, err)
	}
}

func TestNoOngingEntry(t *testing.T) {
	ss := assertStoreSetup(t)
	entry, err := ss.Ongoing()

	if err != nil || entry != nil {
		t.Fatalf("Should return (nil, nil), but got: (%v, %v)", entry, err)
	}
}

func TestNoLastFinished(t *testing.T) {
	ss := assertStoreSetup(t)
	entry, err := ss.LastFinished("test")

	if err != nil || entry != nil {
		t.Fatalf("Should return (nil, nil), but got: (%v, %v)", entry, err)
	}
}

func TestFinishedQueryByNoneUTCTime(t *testing.T) {
	ss := assertStoreSetup(t)

	loc := time.FixedZone("UTC-8", -8*60*60)
	start := time.Date(2009, time.November, 10, 23, 0, 0, 0, loc)
	end := time.Date(2009, time.November, 11, 23, 0, 0, 0, loc)

	entries, err := ss.Finished(start, end.UTC())
	if !errors.Is(err, errTimeShouldBeUTC) || entries != nil {
		t.Fatal("Should fail on non UTC time query.")
	}

	entries, err = ss.Finished(start.UTC(), end)
	if !errors.Is(err, errTimeShouldBeUTC) || entries != nil {
		t.Fatal("Should fail on non UTC time query.")
	}

	entries, err = ss.Finished(start, end)
	if !errors.Is(err, errTimeShouldBeUTC) || entries != nil {
		t.Fatal("Should fail on non UTC time query.")
	}
}
