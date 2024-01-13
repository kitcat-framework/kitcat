package kits3

import (
	"fmt"
	"github.com/kitcat-framework/kitcat"
	"github.com/kitcat-framework/kitcat/kitstorage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"
)

type Config struct {
	Endpoint        string `cfg:"endpoint"`
	AccessKey       string `cfg:"access_key"`
	SecretAccessKey string `cfg:"secret_access_key"`
	SSL             bool   `cfg:"ssl"`
}

func (c *Config) InitConfig(prefix string) kitcat.ConfigUnmarshal {
	prefix = prefix + ".kitstorage.file_systems.s3"

	viper.SetDefault(prefix+".endpoint", "localhost:9000")
	viper.SetDefault(prefix+".access_key", "YOUR_ACCESS_KEY")
	viper.SetDefault(prefix+".secret_access_key", "YOUR_SECRET_ACCESS_KEY")
	viper.SetDefault(prefix+".ssl", false)

	return kitcat.ConfigUnmarshalHandler(prefix, c, "unable to unmarshal kits3 config: %w")

}

func init() {
	kitcat.RegisterConfig(new(Config))
}

func Module(cfg *Config, a *kitcat.App) error {
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretAccessKey, ""),
		Secure: cfg.SSL,
	})

	if err != nil {
		return fmt.Errorf("unable to create minio client: %w", err)
	}

	a.Provides(minioClient, kitstorage.ProvideFileSystem(NewFileStorageS3))

	return nil
}
