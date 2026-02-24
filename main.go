package main

import (
	"github.com/alecthomas/kong"
	"mpr/internal/cli"
)

func main() {
	var c cli.CLI
	ctx := kong.Parse(&c,
		kong.Name("mpr"),
		kong.Description("CLI for USDA AMS MPR Datamart"),
		kong.UsageOnError(),
	)
	ctx.FatalIfErrorf(ctx.Run(&c.Globals))
}
