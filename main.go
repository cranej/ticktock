package main

import (
	"github.com/alecthomas/kong"
	"github.com/cranej/ticktock/store"
	"github.com/cranej/ticktock/version"
)

var Cli struct {
	Db      string           `required:"" env:"TICKTOCK_DB" type:"path" help:"Path of the db file, required if environment not set"`
	Version kong.VersionFlag `help:"Show version"`
	Start   StartCmd         `cmd:"" help:"Start a ticktock"`
	Finish  FinishCmd        `cmd:"" help:"Finish the ongoing ticktock"`
	Titles  TitlesCmd        `cmd:"" help:"Print recent finished titles"`
	Ongoing OngoingCmd       `cmd:"" help:"Show currently ongoing ticktock"`
	Last    LastCmd          `cmd:"" help:"Show last finished ticktock details of title"`
	Report  ReportCmd        `cmd:"" help:"Show time usage report"`
	Server  ServerCmd        `cmd:"" help:"Start a server"`
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
