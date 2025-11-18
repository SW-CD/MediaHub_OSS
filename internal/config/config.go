// internal/config/config.go
package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config holds the application's configuration.
type Config struct {
	Server   ServerConfig   `toml:"server"`
	Database DatabaseConfig `toml:"database"`
	Logging  LoggingConfig  `toml:"logging"`
	Media    MediaConfig    `toml:"media"`
	JWT      JWTConfig      `toml:"jwt"`

	AdminPassword      string `toml:"-"` // Not loaded from file, set by CLI/env
	ResetAdminPassword bool   `toml:"-"` // Not loaded from file, set by CLI/env
	JWTSecret          string `toml:"-"` // Runtime secret (from env, flag, or file)
}

// ServerConfig holds the server configuration.
type ServerConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

// DatabaseConfig holds the database configuration.
type DatabaseConfig struct {
	Path        string `toml:"path"`
	StorageRoot string `toml:"storage_root"`
}

// LoggingConfig holds the logging configuration.
type LoggingConfig struct {
	Level string `toml:"level"`
}

// MediaConfig holds media processing settings.
type MediaConfig struct {
	FFmpegPath  string `toml:"ffmpeg_path"`
	FFprobePath string `toml:"ffprobe_path"`
}

// JWTConfig holds settings for token generation.
type JWTConfig struct {
	AccessDurationMin    int    `toml:"access_duration_min"`
	RefreshDurationHours int    `toml:"refresh_duration_hours"`
	Secret               string `toml:"secret"` // Persisted secret
}

// LoadConfig loads the configuration from a TOML file.
func LoadConfig(path string) (*Config, error) {
	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// SaveConfig writes the current configuration back to a TOML file.
// Used to persist the auto-generated JWT secret.
func SaveConfig(path string, cfg *Config) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file for saving: %w", err)
	}
	defer f.Close()
	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config to file: %w", err)
	}
	return nil
}
