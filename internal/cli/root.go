// filepath: internal/cli/root.go
package cli

import (
	"context"
	"embed"
	"fmt"
	"mediahub/internal/api"
	"mediahub/internal/api/handlers"
	"mediahub/internal/audit"
	"mediahub/internal/config"
	"mediahub/internal/initconfig"
	"mediahub/internal/logging"
	"mediahub/internal/media"
	"mediahub/internal/repository"
	"mediahub/internal/services"
	"mediahub/internal/services/auth"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	// Version info
	Version   = "1.2.0"
	StartTime time.Time

	// Global config object populated by flags/env/file
	cfg *config.Config

	// Flags
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

// RootCmd represents the base command when called without any subcommands.
// It starts the HTTP server.
var RootCmd = &cobra.Command{
	Use:   "mediahub",
	Short: "MediaHub API & Web Interface",
	Long:  `A robust REST API and web frontend for storing and managing camera and microphone data.`,
	// PersistentPreRunE loads the configuration before any command runs.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initializeConfig(cmd)
	},
	// RunE executes the main server logic.
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServer()
	},
}

// frontendFS holds the embedded frontend assets.
var frontendFS embed.FS

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(fs embed.FS) {
	frontendFS = fs // Store for use in runServer
	StartTime = time.Now()

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Define flags.
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config_path", "config.toml", "Path to the base configuration file. (Env: FDB_CONFIG_PATH)")
	RootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "Logging level (debug, info, warn, error). (Env: FDB_LOG_LEVEL)")

	// Server-specific flags
	RootCmd.Flags().StringVar(&password, "password", "", "Password for the 'admin' user. (Env: FDB_PASSWORD)")
	RootCmd.Flags().IntVar(&port, "port", 0, "Port for the HTTP server. (Env: FDB_PORT)")
	RootCmd.Flags().BoolVar(&resetPassword, "reset_pw", false, "If true, reset admin password on startup. (Env: FDB_RESET_PW=true)")
	RootCmd.Flags().StringVar(&ffmpegPath, "ffmpeg-path", "", "Path to ffmpeg executable. (Env: FDB_FFMPEG_PATH)")
	RootCmd.Flags().StringVar(&ffprobePath, "ffprobe-path", "", "Path to ffprobe executable. (Env: FDB_FFPROBE_PATH)")
	RootCmd.Flags().StringVar(&jwtSecret, "jwt-secret", "", "Secret key for signing JWTs. (Env: FDB_JWT_SECRET)")
	RootCmd.Flags().StringVar(&maxSyncUpload, "max-sync-upload", "", "Max size for synchronous/in-memory uploads (e.g. '8MB'). (Env: FDB_MAX_SYNC_UPLOAD)")
	RootCmd.Flags().StringVar(&initConfig, "init_config", "", "Path to a TOML config file for one-time initialization of users/databases. (Env: FDB_INIT_CONFIG)")
	RootCmd.Flags().BoolVar(&auditEnabled, "audit-enabled", false, "Enable detailed audit logging. (Env: FDB_AUDIT_ENABLED=true)")
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
			// Create empty config if not found, rely on defaults/flags
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

	return nil
}

func applyOverrides(c *config.Config, cmd *cobra.Command) {
	// Helper to get string from env or fallback
	getEnv := func(key string) string {
		return os.Getenv(key)
	}

	// --- 1. Environment Variables ---
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

	// --- 2. CLI Flags (Take precedence) ---
	if password != "" {
		c.AdminPassword = password
	}
	if port != 0 {
		c.Server.Port = port
	}
	if logLevel != "" {
		c.Logging.Level = logLevel
	}
	// Check if flag was explicitly set
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

	// --- 3. Defaults ---
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

// runServer contains the logic to start the HTTP server with graceful shutdown.
func runServer() error {
	// Handle JWT Secret
	if cfg.JWTSecret == "" {
		if cfg.JWT.Secret != "" {
			logging.Log.Info("Using JWT secret loaded from config.toml.")
			cfg.JWTSecret = cfg.JWT.Secret
		} else {
			logging.Log.Info("Generating new random JWT secret...")
			newSecret, err := auth.GenerateSecret()
			if err != nil {
				return fmt.Errorf("failed to generate JWT secret: %w", err)
			}
			cfg.JWT.Secret = newSecret
			cfg.JWTSecret = newSecret
			if err := config.SaveConfig(cfgFile, cfg); err != nil {
				logging.Log.Warnf("Failed to save new JWT secret to %s: %v", cfgFile, err)
			} else {
				logging.Log.Infof("New JWT secret saved to %s.", cfgFile)
			}
		}
	}

	media.Initialize(cfg.Media.FFmpegPath, cfg.Media.FFprobePath)

	repo, err := repository.NewRepository(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}
	defer repo.Close()

	// --- Conditional Auto-migrate on startup ---
	if err := repo.EnsureSchemaBootstrapped(); err != nil {
		logging.Log.Errorf("Failed to bootstrap database: %v", err)
		return err
	}

	if err := repo.ValidateSchema(); err != nil {
		logging.Log.Error("---------------------------------------------------------------")
		logging.Log.Errorf("CRITICAL DATABASE ERROR: %v", err)
		logging.Log.Error("---------------------------------------------------------------")
		return err
	}

	// Service Initialization
	storageService := services.NewStorageService(cfg)
	infoService := services.NewInfoService(Version, StartTime, media.IsFFmpegAvailable(), media.IsFFprobeAvailable())
	userService := services.NewUserService(repo)
	tokenService := auth.NewTokenService(cfg, userService, repo)
	databaseService := services.NewDatabaseService(repo, storageService)
	entryService := services.NewEntryService(repo, storageService, cfg)
	housekeepingService := services.NewHousekeepingService(repo, storageService)

	// Auditor Initialization
	loggerAuditor := audit.NewLoggerAuditor(cfg.Logging.AuditEnabled)

	authMiddleware := auth.NewMiddleware(userService, tokenService)

	if err := userService.InitializeAdminUser(cfg); err != nil {
		return fmt.Errorf("failed to handle admin user: %w", err)
	}

	if initConfig != "" {
		logging.Log.Infof("Found init_config, running initialization from: %s", initConfig)
		initconfig.Run(userService, databaseService, initConfig)
	}

	housekeepingService.Start()
	// No defer stop here, we stop explicitly during graceful shutdown

	h := handlers.NewHandlers(
		infoService,
		userService,
		tokenService,
		databaseService,
		entryService,
		housekeepingService,
		loggerAuditor,
		cfg,
	)

	r := api.SetupRouter(h, authMiddleware, cfg, frontendFS)

	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    serverAddr,
		Handler: r,
	}

	// --- Graceful Shutdown Setup ---
	// Create a channel to listen for OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Run server in a goroutine
	go func() {
		logging.Log.Infof("Server starting on %s (Max Sync Upload: %s)", serverAddr, cfg.Server.MaxSyncUploadSize)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Block until a signal is received
	<-stop
	logging.Log.Info("Shutting down server...")

	// Create a deadline for existing requests to complete (30 seconds)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop background services
	housekeepingService.Stop()

	// Shutdown the HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		logging.Log.Errorf("Server forced to shutdown: %v", err)
		return err
	}

	logging.Log.Info("Server exiting")
	return nil
}
