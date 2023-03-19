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
		strings.TrimRight(notes.String(), "\n"))
}

func (entry *FinishedEntry) Tag() string {
	return strings.SplitN(entry.Title, ": ", 2)[0]
}

var ErrOngoingExists = errors.New("ongoing entry exists")
var ErrDuplicateEntry = errors.New("entry already started")

type QueryArg struct {
	values []string
	asTag  bool
}

func NewTitleArg(titles []string) *QueryArg {
	if titles == nil {
		return nil
	}

	return &QueryArg{
		values: titles,
		asTag:  false,
	}
}

func NewTagArg(tags []string) *QueryArg {
	if tags == nil {
		return nil
	}

	return &QueryArg{
		values: tags,
		asTag:  true,
	}
}

func (q *QueryArg) Empty() bool {
	return q == nil || len(q.values) == 0
}

func (q *QueryArg) Values() []string {
	if q == nil {
		return nil
	} else {
		return q.values
	}
}

func (q *QueryArg) IsTag() bool {
	return q != nil && q.asTag
}

type Store interface {
	// Start an entry.
	//  1. No new entry allowed if there is already an ongoing (unfinished) entry exists.
	//  2. Entry considered as duplicated and is not allowed to start,
	//     if there is already an entry with the same Title and Start.
	Start(*UnfinishedEntry) error

	// StartTitle starts an entry with given title and notes, and 'now' as Start.
	StartTitle(title, note string) error

	// Finish finishes the unfinished entry (if any).
	// If there was one, return it's title.
	// If no unfinished entry to finish, return empty string. This case is not treated as error.
	Finish(notes string) (string, error)

	// RecentTitles returns at most 'limit' number of distinct titles of recent finished entries.
	RecentTitles(limit uint8) ([]string, error)

	// Ongoing returns the ongoing entry (if any), otherwise return nil.
	Ongoing() (*UnfinishedEntry, error)

	// LastFinished returns the finished entry with the latest Start of given title, if any. Otherwise return nil.
	LastFinished(title string) (*FinishedEntry, error)

	// Finished queries entries with condition 'Start >= queryStart and Start <= queryEnd'.
	// Both queryStart and queryEnd must be UTC time
	// If filter is not nil:
	//   if filter is title filter, only returns entries with 'title in filter.values'.
	//   if filter is tag filter, returns entries with 'FinishedEntry.Tag() in filter.values'.
	Finished(queryStart, queryEnd time.Time, filter *QueryArg) ([]FinishedEntry, error)
}

func NewSqliteStore(db string) (Store, error) {
	s, err := newSqlite(db)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// View implementation. Views do not modify input entries.
func View(entries []FinishedEntry, viewType string, keyF func(*FinishedEntry) string) (string, error) {
	if keyF == nil {
		keyF = func(e *FinishedEntry) string { return e.Title }
	}
	switch viewType {
	case "summary":
		summary := NewSummary(entries, keyF)
		return summary.String(), nil
	case "detail":
		detail := NewDetail(entries, keyF)
		return detail.String(), nil
	case "dist":
		dist := NewDist(entries, keyF)
		return dist.String(), nil
	case "efforts":
		efforts := NewEfforts(entries, keyF)
		return efforts.String(), nil
	default:
		return "", fmt.Errorf("unknown viewType: %s", viewType)
	}
}

var round time.Duration = time.Duration(time.Minute)

// durS return string represention of d as "72h3m"
func durS(d time.Duration) string {
	return strings.TrimSuffix(d.Round(round).String(), "0s")
}

type SummaryView map[string]map[string]time.Duration

func NewSummary(entries []FinishedEntry, keyF func(*FinishedEntry) string) SummaryView {
	summary := make(SummaryView)

	for _, e := range entries {
		day := e.Start.Local().Format(time.DateOnly)
		dayMap, ok := summary[day]
		if !ok {
			dayMap = make(map[string]time.Duration)
			summary[day] = dayMap
		}

		key := keyF(&e)
		dur := dayMap[key]
		dayMap[key] = dur + e.End.Sub(e.Start)
	}

	return summary
}

func (s SummaryView) String() string {
	var b strings.Builder
	for day, dayMap := range s {
		fmt.Fprintln(&b, day)

		var dayDur time.Duration
		for title, dur := range dayMap {
			fmt.Fprintf(&b, "  %s: %s\n", title, durS(dur))
			dayDur += dur
		}

		fmt.Fprintf(&b, "(Total): %s\n\n", durS(dayDur))
	}

	return strings.TrimRight(b.String(), "\n")
}

type DetailView map[string][]*FinishedEntry

func NewDetail(entries []FinishedEntry, keyF func(*FinishedEntry) string) DetailView {
	detail := make(DetailView)

	for i, e := range entries {
		key := keyF(&e)
		entrySlice, ok := detail[key]
		if !ok {
			entrySlice = make([]*FinishedEntry, 0)
		}

		entrySlice = append(entrySlice, &entries[i])
		detail[key] = entrySlice
	}

	return detail
}

func (d DetailView) String() string {
	layout := "2006-01-02 Mon 15:04"
	short := "15:04"
	var b strings.Builder
	for title, entrySlice := range d {
		fmt.Fprintln(&b, title)

		for _, e := range entrySlice {
			fmt.Fprintf(&b, "  %s ~ %s | %s\n",
				e.Start.Local().Format(layout),
				e.End.Local().Format(short),
				durS(e.End.Sub(e.Start)))
		}

		fmt.Fprintln(&b)
	}

	return strings.TrimRight(b.String(), "\n")
}

type EffortsView map[string]time.Duration

func NewEfforts(entries []FinishedEntry, keyF func(*FinishedEntry) string) EffortsView {
	efforts := make(EffortsView)
	for _, e := range entries {
		key := keyF(&e)
		efforts[key] = efforts[key] + e.End.Sub(e.Start)
	}

	return efforts
}

func (eff EffortsView) String() string {
	var b strings.Builder
	for title, dur := range eff {
		fmt.Fprintf(&b, "%s: %s\n", title, durS(dur))
	}

	return strings.TrimRight(b.String(), "\n")
}

type DistView map[string][]*FinishedEntry

const IDLE_TITLE string = "<idle>"

func NewDist(entries []FinishedEntry, keyF func(*FinishedEntry) string) DistView {
	dist := make(DistView)

	for _, e := range entries {
		day := e.Start.Local().Format(time.DateOnly)
		daySlice, ok := dist[day]
		if !ok {
			daySlice = make([]*FinishedEntry, 0, 1)
		}

		// Do not modify input entries here
		daySlice = append(daySlice, &FinishedEntry{
			&UnfinishedEntry{Title: keyF(&e), Start: e.Start, Notes: ""},
			e.End,
		})
		dist[day] = daySlice
	}

	for day, daySlice := range dist {
		dayTime, _ := time.ParseInLocation(time.DateOnly, day, time.Local)
		dayStart := time.Date(dayTime.Year(), dayTime.Month(), dayTime.Day(),
			8, 30, 0, 0, time.Local)
		dayEnd := time.Date(dayTime.Year(), dayTime.Month(), dayTime.Day(),
			21, 0, 0, 0, time.Local)

		dist[day] = fillIdles(daySlice, dayStart, dayEnd)
	}

	return dist
}

func (d DistView) String() string {
	var b strings.Builder
	for day, daySlice := range d {
		fmt.Fprintln(&b, day)

		var idleDur time.Duration
		for _, e := range daySlice {
			dur := e.End.Sub(e.Start)
			if e.Title == IDLE_TITLE {
				idleDur += dur
			}
			fmt.Fprintf(&b, "  %s ~ %s | %-7s | %s\n",
				e.Start.Local().Format(time.TimeOnly),
				e.End.Local().Format(time.TimeOnly),
				durS(dur),
				e.Title)
		}

		fmt.Fprintf(&b, "(Idle: %s)\n\n", durS(idleDur))
	}

	return strings.TrimRight(b.String(), "\n")
}

func fillIdles(entries []*FinishedEntry, start, end time.Time) []*FinishedEntry {
	result := make([]*FinishedEntry, 0, len(entries))
	for i, d := range entries {
		if d.Start.After(start) {
			result = append(result, &FinishedEntry{
				&UnfinishedEntry{Title: IDLE_TITLE, Start: start, Notes: ""},
				d.Start,
			})
		}

		result = append(result, entries[i])
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
