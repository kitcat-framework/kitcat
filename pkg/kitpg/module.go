package kitpg

import (
	"context"
	"fmt"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitdi"
	"github.com/expectedsh/kitcat/kitevent"
	"github.com/expectedsh/kitcat/pkg/kitpg/kiteventpg"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"math"
	"net/url"
)

type Config struct {
	Host     string `cfg:"host"`
	User     string `cfg:"user"`
	Password string `cfg:"password"`
	Port     string `cfg:"port"`
	Database string `cfg:"database"`
	SSLMode  string `cfg:"sslmode"`
	LogLevel int    `cfg:"log_level"`

	GormConfig *gorm.Config // manually configurable
}

func (c *Config) InitConfig(prefix string) kitcat.ConfigUnmarshal {
	prefix = prefix + ".database.postgres"

	viper.SetDefault(prefix+".host", "localhost")
	viper.SetDefault(prefix+".port", "5444")
	viper.SetDefault(prefix+".user", "postgres")
	viper.SetDefault(prefix+".password", "postgres")
	viper.SetDefault(prefix+".database", "postgres")
	viper.SetDefault(prefix+".sslmode", "disable")
	viper.SetDefault(prefix+".log_level", 1)

	return kitcat.ConfigUnmarshalHandler(prefix, c, "unable to unmarshal kitpg config: %w")
}

func init() {
	kitcat.RegisterConfig(new(Config))
}

type Module struct {
	config     *Config
	connection *gorm.DB
}

func New(_ kitdi.Invokable, app *kitcat.App, config *Config) {
	m := &Module{config: config}

	app.Provides(
		kitcat.ProvideConfigurableModule(m),
		kitevent.ProvideStore(kiteventpg.New),
	)
}

func (m *Module) Configure(_ context.Context, app *kitcat.App) error {

	gc := m.config.GormConfig
	if gc == nil {
		gc = &gorm.Config{}
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

	app.Provides(db)

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
