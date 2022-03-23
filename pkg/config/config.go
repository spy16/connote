package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Duration returns the duration value set for the given key. Default will be
// set if provided.
func Duration(key string, defaultValue int, unit time.Duration) time.Duration {
	def := time.Duration(defaultValue) * unit
	viper.SetDefault(key, def/unit)
	return time.Duration(Int(key, int(def/unit))) * unit
}

// String returns the string value set for the given key. Default will be set
// if provided.
func String(key string, defaultVal ...string) string {
	if len(defaultVal) > 0 {
		viper.SetDefault(key, defaultVal[0])
	}
	return viper.GetString(key)
}

// Bool returns the boolean value set for the given key. Default will be set
// if provided.
func Bool(key string, defaultVal ...bool) bool {
	if len(defaultVal) > 0 {
		viper.SetDefault(key, defaultVal[0])
	}
	return viper.GetBool(key)
}

// Float64 returns the floating point number set for the given key. Default
// will be set if provided.
func Float64(key string, defaultVal ...float64) float64 {
	if len(defaultVal) > 0 {
		viper.SetDefault(key, defaultVal[0])
	}
	return viper.GetFloat64(key)
}

// Int returns the integer value set for the given key. Default will be set
// if provided.
func Int(key string, defaultVal ...int) int {
	if len(defaultVal) > 0 {
		viper.SetDefault(key, defaultVal[0])
	}
	return viper.GetInt(key)
}

func viperInit(envPrefix, configName string, configFile ...string) error {
	if len(configFile) > 0 && configFile[0] != "" {
		viper.SetConfigFile(configFile[0])
		if err := viper.ReadInConfig(); err != nil {
			return err
		}
	} else {
		viper.AddConfigPath("./samples")
		viper.AddConfigPath("./")
		viper.SetConfigName(configName)
		_ = viper.ReadInConfig()
	}

	// for transforming app.host to app_host
	repl := strings.NewReplacer(".", "_", "-", "_")
	viper.SetEnvKeyReplacer(repl)
	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv()
	return nil
}
