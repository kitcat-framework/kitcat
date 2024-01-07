package commands

import (
	"github.com/kitcat-framework/kitcat/pkg/kitcat-cli/utils"
	"github.com/mkideal/cli"
	"strings"
)

type migrateDiff struct {
	cli.Helper

	RevisionSchema string `cli:"revisions-schema" usage:"name of the schema the revisions table resides in"`
	LockTimeout    string `cli:"lock-timeout" usage:"set how long to wait for the database lock (default 10s)"`
	Format         string `cli:"format" usage:"Go template to use to format the output"`
	Qualifier      string `cli:"qualifier" usage:"qualify tables with custom qualifier when working on a single schema"`
	Edit           bool   `cli:"edit" usage:"edit the generated migration file(s)"`
}

var MigrateDiff = &cli.Command{
	Name:    "diff",
	Aliases: []string{"d"},
	Desc:    "diff migrations and output them in the migration directory",
	Argv:    func() interface{} { return new(migrateDiff) },
	Fn: func(ctx *cli.Context) error {
		m := ctx.Argv().(*migrateDiff)
		return migrateDiffFunc(m)
	},
}

func migrateDiffFunc(m *migrateDiff) error {
	cmd := []string{"atlas", "migrate", "diff", "--env", "gen_mig"}

	if m.RevisionSchema != "" {
		cmd = append(cmd, "--revision-schema")
	}

	if m.LockTimeout != "" {
		cmd = append(cmd, "--lock-timeout", m.LockTimeout)
	}

	if m.Format != "" {
		cmd = append(cmd, "--format", m.Format)
	}

	if m.Qualifier != "" {
		cmd = append(cmd, "--qualifier", m.Qualifier)
	}

	if m.Edit {
		cmd = append(cmd, "--edit")
	}

	basePath, err := utils.FindGoModPath()
	if err != nil {
		return utils.Err(err)
	}

	err = utils.ExecShellCommandInTerm(basePath, strings.Join(cmd, " "))
	if err != nil {
		return utils.Err(err)
	}

	return nil
}
