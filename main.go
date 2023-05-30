package main

import (
	"github.com/alecthomas/kong"
	"github.com/cranej/ticktock/store"
	"github.com/cranej/ticktock/version"
)

var Cli struct {
	Db      string           `required:"" env:"TICKTOCK_DB" type:"path" help:"Path of the db file, required if environment not set"`
	Version kong.VersionFlag `help:"Show version"`
	Start   StartCmd         `cmd:"" help:"Start an activity"`
	Close   CloseCmd         `cmd:"" help:"Close the ongoing activity"`
	Titles  TitlesCmd        `cmd:"" help:"Print titles of recent closed activities"`
	Ongoing OngoingCmd       `cmd:"" help:"Show currently ongoing activity"`
	Last    LastCmd          `cmd:"" help:"Show details of the latest closed activity with given title"`
	Report  ReportCmd        `cmd:"" help:"Show time usage report"`
	Server  ServerCmd        `cmd:"" help:"Start a server"`
	Add     AddCmd           `cmd:"" helo:"Add an closed activity"`
}

func main() {
	ctx := kong.Parse(&Cli,
		kong.Description("Ticktock is a tool for better tracking time usage. "),
		kong.Vars{
			"version": version.Version,
		})
	db, err := store.NewSqliteStore(Cli.Db)
	ctx.FatalIfErrorf(err)

	ctx.BindTo(db, (*store.Store)(nil))
	err = ctx.Run(db)
	ctx.FatalIfErrorf(err)
}
