package main

import (
	"fmt"

	"github.com/alecthomas/kong"
	"mpr/internal/cli"
)

var (
	version   = "dev"
	gitCommit = "unknown"
	buildTime = "unknown"
)

func main() {
	// Set version variables for update checking
	cli.Version = version
	cli.GitCommit = gitCommit
	cli.BuildTime = buildTime

	var c cli.CLI
	ctx := kong.Parse(&c,
		kong.Name("mpr"),
		kong.Description("CLI for USDA AMS MPR Datamart"),
		kong.UsageOnError(),
	)

	// If version flag was passed, print version and exit
	if ctx.Command() == "version" || (len(ctx.Args) > 0 && ctx.Args[0] == "--version") {
		fmt.Printf("mpr %s (%s) built %s\n", version, gitCommit, buildTime)
		return
	}

	// Perform background update check on startup (non-blocking)
	// This will check once per day and only notify if an update is available
	cli.AutoUpdateCheck(c.CacheDir)

	ctx.FatalIfErrorf(ctx.Run(&c.Globals))
}
