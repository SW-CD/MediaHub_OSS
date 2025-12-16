// filepath: internal/cli/server.go
package cli

import (
	"context"
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
	"syscall"
	"time"
)

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
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logging.Log.Infof("Server starting on %s (Max Sync Upload: %s)", serverAddr, cfg.Server.MaxSyncUploadSize)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Log.Fatalf("Server failed to start: %v", err)
		}
	}()

	<-stop
	logging.Log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	housekeepingService.Stop()

	if err := srv.Shutdown(ctx); err != nil {
		logging.Log.Errorf("Server forced to shutdown: %v", err)
		return err
	}

	logging.Log.Info("Server exiting")
	return nil
}
