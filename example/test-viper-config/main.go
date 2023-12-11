package main

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"os"
	"strings"
	"time"
)

type Kitcat struct {
	Name string `mapstructure:"_app_name"`
	Env  string `mapstructure:"_env"`
}

type KitEventConfig struct {
	StoreName string `mapstructure:"store_name"`
}

type KitEventInMemoryConfig struct {
	Retention time.Duration `mapstructure:"retention"`
}

func main() {
	viper.SetConfigFile("config.yaml")
	viper.SetDefault("_app_name", "kitcat")
	viper.SetDefault("_env", "development")

	// Set defaults for each environment
	envs := []string{"tests", "development", "staging", "production"}
	for _, env := range envs {
		viper.SetDefault(fmt.Sprintf("%s.kitevent.store_name", env), "in_memory")
		viper.SetDefault(fmt.Sprintf("%s.kitevent.in_memory.retention", env), time.Second)
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		// If the config file doesn't exist, write the default configuration
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) || strings.Contains(err.Error(), "no such file or directory") {
			fmt.Println("Config file not found. Writing default configuration...")
			if err := viper.WriteConfigAs("config.yaml"); err != nil {
				fmt.Printf("Error writing config file: %s\n", err)
				return
			}
			fmt.Println("Default configuration written.")
		} else {
			fmt.Printf("Error reading config file: %s\n", err)
			return
		}
	}

	for _, k := range viper.AllKeys() {
		val := viper.GetString(k)
		if strings.HasPrefix(val, "$") {
			viper.Set(k, os.ExpandEnv(val))
		} else {
			viper.Set(k, val)
		}
	}

	//for _, k := range viper.AllKeys() {
	//	fmt.Println()
	//	v := viper.GetString(k)
	//	fmt.Println("BEFORE", k, v)
	//	//if strings.HasPrefix(v, "$") {
	//	//	fmt.Println("REPLACING ENV VAR", k, v, os.ExpandEnv(v))
	//	//	viper.Set(k, os.ExpandEnv(v))
	//	//}
	//
	//	fmt.Println("RESULT", k, viper.GetString(k))
	//}

	//env := viper.GetString("_env")/
	//fmt.Println("keys in env = ", viper.Sub("development").AllKeys())
	//viper.Set("development.kitevent.in_memory.retention", "1s")
	fmt.Println("keys in env = ", viper.Sub("development").AllKeys())

	var kitcatConfig Kitcat
	if err := viper.Unmarshal(&kitcatConfig); err != nil {
		fmt.Printf("Error unmarshal config file: %s\n", err)
		return
	}

	fmt.Printf("kitcatConfig: %+v\n", kitcatConfig)

	fromEnv := viper.Sub(kitcatConfig.Env)

	var kitEventConfig KitEventConfig
	for i, s := range fromEnv.AllKeys() {
		fmt.Println(i, s, fromEnv.GetString(s))
	}
	if err := fromEnv.Sub("kitevent").Unmarshal(&kitEventConfig); err != nil {
		fmt.Printf("Error unmarshal config file: %s\n", err)
		return
	}

	var kitEventInMemoryConfig KitEventInMemoryConfig
	if err := fromEnv.Sub("kitevent").Sub("in_memory").Unmarshal(&kitEventInMemoryConfig); err != nil {
		fmt.Printf("Error unmarshal config file: %s\n", err)
		return
	}

	fmt.Printf("kitcatConfig: %+v\n", kitcatConfig)
	fmt.Printf("kitEventConfig: %+v\n", kitEventConfig.StoreName)
	fmt.Printf("kitEventInMemoryConfig: %+v\n", kitEventInMemoryConfig)
}
func expandEnvVariables(m interface{}) {
	switch v := m.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if s, ok := value.(string); ok && strings.Contains(s, "$") {
				v[key] = os.ExpandEnv(s)
			} else {
				expandEnvVariables(value)
			}
		}
	case []interface{}:
		for i, item := range v {
			expandEnvVariables(item)
			v[i] = item
		}
	}
}
