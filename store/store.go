package store

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type OpenActivity struct {
	Title string
	Start time.Time
	Notes string
}

type ClosedActivity struct {
	*OpenActivity
	End time.Time
}

func (activity *ClosedActivity) String() string {
	var notes strings.Builder
	for _, s := range strings.Split(activity.Notes, "\n") {
		fmt.Fprintf(&notes, "    %s\n", s)
	}

	return fmt.Sprintf("%s\n%s ~ %s\n%s",
		activity.Title,
		activity.Start.Local().Format(time.DateTime),
		activity.End.Local().Format(time.DateTime),
		strings.TrimRight(notes.String(), "\n"))
}

func (activity *ClosedActivity) Tag() string {
	return strings.SplitN(activity.Title, ": ", 2)[0]
}

var ErrOngoingExists = errors.New("ongoing activity exists")
var ErrDuplicateActivity = errors.New("activity already started")

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
	// Start an activity.
	//  1. No new activity allowed if there is already an open activity exists.
	//  2. activity considered as duplicated and is not allowed to start,
	//     if there is already an activity with the same Title and Start.
	Start(*OpenActivity) error

	// StartTitle starts an activity with given title and notes, and 'now' as Start.
	StartTitle(title, note string) error

	// CloseActivity closes the open activity (if any).
	// If there was one, return it's title.
	// If no open activity to close, return empty string. This case is not treated as error.
	CloseActivity(notes string) (string, error)

	// RecentTitles returns at most 'limit' number of distinct titles of recent closed activities.
	RecentTitles(limit uint8) ([]string, error)

	// Ongoing returns the open activity (if any), otherwise return nil.
	Ongoing() (*OpenActivity, error)

	// LastClosed returns the closed activity with the latest Start of given title, if any. Otherwise return nil.
	LastClosed(title string) (*ClosedActivity, error)

	// Closed queries activities with condition 'Start >= queryStart and Start <= queryEnd'.
	// Both queryStart and queryEnd must be UTC time
	// If filter is not nil:
	//   if filter is title filter, only returns activities with 'title in filter.values'.
	//   if filter is tag filter, returns activities with 'ClosedActivity.Tag() in filter.values'.
	Closed(queryStart, queryEnd time.Time, filter *QueryArg) ([]ClosedActivity, error)
}

func NewSqliteStore(db string) (Store, error) {
	s, err := newSqlite(db)
	if err != nil {
		return nil, err
	}

	return &s, nil
}
