package main

import (
	"os"

	"mpr/internal/commands"
	"mpr/internal/util"
)

func main() {
	os.Args = util.PreprocessArgs(os.Args)
	commands.Execute()
}
