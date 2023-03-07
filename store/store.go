package store

import (
	"errors"
	"fmt"
	"strings"
	"time"
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
	Finished(time.Time, time.Time) ([]FinishedEntry, error)
}

func NewSqliteStore(db string) (Store, error) {
	s, err := newSqlite(db)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

type SummaryView map[string]map[string]time.Duration

func NewSummary(entries []FinishedEntry) SummaryView {
	summary := make(SummaryView)

	for _, e := range entries {
		day := e.Start.Local().Format(time.DateOnly)
		dayMap, ok := summary[day]
		if !ok {
			dayMap = make(map[string]time.Duration)
			summary[day] = dayMap
		}

		dur := dayMap[e.Title]
		dayMap[e.Title] = dur + e.End.Sub(e.Start)
	}

	return summary
}

func (s SummaryView) String() string {
	r := time.Duration(time.Minute)
	var b strings.Builder
	for day, dayMap := range s {
		fmt.Fprintln(&b, day)

		for title, dur := range dayMap {
			fmt.Fprintf(&b, "  %s: %s\n", title, dur.Round(r))
		}

		fmt.Fprintln(&b)
	}

	return b.String()
}
