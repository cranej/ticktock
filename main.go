package main

import (
	"errors"
	"github.com/alecthomas/kong"
	"github.com/cranej/ticktock/store"
	"github.com/cranej/ticktock/version"
	"os"
	"path/filepath"
)

var Cli struct {
	Db      string           `type:"path" help:"Path of the db file, if not specified, try environment $TICKTOCK_DB, then default to $XDG_DATA_HOME/ticktock/db. $XDG_DATA_HOME default to $HOME/.local/share if not set."`
	Version kong.VersionFlag `help:"Show version"`
	Start   StartCmd         `cmd:"" help:"Start an activity"`
	Close   CloseCmd         `cmd:"" help:"Close the ongoing activity"`
	Titles  TitlesCmd        `cmd:"" help:"Print titles of recent closed activities"`
	Ongoing OngoingCmd       `cmd:"" help:"Show currently ongoing activity"`
	Last    LastCmd          `cmd:"" help:"Show details of the latest closed activity with given title"`
	Report  ReportCmd        `cmd:"" help:"Show time usage report"`
	Server  ServerCmd        `cmd:"" help:"Start a server"`
	Add     AddCmd           `cmd:"" help:"Add an closed activity"`
}

func main() {
	ctx := kong.Parse(&Cli,
		kong.Description("Ticktock is a tool for better tracking time usage. "),
		kong.Vars{
			"version": version.Version,
		})
	dbPath, err := dbPath(Cli.Db)
	ctx.FatalIfErrorf(err)
	Cli.Db = dbPath
	db, err := store.NewSqliteStore(Cli.Db)
	ctx.FatalIfErrorf(err)

	ctx.BindTo(db, (*store.Store)(nil))
	err = ctx.Run(db)
	ctx.FatalIfErrorf(err)
}

func dbPath(fromCmd string) (string, error) {
	if fromCmd != "" {
		return fromCmd, nil
	}

	if tickDbEnv := os.Getenv("TICKTOCK_DB"); tickDbEnv != "" {
		return tickDbEnv, nil
	}

	dbDir := ""
	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome == "" {
		xdgDataHome = os.Getenv("HOME")
		if xdgDataHome == "" {
			return "", errors.New("neither XDG_DATA_HOME nor HOME was set, could not determine db path")
		}
		dbDir = filepath.Join(xdgDataHome, ".local/share/ticktock")
	} else {
		dbDir = filepath.Join(xdgDataHome, "ticktock")
	}

	err := os.MkdirAll(dbDir, 0750)
	if err != nil {
		return "", err
	}

	return filepath.Join(dbDir, "db"), nil
}
