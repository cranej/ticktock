package main

import (
	"cranejin.com/ticktock/store"
)

type StartCmd struct {
	Notes string `help:"comments of the ticktock"`
	Title string `arg:"" name:"title" help:"the title of the ticktock"`
}

func (c *StartCmd) Run(ss store.Store) error {
	return ss.StartTitle(c.Title, c.Notes)
}

type FinishCmd struct {
	Title string `arg:"" name: "title" help:"title to finish, or latest one if not specified"`
}

func (c *FinishCmd) Run(ss store.Store) error {
	return ss.Finish(c.Title)
}

var Cli struct {
	Db     string    `required:"" type:"path" help:"path of the db file"`
	Start  StartCmd  `cmd:"" help:"start a ticktock"`
	Finish FinishCmd `cmd: "" help:"finish a ticktock"`
}
