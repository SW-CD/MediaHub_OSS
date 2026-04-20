package cli

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"mediahub_oss/docs" // to get the version
	"mediahub_oss/internal/cli/config"
	"mediahub_oss/internal/cli/initconfig"
	"mediahub_oss/internal/housekeeping"
	"mediahub_oss/internal/httpserver"
	ah "mediahub_oss/internal/httpserver/audithandler"
	"mediahub_oss/internal/httpserver/auth"
	dbh "mediahub_oss/internal/httpserver/databasehandler"
	eh "mediahub_oss/internal/httpserver/entryhandler"
	ih "mediahub_oss/internal/httpserver/infohandler"
	th "mediahub_oss/internal/httpserver/tokenhandler"
	uh "mediahub_oss/internal/httpserver/userhandler"
	"mediahub_oss/internal/logging/audit"
	"mediahub_oss/internal/media/ffmpeg"
	"mediahub_oss/internal/repository"
	"mediahub_oss/internal/repository/postgres"
	"mediahub_oss/internal/repository/sqlite"
	"mediahub_oss/internal/shared"
	"mediahub_oss/internal/storage"
	"mediahub_oss/internal/storage/localstorage"
	"mediahub_oss/internal/storage/s3storage"
	"time"

	// Aliased imports for your sub-handlers

	"net/http"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func NewServeCommand(globalOptions *GlobalOptions, frontendFS fs.FS) *cobra.Command {

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return serve(globalOptions, frontendFS)
		},
	}

	registerFlags(serveCmd)

	return serveCmd
}

func registerFlags(cmd *cobra.Command) {
	// Operational & Startup Flags
	cmd.Flags().String("init_config", "", "Path to a TOML config file for one-time initialization.")
	cmd.Flags().String("password", "", "Password for the 'admin' user.")
	cmd.Flags().Bool("reset_pw", false, "If true, reset admin password on startup.")

	// Server Settings
	cmd.Flags().String("server-host", "0.0.0.0", "The host address to bind to.")
	cmd.Flags().Int("server-port", 8080, "The HTTP port to bind to.")
	cmd.Flags().String("server-basepath", "/", "The base path for reverse proxy.")
	cmd.Flags().String("server-max-sync-upload", "4MB", "RAM threshold for uploads.")
	cmd.Flags().StringSlice("server-cors-origins", []string{}, "Allowed CORS origins.")

	// Database Settings
	cmd.Flags().String("database-driver", "sqlite", "Database driver (sqlite or postgres).")
	cmd.Flags().String("database-source", "mediahub.db", "Path to DB file or connection string.")
	cmd.Flags().Int("database-max-open-conns", 25, "PostgreSQL max open connections.")
	cmd.Flags().Int("database-max-idle-conns", 25, "PostgreSQL max idle connections.")

	// Storage Settings
	cmd.Flags().String("storage-type", "local", "Storage backend type (local or s3).")
	cmd.Flags().String("storage-local-root", "storage_root", "Root directory for local storage.")
	cmd.Flags().String("storage-s3-endpoint", "", "S3 API endpoint.")
	cmd.Flags().String("storage-s3-region", "", "S3 region.")
	cmd.Flags().String("storage-s3-bucket", "", "S3 bucket name.")
	cmd.Flags().String("storage-s3-access-key", "", "S3 Access Key.")
	cmd.Flags().String("storage-s3-secret-key", "", "S3 Secret Key.")
	cmd.Flags().Bool("storage-s3-use-ssl", true, "Enable HTTPS for S3 connection.")

	// Logging Settings
	cmd.Flags().String("logging-level", "info", "Logging verbosity.")
	cmd.Flags().String("logging-audit-type", "stdio", "Where to store audit logs.")
	cmd.Flags().Bool("logging-audit-enabled", false, "Toggle audit logging.")
	cmd.Flags().String("logging-audit-retention", "31d", "How long to keep audit logs.")

	// Media Settings
	cmd.Flags().String("media-ffmpeg-path", "", "Path to FFmpeg executable.")
	cmd.Flags().String("media-ffprobe-path", "", "Path to FFprobe executable.")

	// Auth Settings
	cmd.Flags().String("auth-jwt-access-duration", "5min", "Validity of the JWT.")
	cmd.Flags().String("auth-jwt-refresh-duration", "24h", "Validity of the refresh token.")
	cmd.Flags().String("auth-jwt-secret", "", "Secret key for signing JWTs.")
	cmd.Flags().Bool("auth-oidc-enabled", false, "Toggle OIDC integration.")
	cmd.Flags().Bool("auth-oidc-disable-local-login", false, "Disable internal local login.")
	cmd.Flags().String("auth-oidc-default-user-rights", "_oidc_user", "Default rights for new OIDC users.")
	cmd.Flags().String("auth-oidc-issuer-url", "", "OIDC Issuer URL.")
	cmd.Flags().String("auth-oidc-client-id", "", "OIDC Client ID.")
	cmd.Flags().String("auth-oidc-client-secret", "", "OIDC Client Secret.")
	cmd.Flags().String("auth-oidc-redirect-url", "", "OIDC Redirect callback URL.")

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Convert standard flag "server-port" into Viper's nested format "server.port"
		viperKey := strings.ReplaceAll(f.Name, "-", ".")
		viper.BindPFlag(viperKey, f)
	})
}

func serve(globalOptions *GlobalOptions, frontendFS fs.FS) error {
	// Capture the start time for the InfoHandler's uptime calculation
	startTime := time.Now()

	// 1. Prepare final configuration
	cfg := globalOptions.Conf
	logger := globalOptions.Logger
	ctx := context.Background()

	logger.Info("Bootstrapping MediaHub server...")

	// 2. Initialize Repository (Database)
	repo, err := initRepository(cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}
	// Because we return errors now, this defer will ALWAYS execute safely!
	defer repo.Close()

	// --- Safe Auto-Migration for Fresh Installs ---
	if err := handleInitialMigration(ctx, repo, logger); err != nil {
		return fmt.Errorf("failed to verify or apply database schema: %w", err)
	}

	// Admin User Startup Logic ---
	if err := EnsureAdminUser(ctx, repo, logger); err != nil {
		return fmt.Errorf("admin user initialization failed: %w", err)
	}

	// 3. Initialize Storage Provider
	storageProvider, err := initStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to initialize storage provider: %w", err)
	}

	// Parse and apply initconfig
	if err := processInitConfig(ctx, repo, logger); err != nil {
		return fmt.Errorf("failed to process initialization config: %w", err)
	}

	// 4. Initialize Core Background Services
	auditRetention, err := shared.ParseDuration(cfg.Logging.Audit.Retention)
	if err != nil {
		return fmt.Errorf("failed to parse audit retention duration: %w", err)
	}

	hk := housekeeping.NewHouseKeeper(repo, storageProvider, logger, auditRetention)
	go hk.StartScheduler(ctx)

	converter, err := ffmpeg.NewFFMPEGConverter(cfg.Media.FFmpegPath, cfg.Media.FFprobePath, logger)
	if err != nil {
		return fmt.Errorf("failed to start media converter: %w", err)
	}
	auditLogger := audit.NewAuditLogger(cfg.Logging.Audit.Enabled, cfg.Logging.Audit.Type, logger, repo)
	authMiddleware := auth.NewAuthMiddleware(repo, cfg.Auth.JWT.Secret)

	// Extract specific configs for handlers
	serverCfg, err := cfg.GetServerConfig()
	if err != nil {
		return fmt.Errorf("failed to parse server config: %w", err)
	}

	jwtCfg, err := cfg.GetJWTConfig()
	if err != nil {
		return fmt.Errorf("failed to parse JWT config: %w", err)
	}

	// 5. Build Handlers Struct (Dependency Injection)
	infoH := ih.NewInfoHandler(
		logger,
		auditLogger,
		docs.SwaggerInfo.Version,
		converter,
		cfg.Auth.OIDC.Enabled,
		cfg.Auth.OIDC.DisableLoginPage,
		cfg.Auth.OIDC.IssuerURL,
		cfg.Auth.OIDC.ClientID,
		cfg.Auth.OIDC.RedirectURL,
	)
	// Preserve the precise boot time captured at the top of the serve command
	infoH.StartTime = startTime

	handlers := &httpserver.Handlers{
		InfoHandler: *infoH,
		EntryHandler: eh.EntryHandler{
			Logger:                 logger,
			Auditor:                auditLogger,
			Repo:                   repo,
			Storage:                storageProvider,
			MaxSyncUploadSizeBytes: int64(serverCfg.MaxSyncUploadSize),
			MediaConverter:         converter,
		},
		DatabaseHandler: dbh.DatabaseHandler{
			Logger:      logger,
			Auditor:     auditLogger,
			Repo:        repo,
			HouseKeeper: *hk,
		},
		UserHandler: uh.UserHandler{
			Logger:  logger,
			Auditor: auditLogger,
			Repo:    repo,
		},
		TokenHandler: th.TokenHandler{
			Logger:          logger,
			Auditor:         auditLogger,
			Repo:            repo,
			JWTSecret:       []byte(jwtCfg.Secret),
			AccessDuration:  jwtCfg.AccessDuration,
			RefreshDuration: jwtCfg.RefreshDuration,
		},
		AuditHandler: ah.AuditHandler{
			Logger: logger,
			Repo:   repo,
		},
	}

	// 6. Setup Router
	var fileSystem http.FileSystem
	if frontendFS != nil {
		// TODO update <base href> to the MEDIAHUB_SERVER_BASEPATH
		// or we do it later in the SetupRouter?
		fileSystem = http.FS(frontendFS)
	}

	mux := httpserver.SetupRouter(handlers, fileSystem, authMiddleware, cfg.Server.Basepath, cfg.Server.CorsAllowedOrigins)

	// 7. Start HTTP Server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Info("Starting HTTP server", "address", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}

// handleInitialMigration checks the database version and only auto-migrates if it is a completely fresh installation (version 0).
func handleInitialMigration(ctx context.Context, repo repository.Repository, logger *slog.Logger) error {
	version, err := repo.GetMigrationVersion(ctx)
	if err != nil {
		return fmt.Errorf("could not determine database version: %w", err)
	}

	if version == 0 {
		logger.Info("Fresh database installation detected. Automatically applying initial schema...")
		if err := repo.MigrateUp(ctx); err != nil {
			return fmt.Errorf("initial migration failed: %w", err)
		}
		logger.Info("Initial database schema applied successfully.")
	} else {
		logger.Info("Existing database detected.", "version", repository.FormatVersion(version))
	}

	return nil
}

// initRepository sets up the database connection based on the configuration.
func initRepository(dbCfg config.DatabaseConfig) (repository.Repository, error) {
	switch dbCfg.Driver {
	case "sqlite":
		return sqlite.NewRepository(dbCfg.Source)
	case "postgres":
		return postgres.NewRepository(dbCfg.Source)
	default:
		return nil, fmt.Errorf("unsupported database driver, must be `sqlite` or `postgres`: %s", dbCfg.Driver)
	}
}

// initStorage sets up the file storage provider based on the configuration.
func initStorage(storageCfg config.StorageConfig) (storage.StorageProvider, error) {
	switch storageCfg.Type {
	case "local":
		return &localstorage.LocalStorage{RootPath: storageCfg.Local.Root}, nil
	case "s3":
		s3prov, err := s3storage.NewS3StorageProvider()
		if err != nil {
			return nil, err
		}
		return &s3prov, nil
	default:
		return nil, fmt.Errorf("unsupported storage type, must be `local` or `s3`: %s", storageCfg.Type)
	}
}

// processInitConfig checks for the init_config flag and applies the one-time configuration if present.
func processInitConfig(ctx context.Context, repo repository.Repository, logger *slog.Logger) error {
	initConfPath := viper.GetString("init_config")
	if initConfPath == "" {
		return nil // No init config provided, skip gracefully
	}

	logger.Info("Found init_config flag, attempting to apply initialization data", "path", initConfPath)
	initConfigData, err := initconfig.ParseInitConfig(initConfPath)
	if err != nil {
		return fmt.Errorf("failed to parse init config: %w", err)
	}

	// Apply the configuration to the database
	if err := initconfig.Apply(ctx, &initConfigData, repo, logger, initConfPath); err != nil {
		return fmt.Errorf("failed to apply init config: %w", err)
	}

	return nil
}
