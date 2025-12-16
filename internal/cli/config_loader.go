// filepath: internal/cli/config_loader.go
package cli

import (
	"fmt"
	"mediahub/internal/config"
	"mediahub/internal/logging"
	"os"
	"strconv"

	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

var (
	// Global config object populated by flags/env/file
	cfg *config.Config

	// Flags variables
	cfgFile       string
	password      string
	port          int
	logLevel      string
	resetPassword bool
	ffmpegPath    string
	ffprobePath   string
	jwtSecret     string
	maxSyncUpload string
	initConfig    string
	auditEnabled  bool
)

func registerFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&cfgFile, "config_path", "config.toml", "Path to the base configuration file. (Env: FDB_CONFIG_PATH)")
	cmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "Logging level (debug, info, warn, error). (Env: FDB_LOG_LEVEL)")

	// Server-specific flags
	cmd.Flags().StringVar(&password, "password", "", "Password for the 'admin' user. (Env: FDB_PASSWORD)")
	cmd.Flags().IntVar(&port, "port", 0, "Port for the HTTP server. (Env: FDB_PORT)")
	cmd.Flags().BoolVar(&resetPassword, "reset_pw", false, "If true, reset admin password on startup. (Env: FDB_RESET_PW=true)")
	cmd.Flags().StringVar(&ffmpegPath, "ffmpeg-path", "", "Path to ffmpeg executable. (Env: FDB_FFMPEG_PATH)")
	cmd.Flags().StringVar(&ffprobePath, "ffprobe-path", "", "Path to ffprobe executable. (Env: FDB_FFPROBE_PATH)")
	cmd.Flags().StringVar(&jwtSecret, "jwt-secret", "", "Secret key for signing JWTs. (Env: FDB_JWT_SECRET)")
	cmd.Flags().StringVar(&maxSyncUpload, "max-sync-upload", "", "Max size for synchronous/in-memory uploads (e.g. '8MB'). (Env: FDB_MAX_SYNC_UPLOAD)")
	cmd.Flags().StringVar(&initConfig, "init_config", "", "Path to a TOML config file for one-time initialization of users/databases. (Env: FDB_INIT_CONFIG)")
	cmd.Flags().BoolVar(&auditEnabled, "audit-enabled", false, "Enable detailed audit logging. (Env: FDB_AUDIT_ENABLED=true)")
}

// initializeConfig loads and overrides configuration values.
func initializeConfig(cmd *cobra.Command) error {
	// 1. Check environment variable for config path first
	if envPath := os.Getenv("FDB_CONFIG_PATH"); envPath != "" && cfgFile == "config.toml" {
		cfgFile = envPath
	}

	var err error
	cfg, err = config.LoadConfig(cfgFile)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = &config.Config{}
		} else {
			return fmt.Errorf("failed to load configuration from %s: %w", cfgFile, err)
		}
	}

	// 2. Apply Overrides (Env Vars and CLI Flags)
	applyOverrides(cfg, cmd)

	// 3. Validate
	if err := cfg.ParseAndValidate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// 4. Initialize Logging
	logging.Init(cfg.Logging.Level)
	goose.SetLogger(logging.Log)

	return nil
}

func applyOverrides(c *config.Config, cmd *cobra.Command) {
	getEnv := func(key string) string { return os.Getenv(key) }

	// --- Environment Variables ---
	if v := getEnv("FDB_PASSWORD"); v != "" {
		c.AdminPassword = v
	}
	if v := getEnv("FDB_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Server.Port = p
		}
	}
	if v := getEnv("FDB_LOG_LEVEL"); v != "" {
		c.Logging.Level = v
	}
	if v := getEnv("FDB_AUDIT_ENABLED"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.Logging.AuditEnabled = b
		}
	}
	if v := getEnv("FDB_RESET_PW"); v == "true" {
		c.ResetAdminPassword = true
	}
	if v := getEnv("FDB_DATABASE_PATH"); v != "" {
		c.Database.Path = v
	}
	if v := getEnv("FDB_STORAGE_ROOT"); v != "" {
		c.Database.StorageRoot = v
	}
	if v := getEnv("FDB_FFMPEG_PATH"); v != "" {
		c.Media.FFmpegPath = v
	}
	if v := getEnv("FDB_FFPROBE_PATH"); v != "" {
		c.Media.FFprobePath = v
	}
	if v := getEnv("FDB_JWT_SECRET"); v != "" {
		c.JWTSecret = v
	}
	if v := getEnv("FDB_MAX_SYNC_UPLOAD"); v != "" {
		c.Server.MaxSyncUploadSize = v
	}

	// --- CLI Flags ---
	if password != "" {
		c.AdminPassword = password
	}
	if port != 0 {
		c.Server.Port = port
	}
	if logLevel != "" {
		c.Logging.Level = logLevel
	}
	if cmd.Flags().Changed("audit-enabled") {
		c.Logging.AuditEnabled = auditEnabled
	}
	if resetPassword {
		c.ResetAdminPassword = true
	}
	if ffmpegPath != "" {
		c.Media.FFmpegPath = ffmpegPath
	}
	if ffprobePath != "" {
		c.Media.FFprobePath = ffprobePath
	}
	if jwtSecret != "" {
		c.JWTSecret = jwtSecret
	}
	if maxSyncUpload != "" {
		c.Server.MaxSyncUploadSize = maxSyncUpload
	}
	if initConfig == "" {
		if v := getEnv("FDB_INIT_CONFIG"); v != "" {
			initConfig = v
		}
	}

	// --- Defaults ---
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Database.Path == "" {
		c.Database.Path = "mediahub.db"
	}
	if c.Database.StorageRoot == "" {
		c.Database.StorageRoot = "storage_root"
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.JWT.AccessDurationMin == 0 {
		c.JWT.AccessDurationMin = 5
	}
	if c.JWT.RefreshDurationHours == 0 {
		c.JWT.RefreshDurationHours = 24
	}
}
