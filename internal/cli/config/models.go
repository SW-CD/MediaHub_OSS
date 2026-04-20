package config

import (
	"mediahub_oss/internal/shared"
	"time"
)

// Config holds the application's configuration.
type Config struct {
	Server   serverConfigInternal `toml:"server" mapstructure:"server"`
	Database DatabaseConfig       `toml:"database" mapstructure:"database"`
	Storage  StorageConfig        `toml:"storage" mapstructure:"storage"`
	Logging  LoggingConfig        `toml:"logging" mapstructure:"logging"`
	Media    MediaConfig          `toml:"media" mapstructure:"media"`
	Auth     AuthConfig           `toml:"auth" mapstructure:"auth"`
}

//--------------------
// Public Structs (No conversion required)
//--------------------

// DatabaseConfig holds the database connection settings.
type DatabaseConfig struct {
	Driver       string `toml:"driver" mapstructure:"driver"`
	Source       string `toml:"source" mapstructure:"source"`
	MaxOpenConns int    `toml:"max_open_conns" mapstructure:"max_open_conns"`
	MaxIdleConns int    `toml:"max_idle_conns" mapstructure:"max_idle_conns"`
}

// StorageConfig holds settings for file storage.
type StorageConfig struct {
	Type  string      `toml:"type" mapstructure:"type"` // "local" or "s3
	Local LocalConfig `toml:"local" mapstructure:"local"`
	S3    S3Config    `toml:"s3" mapstructure:"s3"`
}

type LocalConfig struct {
	Root string `toml:"root" mapstructure:"root"`
}

type S3Config struct {
	Endpoint  string `toml:"endpoint" mapstructure:"endpoint"`
	Region    string `toml:"region" mapstructure:"region"`
	Bucket    string `toml:"bucket" mapstructure:"bucket"`
	AccessKey string `toml:"access_key" mapstructure:"access_key"`
	SecretKey string `toml:"secret_key" mapstructure:"secret_key"`
	UseSSL    bool   `toml:"use_ssl" mapstructure:"use_ssl"`
}

// LoggingConfig holds application logging settings.
type LoggingConfig struct {
	Level string      `toml:"level" mapstructure:"level"`
	Audit AuditConfig `toml:"audit" mapstructure:"audit"`
}

type AuditConfig struct {
	Type      string `toml:"type" mapstructure:"type"` // "stdio" or "database
	Enabled   bool   `toml:"enabled" mapstructure:"enabled"`
	Retention string `toml:"retention" mapstructure:"retention"` // How long to keep audit logs (e.g., "7d" for 7 days)
}

// MediaConfig holds media processing settings.
type MediaConfig struct {
	FFmpegPath  string `toml:"ffmpeg_path" mapstructure:"ffmpeg_path"`
	FFprobePath string `toml:"ffprobe_path" mapstructure:"ffprobe_path"`
}

//--------------------
// Internal Structs (Fields require conversion)
//--------------------

type serverConfigInternal struct {
	Host               string   `toml:"host" mapstructure:"host"`
	Port               int      `toml:"port" mapstructure:"port"`
	Basepath           string   `toml:"basepath" mapstructure:"basepath"`
	MaxSyncUploadSize  string   `toml:"max_sync_upload_size" mapstructure:"max_sync_upload_size"`
	CorsAllowedOrigins []string `toml:"cors_allowed_origins" mapstructure:"cors_allowed_origins"`
}

type AuthConfig struct {
	OIDC oidcConfigInternal `toml:"oidc" mapstructure:"oidc"`
	JWT  jwtConfigInternal  `toml:"jwt" mapstructure:"jwt"`
}

type oidcConfigInternal struct {
	Enabled           bool   `toml:"enabled" mapstructure:"enabled"`
	DisableLoginPage  bool   `toml:"disable_login_page" mapstructure:"disable_login_page"`
	DefaultUserRights string `toml:"default_user_rights" mapstructure:"default_user_rights"`
	IssuerURL         string `toml:"issuer_url" mapstructure:"issuer_url"`
	ClientID          string `toml:"client_id" mapstructure:"client_id"`
	ClientSecret      string `toml:"client_secret" mapstructure:"client_secret"`
	RedirectURL       string `toml:"redirect_url" mapstructure:"redirect_url"`
}

type jwtConfigInternal struct {
	AccessDuration  string `toml:"access_duration" mapstructure:"access_duration"`
	RefreshDuration string `toml:"refresh_duration" mapstructure:"refresh_duration"`
	Secret          string `toml:"secret" mapstructure:"secret"`
}

// --------------------
// Converted Public Structs (Return types of getters)
// --------------------

type ServerConfig struct {
	Host               string
	Port               int
	Basepath           string
	MaxSyncUploadSize  uint64 // Threshold in bytes
	CorsAllowedOrigins []string
}

type JWTConfig struct {
	AccessDuration  time.Duration
	RefreshDuration time.Duration
	Secret          string
}

// --------------------
// Getter functions
// --------------------

func (cfg *Config) GetServerConfig() (ServerConfig, error) {
	maxsyncsize_int, err := shared.ParseSize(cfg.Server.MaxSyncUploadSize)
	if err != nil {
		return ServerConfig{}, err
	}

	return ServerConfig{
		Host:               cfg.Server.Host,
		Port:               cfg.Server.Port,
		Basepath:           cfg.Server.Basepath,
		MaxSyncUploadSize:  maxsyncsize_int,
		CorsAllowedOrigins: cfg.Server.CorsAllowedOrigins,
	}, nil
}

func (cfg *Config) GetJWTConfig() (JWTConfig, error) {
	accessDuration, err := shared.ParseDuration(cfg.Auth.JWT.AccessDuration)
	if err != nil {
		return JWTConfig{}, err
	}

	refreshDuration, err := shared.ParseDuration(cfg.Auth.JWT.RefreshDuration)
	if err != nil {
		return JWTConfig{}, err
	}

	return JWTConfig{
		AccessDuration:  accessDuration,
		RefreshDuration: refreshDuration,
		Secret:          cfg.Auth.JWT.Secret,
	}, nil
}
