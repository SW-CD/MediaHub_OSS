package config

// Config holds the application's configuration.
type Config struct {
	Server   serverConfig   `toml:"server"`
	Database DatabaseConfig `toml:"database"`
	Logging  LoggingConfig  `toml:"logging"`
	Media    MediaConfig    `toml:"media"`
	JWT      jwtConfig      `toml:"jwt"`
}

// serverConfig holds the server configuration.
// not exported as maxSyncUploadSize needs converting
type serverConfig struct {
	host              string `toml:"host"`
	port              int    `toml:"port"`
	maxSyncUploadSize string `toml:"max_sync_upload_size"` // e.g. "8MB", "512KB"
}

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

// jwtConfig holds settings for token generation.
// not exported as the durations needs conversion
type jwtConfig struct {
	accessDurationMin    int    `toml:"access_duration_min"`
	refreshDurationHours int    `toml:"refresh_duration_hours"`
	secret               string `toml:"secret"` // Persisted secret
}
