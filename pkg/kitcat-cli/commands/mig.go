package commands

import (
	"github.com/mkideal/cli"
)

type migrate struct {
	cli.Helper
}

var Migrate = &cli.Command{
	Name:    "migrate",
	Aliases: []string{"m"},
	Desc:    "this command apply migrations to your database",
	Argv:    func() interface{} { return new(migrate) },
	Fn: func(ctx *cli.Context) error {
		return help.Run(ctx.Args())
	},
}
