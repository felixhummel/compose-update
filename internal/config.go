package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	DryRun      bool          `mapstructure:"dry_run"`
	Directory   string        `mapstructure:"directory"`
	Exclude     []string      `mapstructure:"exclude"`
	Glob        []string      `mapstructure:"glob"`
	Major       bool          `mapstructure:"major"`
	Minor       bool          `mapstructure:"minor"`
	Patch       bool          `mapstructure:"patch"`
	LogLevel    string        `mapstructure:"log_level"`
	MaxTime     time.Duration `mapstructure:"max_time"`
	Output      string        `mapstructure:"output"`
}

func getConfigDir() string {
	// XDG_CONFIG_HOME defaults to ~/.config
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "."
		}
		configHome = filepath.Join(home, ".config")
	}
	return configHome
}

func ReadConfig() Config {
	// Set up Viper for config file in $XDG_CONFIG_HOME/compose-update/config.toml
	configDir := filepath.Join(getConfigDir(), "compose-update")
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(configDir)

	// Set defaults
	viper.SetDefault("directory", ".")
	viper.SetDefault("log_level", "warning")
	viper.SetDefault("max_time", "5s")
	viper.SetDefault("major", false)
	viper.SetDefault("minor", false)
	viper.SetDefault("patch", false)
	viper.SetDefault("dry_run", false)
	viper.SetDefault("exclude", []string{})
	viper.SetDefault("glob", []string{})
	viper.SetDefault("output", "text")

	// Try to read config file (don't fail if it doesn't exist)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshaling config: %v\n", err)
	}

	return cfg
}
