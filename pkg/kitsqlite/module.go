package kitsqlite

import (
	"context"
	"fmt"
	"github.com/kitcat-framework/kitcat"
	"github.com/kitcat-framework/kitcat/kitdi"
	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"math"
	"net/url"
)

type Config struct {
	File     string `cfg:"file"`
	LogLevel int    `cfg:"log_level"`

	GormConfig *gorm.Config
}

func (c *Config) InitConfig(prefix string) kitcat.ConfigUnmarshal {
	prefix = prefix + ".database.sqlite"

	viper.SetDefault(prefix+".file", "db.sqlite")
	viper.SetDefault(prefix+".log_level", 1)

	return kitcat.ConfigUnmarshalHandler(prefix, c, "unable to unmarshal kitsqlite config: %w")
}

func Init() {
	kitcat.RegisterConfig(new(Config))
}

type Module struct {
	config     *Config
	connection *gorm.DB
}

func New(_ kitdi.Invokable, config *Config, app *kitcat.App) {
	m := &Module{config: config}

	app.Provides(
		kitcat.ProvideConfigurableModule(m),
	)
}

func (m *Module) Configure(_ context.Context, app *kitcat.App) error {
	gc := m.config.GormConfig
	if gc == nil {
		gc = &gorm.Config{}
	}

	if gc.Logger == nil {
		gc.Logger = logger.Default.LogMode(logger.LogLevel(m.config.LogLevel))
	}

	db, err := gorm.Open(sqlite.Open(m.config.File), gc)
	if err != nil {
		return err
	}

	m.connection = db

	app.Provides(db)

	return nil
}

func (m *Module) Priority() uint8 { return math.MaxUint8 }

func (m *Module) Name() string {
	return "kitsqlite"
}

func (c Config) DSN(_ url.Values) (dsn string) {
	return fmt.Sprintf("sqlite://%s", c.File)
}
