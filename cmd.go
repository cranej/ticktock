package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/cranej/ticktock/server"
	"github.com/cranej/ticktock/store"
	"github.com/cranej/ticktock/utils"
	"github.com/cranej/ticktock/view"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"strconv"
	"strings"
	"time"
)

type StartCmd struct {
	Wait  bool     `short:"w" help:"If set, wait for notes input until Ctrl-D, then close the activity"`
	Title string   `arg:"" optional:"" name:"title" help:"Title of the activity. Choose from recent titles interactively if not given"`
	Notes []string `help:"Notes of the activity, each input as a line. If a single '-' is given, read from stdin"`
}

func (c *StartCmd) Run(ss store.Store) error {
	title, err := chooseTitleAsNeed(c.Title, ss)
	if err != nil {
		return err
	}

	notes, err := getNotes(c.Notes)
	if err != nil {
		return err
	}

	if err := ss.StartTitle(title, notes); err != nil {
		return err
	}
	fmt.Printf("(Started: %s)\n", c.Title)

	if c.Wait {
		fmt.Println("Waiting for notes input, Ctrl-D ends the input and close the activity:")
		notes, err := readToEOF()
		if err != nil {
			return fmt.Errorf("failed to read notes: %w, activity is not closed", err)
		}

		r, err := ss.CloseActivity(notes)
		if err != nil {
			return err
		}

		fmt.Printf("(Closed: %s)\n", r)
		return nil
	} else {
		return nil
	}
}

type CloseCmd struct {
	Notes []string `help:"Notes to appends, each input as a line. If a single '-' is given, read from stdin"`
}

func (c *CloseCmd) Run(ss store.Store) error {
	notes, err := getNotes(c.Notes)
	if err != nil {
		return err
	}

	r, err := ss.CloseActivity(notes)
	if err != nil {
		return err
	}

	if len(r) != 0 {
		fmt.Printf("(Closed: %s)\n", r)
	} else {
		fmt.Println("(NothingToClose)")
	}
	return nil
}

type TitlesCmd struct {
	Limit uint8 `short:"n" default:"5" help:"Number of titles to display, default 5"`
	Index bool  `short:"i" help:"If set, prefix titles with index starts from 1"`
}

func (c *TitlesCmd) Run(ss store.Store) error {
	limit := uint8(5)
	if c.Limit > 0 {
		limit = c.Limit
	}

	titles, err := ss.RecentTitles(limit)
	if err != nil {
		return err
	}

	var f func(int, string)
	if c.Index {
		f = func(i int, title string) { fmt.Printf("%d: %s\n", i+1, title) }
	} else {
		f = func(i int, title string) { fmt.Println(title) }
	}

	for i, t := range titles {
		f(i, t)
	}

	return nil
}

type OngoingCmd struct {
	Idle bool `short:"d" help:"If no ongoing activity found, output the idle time since latest closed activity's end time"`
}

func reportIdle(ss store.Store) (bool, error) {
	activity, err := ss.LastClosed("")
	if err != nil {
		return false, err
	}
	if activity == nil {
		return false, nil
	}

	now := time.Now()
	dayStart, dayEnd := utils.DayStartEnd(time.Now())

	// only report idle if the activity is on today and current time is not after dayEnd
	if activity.End.After(dayStart) && now.Before(dayEnd) {
		fmt.Printf("Idle: %.0f minutes\n", time.Since(activity.End).Minutes())
		return true, nil
	} else {
		return false, nil
	}
}

func (c *OngoingCmd) Run(ss store.Store) error {
	activity, err := ss.Ongoing()
	if err != nil {
		return err
	}

	if activity == nil {
		if c.Idle {
			r, err := reportIdle(ss)
			if err != nil {
				return err
			}
			if r {
				return nil
			}
		}

		fmt.Println("No ongoing activity.")
		return nil
	}

	duration := time.Since(activity.Start)
	fmt.Printf("%s: %.0f minutes\n", activity.Title, duration.Minutes())
	return nil
}

type LastCmd struct {
	Title string `arg:"" optional:"" help:"Title of the activity. Choose interactively if not given"`
}

func (c *LastCmd) Run(ss store.Store) error {
	title, err := chooseTitleAsNeed(c.Title, ss)
	if err != nil {
		return err
	}

	last, err := ss.LastClosed(title)
	if err != nil {
		return err
	}

	if last != nil {
		fmt.Println(last)
	} else {
		fmt.Println("No such activity.")
	}
	return nil
}

type ReportCmd struct {
	Type  string   `default:"summary" enum:"summary,detail,dist,efforts" help:"Type of the report to show, valid values are: summary, detail, dist (distribution), and efforts"`
	From  uint16   `short:"f" default:"0" help:"Show report of activities from '@today - From'. For example, '--from 1' shows report from yesterday 00:00:00"`
	To    uint16   `short:"t" default:"0" help:"Show report of activities to @today - To. For example, '--to 1' shows report to yesterday 23:59:59"`
	Week  bool     `short:"w" default:"false" help:"Show report from Monday 0:00:00, ignored if '--from/-f' or '--to/-t' is given"`
	Month bool     `short:"m" default:"false" help:"Show report from the 1st day 0:00:00 of this month , ignored if '--from/-f' or '--to/-t' or '--week/-w' is given"`
	Title []string `help:"filter by titles"`
	Tag   bool     `default:"false" help:"if set, --title 'book' queries all activities with title starts with 'book: ' (here, book is the tag of the activity). Also, activities will be aggregated by tag instead of by title"`
}

func (c *ReportCmd) Run(ss store.Store) error {
	now := time.Now()
	if c.From == 0 && c.To == 0 {
		if c.Week {
			// Weeks start from Monday
			c.From = uint16((now.Weekday() + 7 - 1) % 7)
		} else if c.Month {
			c.From = uint16(now.Day() - 1)
		}
	}
	start := time.Date(now.Year(), now.Month(), now.Day()-int(c.From), 0, 0, 0, 0, time.Local).UTC()
	end := time.Date(now.Year(), now.Month(), now.Day()-int(c.To), 23, 59, 59, 0, time.Local).UTC()

	var arg *store.QueryArg
	if c.Tag {
		arg = store.NewTagArg(c.Title)
	} else {
		arg = store.NewTitleArg(c.Title)
	}
	activities, err := ss.Closed(start, end, arg)
	if err != nil {
		return err
	}

	var keyF func(*store.ClosedActivity) string
	if c.Tag {
		keyF = (*store.ClosedActivity).Tag
	}
	view, err := view.Render(activities, c.Type, keyF)
	if err != nil {
		return err
	}

	if view != "" {
		fmt.Println(view)
	}
	return nil
}

type ServerCmd struct {
	Addr string `arg:"" help:"Address to which the server listens"`
}

func (c *ServerCmd) Run(ss store.Store) error {
	env := server.Env{Store: ss}
	return env.Run(c.Addr)
}

type AddCmd struct {
	Title string   `arg:"" optional:"" name:"title" help:"Title of the activity. Choose from recent titles interactively if not given"`
	Start string   `required:"" help:"Start time of activity, accpets 'HH:mm', 'dd HH:mm', 'MM-dd HH:mm' or 'yyyy-MM-dd HH:mm'"`
	End   string   `required:"" help:"End time of activity, accepts the same formats as Start"`
	Notes []string `help:"Notes of the activity, each input as a line. If a single '-' is given, read from stdin"`
}

func (c *AddCmd) Run(ss store.Store) error {
	title, err := chooseTitleAsNeed(c.Title, ss)
	if err != nil {
		return err
	}

	notes, err := getNotes(c.Notes)
	if err != nil {
		return err
	}

	start, err := parseImportTime(c.Start)
	if err != nil {
		return err
	}

	end, err := parseImportTime(c.End)
	if err != nil {
		return err
	}

	activity := store.ClosedActivity{&store.OpenActivity{title, start.UTC(), notes}, end.UTC()}
	return ss.Add(&activity)
}

// helper functions
var errCannotReadIndex error = errors.New("cannot read index")
var errInvalidIndex error = errors.New("invalid index")
var errNothingToChoose error = errors.New("candidates is empty")

const DEFAULT_LIMIT uint8 = 5

func chooseTitleAsNeed(title string, ss store.Store) (string, error) {
	if title != "" {
		return title, nil
	}

	titles, err := ss.RecentTitles(DEFAULT_LIMIT)
	if err != nil {
		return "", nil
	}

	return chooseString(titles)
}

func readToEOF() (string, error) {
	var b strings.Builder
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Fprintln(&b, scanner.Text())
	}

	return b.String(), scanner.Err()
}

func chooseString(candidates []string) (string, error) {
	if len(candidates) == 0 {
		return "", errNothingToChoose
	}

	for i, s := range candidates {
		fmt.Printf("%d: %s\n", i+1, s)
	}
	fmt.Print("Choose index (default 1): ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", errCannotReadIndex
	}

	var i int
	var err error
	if scanner.Text() == "" {
		i = 1
	} else {
		if i, err = strconv.Atoi(scanner.Text()); err != nil {
			return "", err
		}
		if i > len(candidates) || i < 1 {
			return "", errInvalidIndex
		}
	}

	return candidates[i-1], nil
}

func getNotes(input []string) (string, error) {
	if len(input) == 1 && input[0] == "-" {
		fmt.Println("Input notes (end with Ctrl-D): ")
		return readToEOF()
	} else {
		return strings.Join(input, "\n"), nil
	}
}

const IMPORT_FULL_DT string = "2006-01-02 15:04"

// parseImportTime accepts time in all the following formats:
//
//	yyyy-MM-dd HH:mm
//	MM-dd HH:mm
//	dd HH:mm
//	HH:mm
func parseImportTime(value string) (time.Time, error) {
	now := time.Now()
	y, m, d := now.Year(), now.Month(), now.Day()
	var padded string
	switch len(value) {
	case 5:
		padded = fmt.Sprintf("%d-%02d-%02d %s", y, m, d, value)
	case 8:
		padded = fmt.Sprintf("%d-%02d-%s", y, m, value)
	case 11:
		padded = fmt.Sprintf("%d-%s", y, value)
	case 16:
		padded = value
	default:
		return time.UnixMicro(0), errors.New("Unknown time format: " + value)
	}

	return time.ParseInLocation(IMPORT_FULL_DT, padded, time.Local)
}
