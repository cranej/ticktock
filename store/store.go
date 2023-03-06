package store

import (
	"errors"
	"fmt"
	"time"
	"strings"
)

type UnfinishedEntry struct {
	Title string
	Start time.Time
	Notes string
}

type FinishedEntry struct {
	*UnfinishedEntry
	End time.Time
}

func (entry *FinishedEntry) Format() string {
	var notes strings.Builder
	for _, s := range strings.Split(entry.Notes, "\n") {
		fmt.Fprintf(&notes, "    %s\n", s)
	}

	return fmt.Sprintf("%s\n%s ~ %s\n%s",
		entry.Title,
		entry.Start.Local().Format(time.DateTime),
		entry.End.Local().Format(time.DateTime),
		strings.TrimSuffix(notes.String(), "\n"))
}

var ErrOngoingExists = errors.New("Ongoing entry eixsts")
var ErrDuplicateEntry = errors.New("Entry already started")

type Store interface {
	Start(*UnfinishedEntry) error
	StartTitle(string, string) error
	FinishLatest(string) (string, error)
	RecentTitles(uint8) ([]string, error)
	Ongoing() (string, time.Duration, error)
	LastFinished(string) (*FinishedEntry, error)
}

func NewSqliteStore(db string) (Store, error) {
	s, err := newSqlite(db)
	if err != nil {
		return nil, err
	}

	return &s, nil
}
