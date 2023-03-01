package main

import (
	"os"
	"fmt"
	"cranejin.com/ticktock/store"
)

type StartCmd struct {
	Notes string `help:"comments of the ticktock"`
	Title string `arg:"" name:"title" help:"the title of the ticktock"`
}

func (s *StartCmd) Run(db string) error {
	fmt.Println("start:", s.Title, "notes: ", s.Notes, "db: ", db)
	_,err := store.NewSqliteStore(db)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return nil
}

type Finish struct {
	Title string `arg:"" name: "title" help:"title to finish, or latest one if not specified"`
}

func (s *Finish) Run(db string) error {
	fmt.Println("finish: ", s.Title, "db: ", db)
	return nil
}

var Cli struct {
	Db     string   `required:"" type:"path" help:"path of the db file"`
	Start  StartCmd `cmd:"" help:"start a ticktock"`
	Finish Finish   `cmd: "" help:"finish a ticktock"`
}
