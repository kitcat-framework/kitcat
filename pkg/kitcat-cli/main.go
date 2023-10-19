package main

import (
	"fmt"
	"github.com/expectedsh/kitcat/pkg/kitcat-cli/commands"
	"github.com/mkideal/cli"
	"os"
)

func main() {
	cli := cli.Root(
		root,
		cli.Tree(help),
		cli.Tree(commands.Generator,
			cli.Tree(commands.GenSetupMigration),
		),
	)

	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var help = cli.HelpCommand("display help information")

// root command
type rootT struct {
	cli.Helper
}

var root = &cli.Command{
	Desc: "kitcat-cli is a command line tool for kitcat framework",
	// Argv is a factory function of argument object
	// ctx.Argv() is if Command.Argv == nil or Command.Argv() is nil
	Argv: func() any { return new(rootT) },
	Fn: func(ctx *cli.Context) error {
		return help.Run(ctx.Args())
	},
}
