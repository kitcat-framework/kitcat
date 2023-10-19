package gen_setup_migrations

import _ "embed"

//go:embed gorm_atlas.go.tmpl
var GormAtlasFile string

//go:embed gorm_main.go.tmpl
var GormMainFile string

type AtlasParams struct {
	Dev          string
	MigrationDir string
	Diff         string
	Driver       string
}

func NewAtlasParams(driver, migrationDir string) *AtlasParams {
	driverToDev := map[string]string{
		"sqlite":   "sqlite://file::memory:?cache=shared",
		"mysql":    "docker://mysql/8/dev",
		"postgres": "docker://postgres/15",
	}

	return &AtlasParams{
		Dev:          driverToDev[driver],
		MigrationDir: migrationDir,
		Diff:         "{{ sql . \"  \" }}",
		Driver:       driver,
	}
}
