package view

import (
	"fmt"
	"github.com/cranej/ticktock/store"
	"strings"
	"time"
)

type KeyFunc func(*store.ClosedActivity) string
type Impl interface {
	String() string
}
type Creator func([]store.ClosedActivity, KeyFunc) Impl

var registry map[string]Creator = make(map[string]Creator)

func init() {
	registry["summary"] = NewSummary
	registry["detail"] = NewDetail
	registry["dist"] = NewDist
	registry["efforts"] = NewEfforts
}

func Register(viewType string, viewFunc Creator) error {
	_, ok := registry[viewType]
	if ok {
		return fmt.Errorf("view type %s already registered", viewType)
	}

	registry[viewType] = viewFunc
	return nil
}

func Render(activities []store.ClosedActivity, viewType string, keyF KeyFunc) (string, error) {
	if keyF == nil {
		keyF = func(e *store.ClosedActivity) string { return e.Title }
	}

	viewF, ok := registry[viewType]
	if !ok {
		return "", fmt.Errorf("unknown viewType %s", viewType)
	}

	return viewF(activities, keyF).String(), nil
}

var round time.Duration = time.Duration(time.Minute)

// durS return string represention of d as "72h3m"
func durS(d time.Duration) string {
	return strings.TrimSuffix(d.Round(round).String(), "0s")
}

type Summary map[string]map[string]time.Duration

func NewSummary(activities []store.ClosedActivity, keyF KeyFunc) Impl {
	summary := make(Summary)

	for _, e := range activities {
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

func (s Summary) String() string {
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

type Detail map[string][]*store.ClosedActivity

func NewDetail(activities []store.ClosedActivity, keyF KeyFunc) Impl {
	detail := make(Detail)

	for i, e := range activities {
		key := keyF(&e)
		slice, ok := detail[key]
		if !ok {
			slice = make([]*store.ClosedActivity, 0)
		}

		slice = append(slice, &activities[i])
		detail[key] = slice
	}

	return detail
}

func (d Detail) String() string {
	layout := "2006-01-02 Mon 15:04"
	short := "15:04"
	var b strings.Builder
	for title, activities := range d {
		fmt.Fprintln(&b, title)

		for _, e := range activities {
			fmt.Fprintf(&b, "  %s ~ %s | %s\n",
				e.Start.Local().Format(layout),
				e.End.Local().Format(short),
				durS(e.End.Sub(e.Start)))
		}

		fmt.Fprintln(&b)
	}

	return strings.TrimRight(b.String(), "\n")
}

type Efforts map[string]time.Duration

func NewEfforts(activities []store.ClosedActivity, keyF KeyFunc) Impl {
	efforts := make(Efforts)
	for _, e := range activities {
		key := keyF(&e)
		efforts[key] = efforts[key] + e.End.Sub(e.Start)
	}

	return efforts
}

func (eff Efforts) String() string {
	var b strings.Builder
	for title, dur := range eff {
		fmt.Fprintf(&b, "%s: %s\n", title, durS(dur))
	}

	return strings.TrimRight(b.String(), "\n")
}

type Distribution map[string][]*store.ClosedActivity

const IDLE_TITLE string = "<idle>"

func NewDist(activities []store.ClosedActivity, keyF KeyFunc) Impl {
	dist := make(Distribution)

	for _, e := range activities {
		day := e.Start.Local().Format(time.DateOnly)
		daySlice, ok := dist[day]
		if !ok {
			daySlice = make([]*store.ClosedActivity, 0, 1)
		}

		// Do not modify input activities here
		daySlice = append(daySlice, &store.ClosedActivity{
			OpenActivity: &store.OpenActivity{Title: keyF(&e), Start: e.Start, Notes: ""},
			End:          e.End,
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

func (d Distribution) String() string {
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

func fillIdles(activities []*store.ClosedActivity, start, end time.Time) []*store.ClosedActivity {
	result := make([]*store.ClosedActivity, 0, len(activities))
	for i, d := range activities {
		// ignore idles less than 1 minute
		if d.Start.Sub(start) >= time.Minute {
			result = append(result, &store.ClosedActivity{
				OpenActivity: &store.OpenActivity{Title: IDLE_TITLE, Start: start, Notes: ""},
				End:          d.Start,
			})
		}

		result = append(result, activities[i])
		start = d.End
	}

	// for today's activities, make end no later than now
	now := time.Now()
	if end.After(now) {
		end = now
	}
	if end.Sub(start) >= time.Minute {
		result = append(result, &store.ClosedActivity{
			OpenActivity: &store.OpenActivity{Title: IDLE_TITLE, Start: start, Notes: ""},
			End:          end,
		})
	}

	return result
}
