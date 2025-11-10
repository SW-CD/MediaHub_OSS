// internal/config/config.go
package config

import (
	"github.com/BurntSushi/toml"
)

// Config holds the application's configuration.
type Config struct {
	Server   ServerConfig   `toml:"server"`
	Database DatabaseConfig `toml:"database"`
	Logging  LoggingConfig  `toml:"logging"`
	Media    MediaConfig    `toml:"media"`

	AdminPassword      string `toml:"-"` // Not loaded from file, set by CLI/env
	ResetAdminPassword bool   `toml:"-"` // Not loaded from file, set by CLI/env
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
	FFprobePath string `toml:"ffprobe_path"` // <-- ADDED
}

// LoadConfig loads the configuration from a TOML file.
func LoadConfig(path string) (*Config, error) {
	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
