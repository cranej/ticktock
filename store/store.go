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
