package main

import (
	"bufio"
	"cranejin.com/ticktock/server"
	"cranejin.com/ticktock/store"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

var Cli struct {
	Db      string     `required:"" env:"TICKTOCK_DB" type:"path" help:"Path of the db file"`
	Start   StartCmd   `cmd:"" help:"Start a ticktock"`
	Finish  FinishCmd  `cmd:"" help:"Finish the ongoing ticktock"`
	Titles  TitlesCmd  `cmd:"" help:"Print recent finished titles"`
	Ongoing OngoingCmd `cmd:"" help:"Show currently ongoing ticktock"`
	Last    LastCmd    `cmd:"" help:"Show last finished ticktock details of title"`
	Report  ReportCmd  `cmd:"" help:"Show time usage report"`
	Server  ServerCmd  `cmd:"" help:"Start a server"`
}

type StartCmd struct {
	Wait  bool   `short:"w" help:"If set, wait for notes input until Ctrl-D, then finish the ticktock"`
	Title string `arg:"" optional:"" name:"title" help:"The title of the ticktock. Choose interactively if not given"`
	Notes string `help:"Notes of the ticktock"`
}

func (c *StartCmd) Run(ss store.Store) error {
	title, err := chooseTitleAsNeed(c.Title, ss)
	if err != nil {
		return err
	}

	if err := ss.StartTitle(title, c.Notes); err != nil {
		return err
	}
	fmt.Printf("(Started: %s)\n", c.Title)

	if c.Wait {
		fmt.Println("Waiting for notes input, Ctrl-D ends the input and finish the ticktock:")
		notes, err := readToEOF()
		if err != nil {
			return fmt.Errorf("failed to read notes: %w, ticktock not finished", err)
		}

		r, err := ss.Finish(notes)
		if err != nil {
			return err
		}

		fmt.Printf("(Finished: %s)\n", r)
		return nil
	} else {
		return nil
	}
}

type FinishCmd struct {
	Notes []string `help:"Notes to appends, each input as a line. If a single '-' is specified, read from stdin."`
}

func (c *FinishCmd) Run(ss store.Store) error {
	var notes string
	var err error
	if len(c.Notes) == 1 && c.Notes[0] == "-" {
		notes, err = readToEOF()
		if err != nil {
			return err
		}
	} else if len(c.Notes) > 1 {
		notes = strings.Join(c.Notes, "\n")
	}

	r, err := ss.Finish(notes)
	if err != nil {
		return err
	}

	if len(r) != 0 {
		fmt.Printf("(Finished: %s)\n", r)
	} else {
		fmt.Println("(NothingToFinish)")
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
}

func (c *OngoingCmd) Run(ss store.Store) error {
	entry, err := ss.Ongoing()
	if err != nil {
		return err
	}

	if entry == nil {
		fmt.Println("No ongoing entry.")
		return nil
	}

	duration := time.Since(entry.Start)
	fmt.Printf("%s\n%.0f minutes ago\n", entry.Title, duration.Minutes())
	return nil
}

type LastCmd struct {
	Title string `arg:"" optional:"" help:"Title of the ticktock. Choose interactively if not given"`
}

func (c *LastCmd) Run(ss store.Store) error {
	title, err := chooseTitleAsNeed(c.Title, ss)
	if err != nil {
		return err
	}

	last, err := ss.LastFinished(title)
	if err != nil {
		return err
	}

	fmt.Println(last.Format())
	return nil
}

type ReportCmd struct {
	Type  string   `default:"summary" enum:"summary,detail,dist,efforts" help:"Type of the report to show, valid values are: summary, detail, dist (distribution), and efforts"`
	From  uint16   `short:"f" default:"0" help:"Show report of ticktocks from '@today - From'. For example, '--from 1' shows report from yesterday 00:00:00"`
	To    uint16   `short:"t" default:"0" help:"Show report to @today - To. For example, '--to 1' shows report to yesterday 23:59:59"`
	Title []string `help:"filter by titles"`
}

func (c *ReportCmd) Run(ss store.Store) error {
	now := time.Now()
	queryStart := time.Date(now.Year(), now.Month(), now.Day()-int(c.From), 0, 0, 0, 0, time.Local).UTC()
	queryEnd := time.Date(now.Year(), now.Month(), now.Day()-int(c.To), 23, 59, 59, 0, time.Local).UTC()

	entries, err := ss.Finished(queryStart, queryEnd, c.Title)
	if err != nil {
		return err
	}

	view, err := store.View(entries, c.Type)
	if err != nil {
		return err
	}
	fmt.Println(view)
	return nil
}

type ServerCmd struct {
	Addr string `arg:"" help:"Address to which the server listens"`
}

func (c *ServerCmd) Run(ss store.Store) error {
	env := server.Env{Store: ss}
	return env.Run(c.Addr)
}

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
