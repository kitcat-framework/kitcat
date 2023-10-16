package kitpg

import (
	"fmt"
	"github.com/expectedsh/kitcat"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Host     string `env:"POSTGRES_HOST"`
	User     string `env:"POSTGRES_USER"`
	Password string `env:"POSTGRES_PASSWORD"`
	Port     string `env:"POSTGRES_PORT"`
	Database string `env:"POSTGRES_DB"`
	SSLMode  string `env:"POSTGRES_SSL_MODE" envDefault:"disable"`
	LogLevel int    `env:"POSTGRES_LOG_LEVEL" envDefault:"1"`

	GormConfig *gorm.Config
}

type Module struct {
	Config *Config

	connection *gorm.DB
}

func New(config *Config) func(app *kitcat.App) *Module {
	return func(app *kitcat.App) *Module {
		return &Module{
			Config: config,
		}
	}
}

func (m *Module) OnStart(app *kitcat.App) error {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		m.Config.Host, m.Config.Port, m.Config.User, m.Config.Database, m.Config.Password, m.Config.SSLMode,
	)

	if m.Config.GormConfig.Logger == nil {
		m.Config.GormConfig.Logger = logger.Default.LogMode(logger.LogLevel(m.Config.LogLevel))
	}

	db, err := gorm.Open(postgres.Open(dsn), m.Config.GormConfig)
	if err != nil {
		return err
	}

	m.connection = db
	app.Provide(func() *gorm.DB { return db })

	return nil
}

func (m *Module) OnStop(app *kitcat.App) error {
	return nil
}

func (m *Module) Name() string {
	return "kitpg"
}
