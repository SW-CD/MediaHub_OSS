// Currently the code uses simple if then statements. If more options are added,
// swapping to github.com/spf13/viper could be helpful. For now, I like simplicity.
package cli

import (
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

type ServeOptions struct {
	StartTime     time.Time
	Password      string
	Port          int
	ResetPW       bool
	FFMPEGPath    string
	FFPROBEPath   string
	JWTSecret     string
	MaxSyncUpload string
	InitConfig    string
	AuditEnabled  bool
}

func NewServeCommand(globalOptions *GlobalOptions) *cobra.Command {
	serveOptions := &ServeOptions{}

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the server",
		Run: func(cmd *cobra.Command, args []string) {
			serve(globalOptions, serveOptions)
		},
	}

	serveOptions.registerFlags(serveCmd)
	serveOptions.registerEnvVars(serveCmd)

	return serveCmd
}

func (options *ServeOptions) registerFlags(cmd *cobra.Command) {
	// flags for the serve command only
	cmd.Flags().StringVar(&options.Password, "password", "", "Password for the 'admin' user. (Env: MEDIAHUB_PASSWORD)")
	cmd.Flags().IntVar(&options.Port, "port", 0, "Port for the HTTP server. (Env: MEDIAHUB_PORT)")
	cmd.Flags().BoolVar(&options.ResetPW, "reset_pw", false, "If true, reset admin password on startup. (Env: MEDIAHUB_RESET_PW=true)")
	cmd.Flags().StringVar(&options.FFMPEGPath, "ffmpeg-path", "", "Path to ffmpeg executable. (Env: MEDIAHUB_FFMPEG_PATH)")
	cmd.Flags().StringVar(&options.FFPROBEPath, "ffprobe-path", "", "Path to ffprobe executable. (Env: MEDIAHUB_FFPROBE_PATH)")
	cmd.Flags().StringVar(&options.JWTSecret, "jwt-secret", "", "Secret key for signing JWTs. (Env: MEDIAHUB_JWT_SECRET)")
	cmd.Flags().StringVar(&options.MaxSyncUpload, "max-sync-upload", "", "Max size for synchronous/in-memory uploads (e.g. '8MB'). (Env: MEDIAHUB_MAX_SYNC_UPLOAD)")
	cmd.Flags().StringVar(&options.InitConfig, "init_config", "", "Path to a TOML config file for one-time initialization of users/databases. (Env: MEDIAHUB_INIT_CONFIG)")
	cmd.Flags().BoolVar(&options.AuditEnabled, "audit-enabled", false, "Enable detailed audit logging. (Env: MEDIAHUB_AUDIT_ENABLED=true)")
}

// In case a variable was not defined in the cli arguments, we check for env variables
func (options *ServeOptions) registerEnvVars(cmd *cobra.Command) {
	getEnv := func(key string) string { return os.Getenv(key) }

	if options.Password == "" {
		options.Password = getEnv("MEDIAHUB_PASSWORD")
	}
	if options.Port == 0 {
		portstr := getEnv("MEDIAHUB_PORT")
		if p, err := strconv.Atoi(portstr); err == nil {
			options.Port = p
		}
	}
	if !options.ResetPW {
		options.ResetPW = getEnv("MEDIAHUB_RESET_PW") == "true"
	}
	if options.FFMPEGPath == "" {
		options.FFMPEGPath = getEnv("MEDIAHUB_FFMPEG_PATH")
	}
	if options.FFPROBEPath == "" {
		options.FFPROBEPath = getEnv("MEDIAHUB_FFPROBE_PATH")
	}
	if options.JWTSecret == "" {
		options.JWTSecret = getEnv("MEDIAHUB_JWT_SECRET")
	}
	if options.MaxSyncUpload == "" {
		options.MaxSyncUpload = getEnv("MEDIAHUB_MAX_SYNC_UPLOAD")
	}
	if options.InitConfig == "" {
		options.InitConfig = getEnv("MEDIAHUB_INIT_CONFIG")
	}
	if options.AuditEnabled == false {
		options.AuditEnabled = getEnv("MEDIAHUB_AUDIT_ENABLED") == "true"
	}

}

func serve(globalOptions *GlobalOptions, serveOptions *ServeOptions) {
	//TODO
}
