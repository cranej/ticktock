package main

import (
	"github.com/alecthomas/kong"
)

func main() {
	ctx := kong.Parse(&Cli)
	err := ctx.Run(Cli.Db)
	ctx.FatalIfErrorf(err)
}
