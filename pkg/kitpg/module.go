package kitpg

import (
	"context"
	"fmt"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitdi"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"math"
	"net/url"
)

type Config struct {
	Host     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	User     string `env:"POSTGRES_USER" envDefault:"postgres"`
	Password string `env:"POSTGRES_PASSWORD" envDefault:"postgres"`
	Port     string `env:"POSTGRES_PORT" envDefault:"5432"`
	Database string `env:"POSTGRES_DB" envDefault:"postgres"`
	SSLMode  string `env:"POSTGRES_SSL_MODE" envDefault:"disable"`
	LogLevel int    `env:"POSTGRES_LOG_LEVEL" envDefault:"1"`

	GormConfig *gorm.Config

	ConnectionName *string `env:"POSTGRES_CONNECTION_NAME"`
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
			annots = append(annots, kitdi.Name(fmt.Sprintf("kitpg.config.%s", *config.ConnectionName)))
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

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		m.config.Host, m.config.Port, m.config.User, m.config.Database, m.config.Password, m.config.SSLMode,
	)

	if gc.Logger == nil {
		gc.Logger = logger.Default.LogMode(logger.LogLevel(m.config.LogLevel))
	}

	db, err := gorm.Open(postgres.Open(dsn), gc)
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
	return "kitpg"
}

func (c Config) DSN(queries url.Values) (dsn string) {
	dsn = fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		c.User, c.Password, c.Host, c.Port, c.Database,
	)

	queries.Set("sslmode", c.SSLMode)

	if len(queries) > 0 {
		dsn += "?" + queries.Encode()
	}

	return
}
