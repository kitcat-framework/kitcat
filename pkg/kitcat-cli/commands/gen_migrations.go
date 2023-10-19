package commands

import (
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/expectedsh/kitcat/pkg/kitcat-cli/teacomponents"
	"github.com/expectedsh/kitcat/pkg/kitcat-cli/templates/gen_setup_migrations"
	"github.com/expectedsh/kitcat/pkg/kitcat-cli/utils"
	"github.com/mkideal/cli"
	"path"
)

type generateSetupMigrations struct {
	cli.Helper

	Driver    string `cli:"driver" usage:"driver (mysql, postgres, sqlite)" validate:"required,oneof=mysql postgres sqlite"`
	Strategy  string `cli:"strategy" usage:"atlasgo strategy (golang-migrate, gorm)" validate:"oneof=golang-migrate gorm"`
	Directory string `cli:"directory" usage:"directory where migrations will be stored" validate:"required"`
}

var GenSetupMigration = &cli.Command{
	Name:    "setup-migrations",
	Aliases: []string{"sm"},
	Desc:    "this command generate files for kitcat framework to allow you to migrate your models",
	Argv:    func() interface{} { return new(generateSetupMigrations) },
	Fn: func(ctx *cli.Context) error {
		m := ctx.Argv().(*generateSetupMigrations)
		err := genSetupMigrationFunc(m)
		if err != nil {
			return utils.Err(err)
		}

		return nil
	},
}

func genSetupMigrationFunc(m *generateSetupMigrations) error {
	if !utils.HasBinary("atlas") {
		fmt.Println("To migrate kitcat use https://atlasgo.io/ go to https://atlasgo.io/getting-started.\n" +
			"Install it and then you can do this command")
		return errors.New("atlasgo not found")
	}

	err := genSetupMigrationQuestions(m)
	if err != nil {
		return err
	}

	switch m.Strategy {
	case "gorm":
		return genSetupMigrationGorm(m)
	}

	return nil
}

func genSetupMigrationGorm(m *generateSetupMigrations) error {
	fmt.Println()

	params := gen_setup_migrations.NewAtlasParams(m.Driver, m.Directory)

	mainFile, err := utils.Template(gen_setup_migrations.GormMainFile, params)
	if err != nil {
		return err
	}

	atlasFile, err := utils.Template(gen_setup_migrations.GormAtlasFile, params)
	if err != nil {
		return err
	}

	basePath, err := utils.FindGoModPath()
	if err != nil {
		return err
	}

	f, err := utils.CreateFileWithDirsIfNotExist(path.Join(basePath, "cmd/kitmigrate/main.go"), mainFile.String())
	if err != nil {
		return err
	}
	defer f.Close()

	f, err = utils.CreateFileWithDirsIfNotExist(path.Join(basePath, "atlas.hcl"), atlasFile.String())
	if err != nil {
		return err
	}
	defer f.Close()

	modsToInstall := []string{
		"ariga.io/atlas-go-sdk",
		"ariga.io/atlas-provider-gorm",
	}

	for _, mod := range modsToInstall {
		_, err := utils.ExecShellCommandAt(basePath, fmt.Sprintf("go get -u %s", mod))
		if err != nil {
			return err
		}
	}

	if _, err := utils.ExecShellCommandAt(basePath, "go mod tidy"); err != nil {
		return err
	}

	fmt.Println("You can now run `kitcat-cli g mig` to generate your migrations")

	fmt.Println("")

	return nil
}

func genSetupMigrationQuestions(m *generateSetupMigrations) error {
	if m.Driver == "" {
		p := tea.NewProgram(&teacomponents.Choice{
			Question: "Which database driver do you want to use?",
			Choices:  []teacomponents.ChoiceItem{{Value: "mysql"}, {Value: "postgres"}, {Value: "sqlite"}},
			Choice:   "postgres",
		})

		run, err := p.Run()
		if err != nil {
			return err
		}

		if c, ok := run.(*teacomponents.Choice); ok && c.Choice != "" {
			m.Driver = c.Choice
		}
	}

	if m.Strategy == "" {
		p := tea.NewProgram(&teacomponents.Choice{
			Question: "Which strategy do you want to use?",
			Choices: []teacomponents.ChoiceItem{
				{
					Value: "gorm",
					Hint:  "use your gorm models as a source of truth",
				},
				{
					Value: "golang-migrate",
					Hint:  "use a schema.sql file as a source of truth",
				},
			},
			Choice: "gorm",
		})

		run, err := p.Run()
		if err != nil {
			return err
		}

		if c, ok := run.(*teacomponents.Choice); ok && c.Choice != "" {
			m.Strategy = c.Choice
		}
	}

	if m.Directory == "" {
		p := tea.NewProgram(teacomponents.Input{
			Question:  "Where do you want to store your migrations?",
			TextInput: teacomponents.NewTextInput("./migrations"),
		})

		run, err := p.Run()
		if err != nil {
			return err
		}

		if c, ok := run.(teacomponents.Input); ok && c.TextInput.Value() != "" {
			m.Directory = c.TextInput.Value()
		}
	}
	return nil
}
