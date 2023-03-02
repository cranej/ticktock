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
	Title string `arg:"" name:"title" help:"the title of the ticktock"`
	Notes string `help:"comments of the ticktock"`
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
	Notes string `help:"comments to appends"`
}

func (c *FinishCmd) Run(ss store.Store) error {
	r, err := ss.FinishLatest(c.Notes)
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

var Cli struct {
	Db     string    `required:"" type:"path" help:"path of the db file"`
	Start  StartCmd  `cmd:"" help:"start a ticktock"`
	Finish FinishCmd `cmd: "" help:"finish the ongoing ticktock"`
}

func readToEOF() (string, error) {
	var b strings.Builder
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Fprintln(&b, scanner.Text())
	}

	return b.String(), scanner.Err()
}
