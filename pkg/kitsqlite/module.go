package kitsqlite

import (
	"context"
	"fmt"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitdi"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"math"
)

type Config struct {
	File     string `env:"SQLITE_FILE" envDefault:"db.sqlite"`
	LogLevel int    `env:"SQLITE_LOG_LEVEL" envDefault:"1"`

	GormConfig *gorm.Config

	ConnectionName *string `env:"SQLITE_CONNECTION_NAME"`
}

type Module struct {
	config     *Config
	connection *gorm.DB
}

func New(config *Config) func(a *kitcat.App) {
	return func(app *kitcat.App) {
		m := &Module{config: config}

		app.Provides(
			kitcat.ConfigurableAnnotation(m),
		)

		var annots []kitdi.AnnotateOption
		if config.ConnectionName != nil {
			annots = append(annots, kitdi.Name(fmt.Sprintf("kitsqlite.config.%s", *config.ConnectionName)))
		}

		app.Provides(
			kitdi.Annotate(config, annots...),
		)
	}
}

func (m *Module) Configure(_ context.Context, app *kitcat.App) error {
	var annots []kitdi.AnnotateOption
	gc := m.config.GormConfig
	if gc == nil {
		gc = &gorm.Config{}
	}

	if m.config.ConnectionName != nil {
		annots = append(annots, kitdi.Name(fmt.Sprintf("gorm.conn.%s", *m.config.ConnectionName)))
	}

	if gc.Logger == nil {
		gc.Logger = logger.Default.LogMode(logger.LogLevel(m.config.LogLevel))
	}

	db, err := gorm.Open(sqlite.Open(m.config.File), m.config.GormConfig)
	if err != nil {
		return err
	}

	m.connection = db

	app.Provides(
		kitdi.Annotate(db, annots...),
	)

	return nil
}

func (m *Module) Priority() uint8 { return math.MaxUint8 }

func (m *Module) Name() string {
	return "kitsqlite"
}
