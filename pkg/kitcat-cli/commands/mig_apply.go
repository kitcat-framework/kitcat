package commands

import (
	"github.com/expectedsh/kitcat/pkg/kitcat-cli/utils"
	"github.com/mkideal/cli"
	"strings"
)

type migrateApply struct {
	cli.Helper

	RevisionSchema string `cli:"revisions-schema" usage:"name of the schema the revisions table resides in"`
	DryRun         bool   `cli:"dry-run" usage:"print SQL without executing it"`
	LockTimeout    string `cli:"lock-timeout" usage:"set how long to wait for the database lock (default 10s)"`
	Baseline       string `cli:"baseline" usage:"start the first migration after the given baseline version"`
	TxMode         string `cli:"tx-mode" usage:"set transaction mode [none, file, all] (default \"file\")"`
	AllowDirty     bool   `cli:"allow-dirty" usage:"allow start working on a non-clean database"`
}

var MigrateApply = &cli.Command{
	Name:    "apply",
	Aliases: []string{"a"},
	Desc:    "apply migrations to your database",
	Argv:    func() interface{} { return new(migrateApply) },
	Fn: func(ctx *cli.Context) error {
		m := ctx.Argv().(*migrateApply)
		return migrateApplyFunc(m)
	},
}

func migrateApplyFunc(m *migrateApply) error {
	cmd := []string{"atlas", "migrate", "apply", "--env", "apply_mig"}

	if m.RevisionSchema != "" {
		cmd = append(cmd, "--revision-schema")
	}

	if m.DryRun {
		cmd = append(cmd, "--dry-run")
	}

	if m.LockTimeout != "" {
		cmd = append(cmd, "--lock-timeout", m.LockTimeout)
	}

	if m.Baseline != "" {
		cmd = append(cmd, "--baseline", m.Baseline)
	}

	if m.TxMode != "" {
		cmd = append(cmd, "--tx-mode")
	}

	if m.AllowDirty {
		cmd = append(cmd, "--allow-dirty")
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
