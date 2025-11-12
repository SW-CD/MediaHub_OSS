// filepath: cmd/mediahub/main.go
package main

import (
	"embed" // Import the embed package
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

	var initConfigPath string // This variable will be filled by the flag
	flag.StringVar(&initConfigPath, "init_config", initConfigPathDefault, "Path to a TOML config file for one-time initialization of users/databases. (Env: FDB_INIT_CONFIG)")

	flag.Usage = customUsage

	flag.Parse()

	// 2. Load configuration from file
	// cliCfg.ConfigPath now holds the correct path (CLI > ENV > Default)
	cfg, err := config.LoadConfig(cliCfg.ConfigPath)
	if err != nil {
		isHelp := false
		for _, arg := range os.Args {
			if arg == "-h" || arg == "--help" || arg == "help" {
				isHelp = true
				break
			}
		}
		if !isHelp {
			// Only error if --help was not requested
			fmt.Printf("Failed to load configuration from %s: %v\n", cliCfg.ConfigPath, err)
			os.Exit(1)
		} else {
			// If help was requested, `flag.Parse()` already printed it and exited.
			return
		}
	}

	// 3. Override with CLI flags and environment variables
	overrideConfigFromEnvAndCLI(&cliCfg, cfg)

	logging.Init(cfg.Logging.Level)

	media.Initialize(cfg.Media.FFmpegPath, cfg.Media.FFprobePath)

	repo, err := repository.NewRepository(cfg)
	if err != nil {
		logging.Log.Fatalf("Failed to initialize repository: %v", err)
	}
	defer repo.Close()

	storageService := services.NewStorageService(cfg)
	infoService := services.NewInfoService(version, startTime, media.IsFFmpegAvailable(), media.IsFFprobeAvailable())
	userService := services.NewUserService(repo)
	databaseService := services.NewDatabaseService(repo, storageService)
	entryService := services.NewEntryService(repo, storageService, cfg)
	housekeepingService := services.NewHousekeepingService(repo, storageService)

	// Initialize Auth Middleware (depends on UserService) ---
	authMiddleware := auth.NewMiddleware(userService)

	// Call Admin User Setup from UserService ---
	if err := userService.InitializeAdminUser(cfg); err != nil {
		logging.Log.Fatalf("Failed to handle admin user: %v", err)
	}

	// Run init config (depends on Repository) ---
	// initConfigPath now holds the correct path (CLI > ENV > Default)
	if initConfigPath != "" {
		logging.Log.Infof("Found init_config, running initialization from: %s", initConfigPath)
		initconfig.Run(userService, databaseService, initConfigPath)
	}

	// Start Housekeeping Service ---
	housekeepingService.Start()
	defer housekeepingService.Stop()

	// Create Handlers struct with all services ---
	// This now passes the concrete service structs, which satisfy the
	// interfaces expected by NewHandlers.
	h := handlers.NewHandlers(
		infoService,
		userService,
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
}
