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

	if _, err := ss.FinishLatest(""); err != nil {
		t.Fatal(err)
	}

	err := ss.Start(&entry)
	if err == nil || !errors.Is(err, ErrDuplicateEntry) {
		t.Fatalf("Expect ErrDuplicateEntry, got: %v", err)
	}
}
