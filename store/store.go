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

type DetailView map[string][]*FinishedEntry

func NewDetail(entries []FinishedEntry) DetailView {
	detail := make(DetailView)

	for i, e := range entries {
		entrySlice, ok := detail[e.Title]
		if !ok {
			entrySlice = make([]*FinishedEntry, 0)
		}

		entrySlice = append(entrySlice, &entries[i])
		detail[e.Title] = entrySlice
	}

	return detail
}

func (d DetailView) String() string {
	r := time.Duration(time.Minute)
	layout := "2006-01-02 Mon 15:04"
	var b strings.Builder
	for title, entrySlice := range d {
		fmt.Fprintln(&b, title)

		for _, e := range entrySlice {
			fmt.Fprintf(&b, "  %s ~ %s, %s\n",
				e.Start.Local().Format(layout),
				e.End.Local().Format(layout),
				e.End.Sub(e.Start).Round(r))
		}

		fmt.Fprintln(&b)
	}

	return b.String()
}

type DistView map[string][]*FinishedEntry

const IDLE_TITLE string = "<idle>"

func NewDist(entries []FinishedEntry) DistView {
	dist := make(DistView)

	for i, e := range entries {
		day := e.Start.Local().Format(time.DateOnly)
		daySlice, ok := dist[day]
		if !ok {
			daySlice = make([]*FinishedEntry, 0, 1)
		}

		daySlice = append(daySlice, &entries[i])
		dist[day] = daySlice
	}

	for day, daySlice := range dist {
		dayTime, _ := time.ParseInLocation(time.DateOnly, day, time.Local)
		dayStart := time.Date(dayTime.Year(), dayTime.Month(), dayTime.Day(),
			9, 0, 0, 0, time.Local)
		dayEnd := time.Date(dayTime.Year(), dayTime.Month(), dayTime.Day(),
			21, 0, 0, 0, time.Local)

		dist[day] = fillWithIdles(daySlice, dayStart, dayEnd)
	}

	return dist
}

func (d DistView) String() string {
	r := time.Duration(time.Minute)
	var b strings.Builder
	for day, daySlice := range d {
		fmt.Fprintln(&b, day)

		for _, e := range daySlice {
			fmt.Fprintf(&b, "  %s ~ %s, %s: %s\n",
				e.Start.Local().Format(time.TimeOnly),
				e.End.Local().Format(time.TimeOnly),
				e.End.Sub(e.Start).Round(r),
				e.Title)
		}
	}

	return b.String()
}

func fillWithIdles(durs []*FinishedEntry, start, end time.Time) []*FinishedEntry {
	result := make([]*FinishedEntry, 0, len(durs))
	for i, d := range durs {
		if d.Start.After(start) {
			result = append(result, &FinishedEntry{
				&UnfinishedEntry{Title: IDLE_TITLE, Start: start, Notes: ""},
				d.Start,
			})
		}

		result = append(result, durs[i])
		start = d.End
	}

	if end.After(start) {
		result = append(result, &FinishedEntry{
			&UnfinishedEntry{Title: IDLE_TITLE, Start: start, Notes: ""},
			end,
		})
	}

	return result
}
