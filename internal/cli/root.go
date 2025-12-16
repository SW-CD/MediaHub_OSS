// filepath: internal/cli/root.go
package cli

import (
	"embed"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	// Version info
	Version   = "1.2.0"
	StartTime time.Time

	// frontendFS holds the embedded frontend assets.
	frontendFS embed.FS
)

// RootCmd represents the base command when called without any subcommands.
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

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute(fs embed.FS) {
	frontendFS = fs // Store for use in runServer
	StartTime = time.Now()

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Register flags (implementation in config_loader.go)
	registerFlags(RootCmd)
}
