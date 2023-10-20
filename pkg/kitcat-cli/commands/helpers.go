package commands

import "github.com/mkideal/cli"

var help = cli.HelpCommand("display help information")

type WithEnv struct {
	Env string `cli:"env" usage:"env to use to migrate with atlas"`
}
