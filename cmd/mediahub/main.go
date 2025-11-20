// filepath: cmd/mediahub/main.go
package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"mediahub/internal/api"
	"mediahub/internal/api/handlers"
	"mediahub/internal/config"
	"mediahub/internal/initconfig"
	"mediahub/internal/logging"
	"mediahub/internal/media"
	"mediahub/internal/repository"
	"mediahub/internal/services"
	"mediahub/internal/services/auth"

	_ "mediahub/docs"
)

// CLIConfig holds the command-line argument values.
type CLIConfig struct {
	Password      string
	Port          int
	LogLevel      string
	ResetPassword bool
	ConfigPath    string
	FFmpegPath    string
	FFprobePath   string
	JWTSecret     string
}

//go:embed all:frontend_embed/browser
var frontendFS embed.FS

// @title SWCD MediaHub-API
// @version 1.0.0
// @description This is a sample server for a file store.
// @contact.name Christian Dengler
// @contact.url https://www.swcd.lu
// @contact.email denglerchr@gmail.com
// @BasePath /api
// @schemes http
// @securityDefinitions.basic BasicAuth
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and a JWT token.
// @import encoding/json

// version holds the current application version.
const version = "1.0.0"

// startTime holds the time the application was started.
var startTime time.Time

// customUsage prints a detailed help message.
func customUsage() {
	fmt.Fprintf(os.Stderr, "FileStore API v%s - https://www.swcd.lu\n", version)
	fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Configuration (flags override environment variables, which override config.toml):\n\n")
	flag.PrintDefaults() // Print the default flag help
	fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
	fmt.Fprintf(os.Stderr, "  FDB_PORT\t\t(see --port)\n")
	fmt.Fprintf(os.Stderr, "  FDB_LOG_LEVEL\t\t(see --log-level)\n")
	fmt.Fprintf(os.Stderr, "  FDB_PASSWORD\t\t(see --password)\n")
	fmt.Fprintf(os.Stderr, "  FDB_RESET_PW\t\t(see --reset_pw, use 'true')\n")
	fmt.Fprintf(os.Stderr, "  FDB_DATABASE_PATH\t(Overrides 'database.path' in config.toml)\n")
	fmt.Fprintf(os.Stderr, "  FDB_STORAGE_ROOT\t(Overrides 'database.storage_root' in config.toml)\n")
	fmt.Fprintf(os.Stderr, "  FDB_CONFIG_PATH\t(see --config_path)\n")
	fmt.Fprintf(os.Stderr, "  FDB_FFMPEG_PATH\t(see --ffmpeg-path)\n")
	fmt.Fprintf(os.Stderr, "  FDB_FFPROBE_PATH\t(see --ffprobe-path)\n")
	fmt.Fprintf(os.Stderr, "  FDB_INIT_CONFIG\t(see --init_config)\n")
	fmt.Fprintf(os.Stderr, "  FDB_JWT_SECRET\t(see --jwt-secret)\n")
}

func main() {
	startTime = time.Now() // Record start time

	// --- Set config_path precedence correctly ---
	configPathDefault := "config.toml"

	if envConfigPath := os.Getenv("FDB_CONFIG_PATH"); envConfigPath != "" {
		configPathDefault = envConfigPath
	}

	// --- Set init_config precedence correctly ---
	initConfigPathDefault := ""

	if envInitConfigPath := os.Getenv("FDB_INIT_CONFIG"); envInitConfigPath != "" {
		initConfigPathDefault = envInitConfigPath
	}

	// 1. Parse CLI flags
	var cliCfg CLIConfig
	flag.StringVar(&cliCfg.Password, "password", "", "Password for the 'admin' user. (Env: FDB_PASSWORD)")
	flag.IntVar(&cliCfg.Port, "port", 0, "Port for the HTTP server. (Env: FDB_PORT)")
	flag.StringVar(&cliCfg.LogLevel, "log-level", "", "Logging level (debug, info, warn, error). (Env: FDB_LOG_LEVEL)")
	flag.BoolVar(&cliCfg.ResetPassword, "reset_pw", false, "If true, reset admin password on startup. (Env: FDB_RESET_PW=true)")
	flag.StringVar(&cliCfg.ConfigPath, "config_path", configPathDefault, "Path to the base configuration file. (Env: FDB_CONFIG_PATH)")
	flag.StringVar(&cliCfg.FFmpegPath, "ffmpeg-path", "", "Path to ffmpeg executable. (Env: FDB_FFMPEG_PATH)")
	flag.StringVar(&cliCfg.FFprobePath, "ffprobe-path", "", "Path to ffprobe executable. (Env: FDB_FFPROBE_PATH)")
	flag.StringVar(&cliCfg.JWTSecret, "jwt-secret", "", "Secret key for signing JWTs. (Env: FDB_JWT_SECRET)")

	var initConfigPath string // This variable will be filled by the flag
	flag.StringVar(&initConfigPath, "init_config", initConfigPathDefault, "Path to a TOML config file for one-time initialization of users/databases. (Env: FDB_INIT_CONFIG)")

	flag.Usage = customUsage

	flag.Parse()

	// 2. Load configuration from file
	cfg, err := config.LoadConfig(cliCfg.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			logging.Log.Warnf("Config file not found at %s. Creating a new one with defaults.", cliCfg.ConfigPath)
			cfg = &config.Config{} // Create empty config
		} else {
			// Handle other errors (e.g., permission, malformed TOML)
			isHelp := false
			for _, arg := range os.Args {
				if arg == "-h" || arg == "--help" || arg == "help" {
					isHelp = true
					break
				}
			}
			if !isHelp {
				fmt.Printf("Failed to load configuration from %s: %v\n", cliCfg.ConfigPath, err)
				os.Exit(1)
			} else {
				return
			}
		}
	}

	// 3. Override with CLI flags and environment variables
	overrideConfigFromEnvAndCLI(&cliCfg, cfg)

	// JWT Secret Initialization Logic
	// Priority 1: Use secret from flag/env
	if cfg.JWTSecret == "" {
		// Priority 2: Use secret from config.toml
		if cfg.JWT.Secret != "" {
			logging.Log.Info("Using JWT secret loaded from config.toml.")
			cfg.JWTSecret = cfg.JWT.Secret
		} else {
			// Priority 3: Generate, save, and use a new secret
			logging.Log.Info("No JWT secret found in flags, env, or config. Generating a new random secret...")
			newSecretBytes := make([]byte, 32) // 256 bits
			if _, err := rand.Read(newSecretBytes); err != nil {
				logging.Log.Fatalf("Failed to generate new JWT secret: %v", err)
			}
			newSecretString := hex.EncodeToString(newSecretBytes)

			cfg.JWT.Secret = newSecretString // This will be saved to the file
			cfg.JWTSecret = newSecretString  // This is used at runtime

			// Save back to file
			if err := config.SaveConfig(cliCfg.ConfigPath, cfg); err != nil {
				logging.Log.Warnf("Failed to save new JWT secret to %s: %v", cliCfg.ConfigPath, err)
			} else {
				logging.Log.Infof("New JWT secret has been generated and saved to %s.", cliCfg.ConfigPath)
			}
		}
	}

	logging.Init(cfg.Logging.Level)

	media.Initialize(cfg.Media.FFmpegPath, cfg.Media.FFprobePath)

	repo, err := repository.NewRepository(cfg)
	if err != nil {
		logging.Log.Fatalf("Failed to initialize repository: %v", err)
	}
	defer repo.Close()

	// Service Initialization
	storageService := services.NewStorageService(cfg)
	infoService := services.NewInfoService(version, startTime, media.IsFFmpegAvailable(), media.IsFFprobeAvailable())
	userService := services.NewUserService(repo)

	// Initialize Token Service
	tokenService := auth.NewTokenService(cfg, userService, repo) // <-- NEW

	databaseService := services.NewDatabaseService(repo, storageService)
	entryService := services.NewEntryService(repo, storageService, cfg)
	housekeepingService := services.NewHousekeepingService(repo, storageService)

	// Initialize Auth Middleware (Hybrid)
	authMiddleware := auth.NewMiddleware(userService, tokenService) // <-- UPDATED

	// Call Admin User Setup from UserService ---
	if err := userService.InitializeAdminUser(cfg); err != nil {
		logging.Log.Fatalf("Failed to handle admin user: %v", err)
	}

	// Run init config (depends on Repository) ---
	if initConfigPath != "" {
		logging.Log.Infof("Found init_config, running initialization from: %s", initConfigPath)
		initconfig.Run(userService, databaseService, initConfigPath)
	}

	// Start Housekeeping Service ---
	housekeepingService.Start()
	defer housekeepingService.Stop()

	// Create Handlers struct with all services ---
	h := handlers.NewHandlers(
		infoService,
		userService,
		tokenService,
		databaseService,
		entryService,
		housekeepingService,
		cfg,
	)

	// Setup Router with Handlers and Auth Middleware ---
	r := api.SetupRouter(h, authMiddleware, cfg, frontendFS)

	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logging.Log.Infof("Server starting on %s", serverAddr)
	if err := http.ListenAndServe(serverAddr, r); err != nil {
		logging.Log.Fatalf("Failed to start server: %v", err)
	}
}

// overrideConfigFromEnvAndCLI applies overrides from environment variables and CLI flags.
// CLI flags take precedence over environment variables.
func overrideConfigFromEnvAndCLI(cliCfg *CLIConfig, cfg *config.Config) {
	// --- Load from Environment Variables FIRST ---
	if envPassword := os.Getenv("FDB_PASSWORD"); envPassword != "" {
		cfg.AdminPassword = envPassword
	}
	if envPort := os.Getenv("FDB_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			cfg.Server.Port = p
		}
	}
	if envLogLevel := os.Getenv("FDB_LOG_LEVEL"); envLogLevel != "" {
		cfg.Logging.Level = envLogLevel
	}
	if envResetPW := os.Getenv("FDB_RESET_PW"); envResetPW == "true" {
		cfg.ResetAdminPassword = true
	}
	if envDBPath := os.Getenv("FDB_DATABASE_PATH"); envDBPath != "" {
		cfg.Database.Path = envDBPath
	}
	if envStorageRoot := os.Getenv("FDB_STORAGE_ROOT"); envStorageRoot != "" {
		cfg.Database.StorageRoot = envStorageRoot
	}
	if envFFmpegPath := os.Getenv("FDB_FFMPEG_PATH"); envFFmpegPath != "" {
		cfg.Media.FFmpegPath = envFFmpegPath
	}
	if envFFprobePath := os.Getenv("FDB_FFPROBE_PATH"); envFFprobePath != "" {
		cfg.Media.FFprobePath = envFFprobePath
	}
	if envJWTSecret := os.Getenv("FDB_JWT_SECRET"); envJWTSecret != "" {
		cfg.JWTSecret = envJWTSecret
	}

	// --- Load from CLI Flags SECOND (to override env) ---
	if cliCfg.Password != "" {
		cfg.AdminPassword = cliCfg.Password
	}
	if cliCfg.Port != 0 {
		cfg.Server.Port = cliCfg.Port
	}
	if cliCfg.LogLevel != "" {
		cfg.Logging.Level = cliCfg.LogLevel
	}
	if cliCfg.ResetPassword {
		cfg.ResetAdminPassword = true
	}
	if cliCfg.FFmpegPath != "" {
		cfg.Media.FFmpegPath = cliCfg.FFmpegPath
	}
	if cliCfg.FFprobePath != "" {
		cfg.Media.FFprobePath = cliCfg.FFprobePath
	}
	if cliCfg.JWTSecret != "" {
		cfg.JWTSecret = cliCfg.JWTSecret
	}

	// --- Set Defaults ---
	if cfg.Server.Host == "" {
		cfg.Server.Host = "localhost"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = "mediahub.db"
	}
	if cfg.Database.StorageRoot == "" {
		cfg.Database.StorageRoot = "storage_root"
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.JWT.AccessDurationMin == 0 {
		cfg.JWT.AccessDurationMin = 5
	}
	if cfg.JWT.RefreshDurationHours == 0 {
		cfg.JWT.RefreshDurationHours = 24
	}
}
