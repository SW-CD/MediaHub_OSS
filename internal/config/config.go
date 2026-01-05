// filepath: internal/config/config.go
package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

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

	MaxSyncUploadSizeBytes int64 `toml:"-"` // Runtime computed value
}

// ServerConfig holds the server configuration.
type ServerConfig struct {
	Host              string `toml:"host"`
	Port              int    `toml:"port"`
	MaxSyncUploadSize string `toml:"max_sync_upload_size"` // e.g. "8MB", "512KB"
}

// DatabaseConfig holds the database configuration.
type DatabaseConfig struct {
	Path        string `toml:"path"`
	StorageRoot string `toml:"storage_root"`
}

// LoggingConfig holds the logging configuration.
type LoggingConfig struct {
	Level        string `toml:"level"`
	AuditEnabled bool   `toml:"audit_enabled"` // <-- ADDED: Toggle for audit logs
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

// ParseAndValidate processes configuration strings into runtime values.
// It sets defaults if values are missing and parses human-readable sizes.
func (c *Config) ParseAndValidate() error {
	// Default MaxSyncUploadSize to 8MB if not specified
	if c.Server.MaxSyncUploadSize == "" {
		c.Server.MaxSyncUploadSize = "8MB"
	}

	sizeBytes, err := parseSize(c.Server.MaxSyncUploadSize)
	if err != nil {
		return fmt.Errorf("invalid max_sync_upload_size: %w", err)
	}
	c.MaxSyncUploadSizeBytes = sizeBytes

	return nil
}

// parseSize parses a size string (e.g., "100G", "500MB") into bytes.
// Duplicated here to keep the config package self-contained and dependency-free.
func parseSize(sizeStr string) (int64, error) {
	re := regexp.MustCompile(`(?i)^(\d+)\s*(K|M|G|T)?B?$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(sizeStr))

	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size number: %s", matches[1])
	}

	unit := ""
	if len(matches) > 2 {
		unit = strings.ToUpper(matches[2])
	}

	switch unit {
	case "T":
		return value * (1 << 40), nil
	case "G":
		return value * (1 << 30), nil
	case "M":
		return value * (1 << 20), nil
	case "K":
		return value * (1 << 10), nil
	default:
		return value, nil
	}
}
