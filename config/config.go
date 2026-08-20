package config

import (
	"flag"
	"fmt"
	"strings"

	"github.com/spf13/viper"

	"github.com/MatiXxD/beerer-bot/pkg/logger"
)

// AppConfig config represents the configuration of the application.
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
}

// Config structs that contain all configuration for the application.
type Config struct {
	AppCfg AppConfig            `mapstructure:"app"`
	Logger logger.ZerologConfig `mapstructure:"logger"`
}

// Load loads the configuration.
func Load() (*Config, error) {
	var path string

	// try to get the config path from cli params
	flag.StringVar(&path, "config", "./config/config.yaml", "path to the config file")
	flag.Parse()

	return configFromPath(path)
}

// configFromPath loads the configuration from the yaml file.
func configFromPath(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file, %w", err)
	}

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config, %w", err)
	}

	cfg.Logger.AppName = cfg.AppCfg.Name
	cfg.Logger.AppVersion = cfg.AppCfg.Version

	return &cfg, nil
}
