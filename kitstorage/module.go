package kitstorage

import (
	"context"
	"fmt"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitdi"
	"github.com/expectedsh/kitcat/kitslog"
	"github.com/expectedsh/kitcat/kitweb"
	"github.com/spf13/viper"
	"log/slog"
)

type Config struct {
	FileSystemName string `cfg:"filesystem_name"`
}

func (c *Config) InitConfig(prefix string) kitcat.ConfigUnmarshal {
	prefix = prefix + ".kitstorage"
	viper.SetDefault(prefix+".filesystem_name", "local")

	return kitcat.ConfigUnmarshalHandler(prefix, c, "unable to unmarshal kitstorage config: %w")
}

func init() {
	kitcat.RegisterConfig(new(Config))
}

type Module struct {
	Config            *Config
	CurrentFileSystem FileSystem

	logger *slog.Logger
}

func New(_ kitdi.Invokable, cfg *Config, app *kitcat.App) {
	mod := &Module{
		Config: cfg,
		logger: slog.With(kitslog.Module("kitstorage")),
	}

	app.Provides(
		kitcat.ProvideConfigurableModule(mod),
		ProvideFileSystem(NewLocalFileSystem),
	)
}

func (m *Module) Configure(_ context.Context, app *kitcat.App) error {
	app.Invoke(m.setCurrentFileSystem)

	return nil
}

func (m *Module) Priority() uint8 { return 254 }

func (m *Module) setCurrentFileSystem(a *kitcat.App, fs fileSystems) error {
	fmt.Println("setCurrentFileSystem", fs)
	implementation, err := kitcat.UseImplementation(kitcat.UseImplementationParams[FileSystem]{
		ModuleName:                m.Name(),
		ImplementationTerminology: "filesystem",
		ConfigImplementationName:  m.Config.FileSystemName,
		Implementations:           fs.FileSystems,
	})
	if err != nil {
		return fmt.Errorf("unable to use implementation: %w", err)
	}

	m.CurrentFileSystem = implementation.(FileSystem)
	m.logger.Info("using filesystem", slog.String("fs", m.CurrentFileSystem.Name()))

	a.Provides(kitdi.Annotate(m.CurrentFileSystem, kitdi.As(new(FileSystem))))

	if fs, ok := m.CurrentFileSystem.(*LocalFileSystem); ok {
		a.Provides(kitweb.ProvideHandler(fs)) // todo: maybe delegate this to kitweb module
	}

	return nil
}

func (m *Module) Name() string { return "kitstorage" }
