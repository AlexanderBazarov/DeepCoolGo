package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	RefreshRate float32 `mapstructure:"refresh_rate"`
	Theme       string  `mapstructure:"theme"`
	ThemesPath  string  `mapstructure:"themes_path"`
}

func DefaultPath() (string, error) {
	home, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config dir: %w", err)
	}

	return filepath.Join(home, "DCGO", "config.yml"), nil
}

func (c *Config) ThemePath() string {
	return filepath.Join(c.ThemesPath, c.Theme)
}

func (c *Config) Interval() time.Duration {
	return time.Duration(float64(c.RefreshRate) * float64(time.Second))
}

func (c *Config) Validate() error {
	if c.RefreshRate <= 0 {
		return fmt.Errorf("refresh_rate must be greater than 0, got %f", c.RefreshRate)
	}
	if c.Theme == "" {
		return fmt.Errorf("theme must not be empty")
	}
	if c.ThemesPath == "" {
		return fmt.Errorf("themes_path must not be empty")
	}

	return nil
}

func Load(path string) (*Config, error) {
	if err := createDefault(path); err != nil {
		return nil, err
	}

	v := viper.New()

	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.SetDefault("refresh_rate", 30)
	v.SetDefault("theme", "aether.so")
	v.SetDefault("themes_path", defaultThemesPath())

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}

	return &cfg, nil
}

func createDefault(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	v := viper.New()

	v.Set("refresh_rate", 30)
	v.Set("theme", "aether.so")
	v.Set("themes_path", defaultThemesPath())

	if err := v.WriteConfigAs(path); err != nil {
		return fmt.Errorf("create config: %w", err)
	}

	return nil
}

func defaultThemesPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "Themes"
	}

	return filepath.Join(home, ".config", "DCGO", "Themes")
}
