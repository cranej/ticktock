package main

import (
	"cranejin.com/ticktock/store"
	"github.com/alecthomas/kong"
)

func main() {
	ctx := kong.Parse(&Cli,
		kong.Description("Ticktock is a tool for better tracking time usage. "))
	db, err := store.NewSqliteStore(Cli.Db)
	ctx.FatalIfErrorf(err)

	ctx.BindTo(db, (*store.Store)(nil))
	err = ctx.Run(db)
	ctx.FatalIfErrorf(err)
}
