// filepath: internal/cli/recovery.go
package cli

import (
	"fmt"
	"mediahub/internal/logging"
	"mediahub/internal/repository"

	"github.com/spf13/cobra"
)

var recoveryCmd = &cobra.Command{
	Use:   "recovery",
	Short: "Run maintenance tasks to fix database inconsistencies",
	Long: `Scans the database for entries stuck in 'processing' state (e.g., due to a server crash during upload)
and marks them as 'error'. This does not start the HTTP server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRecovery()
	},
}

func init() {
	RootCmd.AddCommand(recoveryCmd)
}

func runRecovery() error {
	// Initialize repository (using cfg loaded by RootCmd)
	repo, err := repository.NewRepository(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer repo.Close()

	if err := repo.ValidateSchema(); err != nil {
		return fmt.Errorf("cannot run recovery on outdated database: %w", err)
	}

	logging.Log.Info("Starting recovery process...")

	// Use the new repository method
	totalFixed, err := repo.FixZombieEntries()
	if err != nil {
		return err
	}

	logging.Log.Infof("Recovery complete. Total entries fixed: %d", totalFixed)
	return nil
}
