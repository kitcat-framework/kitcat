package kitconfig

import (
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestEnv(t *testing.T) {
	t.Run("for string", func(t *testing.T) {
		require.Equal(t, "production", Env("ENV", "production"))
		require.NoError(t, os.Setenv("ENV", "development"))
		require.Equal(t, "development", Env("ENV", "production"))
	})

	t.Run("for duration", func(t *testing.T) {
		require.Equal(t, time.Second*30, Env("DURATION", time.Second*30))
		require.NoError(t, os.Setenv("DURATION", "1s"))
		require.Equal(t, time.Second*1, Env("DURATION", time.Second*30))
	})
}

func TestEnvPtr(t *testing.T) {
	t.Run("for string", func(t *testing.T) {
		require.Equal(t, lo.ToPtr("production"), EnvPtr("ENV", lo.ToPtr("production")))
		require.NoError(t, os.Setenv("ENV", "development"))
		require.Equal(t, lo.ToPtr("development"), EnvPtr("ENV", lo.ToPtr("production")))
	})

	t.Run("for duration", func(t *testing.T) {
		require.Equal(t, lo.ToPtr(time.Second*30), EnvPtr("DURATION", lo.ToPtr(time.Second*30)))
		require.NoError(t, os.Setenv("DURATION", "1s"))
		require.Equal(t, lo.ToPtr(time.Second*1), EnvPtr("DURATION", lo.ToPtr(time.Second*30)))
	})
}

func TestFromEnv(t *testing.T) {
	type testEnv struct {
		SuperEnv    string `env:"SUPER_ENV,required"`
		SuperConfig string `envDefault:"defaultValue"`
	}

	assert.Panics(t, func() {
		FromEnv[testEnv]()
	})

	require.NoError(t, os.Setenv("SUPER_ENV", "superEnv"))

	env := FromEnv[testEnv]()

	require.Equal(t, "superEnv", env.SuperEnv)
	require.Equal(t, "defaultValue", env.SuperConfig)

	env = FromEnv[testEnv](&testEnv{
		SuperConfig: "superConfig",
	})

	require.Equal(t, "superEnv", env.SuperEnv)
	require.Equal(t, "superConfig", env.SuperConfig)

	require.NoError(t, os.Setenv("SUPER_ENV", ""))

	env = FromEnv[testEnv](&testEnv{
		SuperConfig: "superConfig",
		SuperEnv:    "superEnv",
	})
}
