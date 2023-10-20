package commands

import "github.com/mkideal/cli"

type generator struct {
	cli.Helper
}

var Generator = &cli.Command{
	Name:    "generator",
	Aliases: []string{"g"},
	Desc:    "this command generate files for kitcat framework",
	Argv:    func() interface{} { return new(generator) },
	Fn: func(ctx *cli.Context) error {
		return cli.HelpCommandFn(ctx)
	},
}
