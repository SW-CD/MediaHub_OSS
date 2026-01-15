package cli

import (
	"embed"
	"fmt"
	"os"

	conf "mediahub/internal/cli/config"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type GlobalOptions struct {
	CfgFilePath string
	LogLevel    string

	Logger *logrus.Logger
	Conf   *conf.Config
}

func NewRootCMD() *cobra.Command {

	globalOptions := &GlobalOptions{}

	rootCMD := &cobra.Command{
		Use:   "mediahub",
		Short: "SWCD MediaHub",
		Long:  "A server for a image, audio and file storage with integrated web-ui.",
		// TODO: run help if no other command provided
	}

	// register global flags
	globalOptions.registerFlags(rootCMD)

	// add subcommands
	rootCMD.AddCommand(NewServeCommand(globalOptions))
	rootCMD.AddCommand(NewMigrateCommand(globalOptions))
	rootCMD.AddCommand(NewRecoveryCommand(globalOptions))

	return rootCMD
}

func (options *GlobalOptions) registerFlags(cmd *cobra.Command) {
	// flags that can be used for each command
	cmd.PersistentFlags().StringVar(&options.CfgFilePath, "config_path", "config.toml", "Path to the base configuration file. (Env: FDB_CONFIG_PATH)")
	cmd.PersistentFlags().StringVar(&options.LogLevel, "log-level", "", "Logging level (debug, info, warn, error). (Env: FDB_LOG_LEVEL)")
}

func Execute(frontendFS embed.FS) {

	rootCmd := NewRootCMD()

	// Run the command based on os.Args
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
