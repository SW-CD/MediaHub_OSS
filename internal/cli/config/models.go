package config

import (
	"mediahub/internal/shared"
	"time"
)

// Config holds the application's configuration.
type Config struct {
	Server   serverConfigInternal `toml:"server"`
	Database DatabaseConfig       `toml:"database"`
	Logging  LoggingConfig        `toml:"logging"`
	Media    MediaConfig          `toml:"media"`
	JWT      jwtConfigInternal    `toml:"jwt"`
}

//--------------------
// No getter required for the following, as fields dont require type conversion
//--------------------

// databaseConfig holds the database configuration.
type DatabaseConfig struct {
	Path        string `toml:"path"`
	StorageRoot string `toml:"storage_root"`
}

// loggingConfig holds the logging configuration.
type LoggingConfig struct {
	Level        string `toml:"level"`
	AuditEnabled bool   `toml:"audit_enabled"` // <-- ADDED: Toggle for audit logs
}

// MediaConfig holds media processing settings.
type MediaConfig struct {
	FFmpegPath  string `toml:"ffmpeg_path"`
	FFprobePath string `toml:"ffprobe_path"`
}

//--------------------
// These are not public, as fields do require type conversion
//--------------------

// serverConfig holds the server configuration.
// not exported as maxSyncUploadSize needs converting
type serverConfigInternal struct {
	Host              string `toml:"host"`
	Port              int    `toml:"port"`
	MaxSyncUploadSize string `toml:"max_sync_upload_size"` // e.g. "8MB", "512KB"
}

// jwtConfig holds settings for token generation.
// not exported as the durations needs conversion
type jwtConfigInternal struct {
	AccessDuration  string `toml:"access_duration"`
	RefreshDuration string `toml:"refresh_duration"`
	Secret          string `toml:"secret"` // Persisted secret
}

// --------------------
// Return types of getters
// --------------------
type ServerConfig struct {
	Host              string
	Port              int
	MaxSyncUploadSize uint64 // threshold in bytes for "large file" treatment
}

type JWTConfig struct {
	AccessDuration  time.Duration
	RefreshDuration time.Duration
	Secret          string
}

// --------------------
// Getter functions for types that require field conversion to the public space
// --------------------
func (cfg *Config) GetServerConfig() (ServerConfig, error) {

	maxsyncsize_int, err := shared.ParseSize(cfg.Server.MaxSyncUploadSize)
	if err != nil {
		return ServerConfig{}, err
	}

	return ServerConfig{
		Host:              cfg.Server.Host,
		Port:              cfg.Server.Port,
		MaxSyncUploadSize: maxsyncsize_int,
	}, nil
}

func (cfg *Config) GetJWTConfig() (JWTConfig, error) {

	accessDuration, err := shared.ParseDuration(cfg.JWT.AccessDuration)
	if err != nil {
		return JWTConfig{}, err
	}

	refreshDuration, err := shared.ParseDuration(cfg.JWT.RefreshDuration)
	if err != nil {
		return JWTConfig{}, err
	}

	return JWTConfig{
		AccessDuration:  accessDuration,
		RefreshDuration: refreshDuration,
		Secret:          cfg.JWT.Secret,
	}, nil
}
