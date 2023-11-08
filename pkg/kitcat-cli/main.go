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
			cli.Tree(commands.GenDockerCompose),
		),
		cli.Tree(commands.Migrate,
			cli.Tree(commands.MigrateApply),
			cli.Tree(commands.MigrateDiff),
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
	Argv: func() any { return new(rootT) },
	Fn: func(ctx *cli.Context) error {
		return help.Run(ctx.Args())
	},
}
