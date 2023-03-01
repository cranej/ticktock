package main

import (
	"cranejin.com/ticktock/store"
	"fmt"
)

type StartCmd struct {
	Title string `arg:"" name:"title" help:"the title of the ticktock"`
	Notes string `help:"comments of the ticktock"`
}

func (c *StartCmd) Run(ss store.Store) error {
	if err := ss.StartTitle(c.Title, c.Notes); err != nil {
		return err
	}

	fmt.Printf("(Started: %s)\n", c.Title)
	return nil
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
