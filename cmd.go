package main

import (
	"bufio"
	"cranejin.com/ticktock/store"
	"fmt"
	"os"
	"strings"
)

type StartCmd struct {
	Wait  bool   `short:"w" help:"If set, wait for notes input until Ctrl-D, then finish the ticktock"`
	Title string `arg:"" name:"title" help:"The title of the ticktock"`
	Notes string `help:"Notes of the ticktock"`
}

func (c *StartCmd) Run(ss store.Store) error {
	if err := ss.StartTitle(c.Title, c.Notes); err != nil {
		return err
	}
	fmt.Printf("(Started: %s)\n", c.Title)

	if c.Wait {
		fmt.Println("Waiting for notes input, Ctrl-D ends the input and finish the ticktock:")
		notes, err := readToEOF()
		if err != nil {
			return fmt.Errorf("Failed to read notes: %w, ticktock not finished.", err)
		}

		r, err := ss.FinishLatest(notes)
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
	Notes []string `help:"Notes to appends, each input as a line. If a single '-' specified, read from stdin."`
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

	r, err := ss.FinishLatest(notes)
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
	Limit uint8 `short:"n" help:"Number of titles to display, default 5"`
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

var Cli struct {
	Db     string    `required:"" type:"path" help:"Path of the db file"`
	Start  StartCmd  `cmd:"" help:"Start a ticktock"`
	Finish FinishCmd `cmd:"" help:"Finish the ongoing ticktock"`
	Titles TitlesCmd `cmd:"" help:"Print recent finished titles"`
}

func readToEOF() (string, error) {
	var b strings.Builder
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Fprintln(&b, scanner.Text())
	}

	return b.String(), scanner.Err()
}
