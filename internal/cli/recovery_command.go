package cli

import (
	"context"
	"fmt"

	// Import your new package
	"mediahub_oss/internal/cli/recovery"

	"github.com/spf13/cobra"
)

type RecoveryOptions struct {
	DryRun bool // If true, report only without editing
}

func NewRecoveryCommand(globalOptions *GlobalOptions) *cobra.Command {

	recoveryOptions := &RecoveryOptions{DryRun: false}

	recoveryCommand := &cobra.Command{
		Use:   "recovery",
		Short: "Run maintenance tasks to fix database inconsistencies",
		Long: `Scans the database for entries stuck in 'processing' or 'deleting' state. 
		Also checks for orphan files or database entries and cleans them up.
		This does not start the HTTP server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRecovery(globalOptions, recoveryOptions)
		},
	}

	recoveryOptions.registerFlags(recoveryCommand, globalOptions)

	return recoveryCommand
}

func (opt *RecoveryOptions) registerFlags(cmd *cobra.Command, globalOptions *GlobalOptions) {
	cmd.Flags().BoolVar(&opt.DryRun, "dryrun", false, "If true, report only without editing.")
}

func runRecovery(globalOptions *GlobalOptions, recoveryOptions *RecoveryOptions) error {
	logger := globalOptions.Logger
	ctx := context.Background()

	logger.Info("Starting recovery maintenance...", "dryRun", recoveryOptions.DryRun)

	// 1. Initialize the new recovery service
	// We pass the configuration so the service can connect to the DB and Storage,
	// along with the logger and the DryRun flag.
	recoverySvc, err := recovery.NewRecoveryService(globalOptions.Conf, logger, recoveryOptions.DryRun)
	if err != nil {
		return fmt.Errorf("failed to initialize recovery service: %w", err)
	}
	defer recoverySvc.Close()

	// 2. Execute Phase 1: Zombie Fix
	// Scans for entries stuck in "processing" or "deleting"
	logger.Info("Phase 1: Running Zombie Fix...")
	if err := recoverySvc.EntryStatusCorrection(ctx); err != nil {
		return fmt.Errorf("Entry status correction failed: %w", err)
	}

	// 3. Execute Phase 2: Integrity Check
	// Verifies file/DB parity
	logger.Info("Phase 2: Running Integrity Check...")
	if err := recoverySvc.IntegrityCheck(ctx); err != nil {
		return fmt.Errorf("integrity check failed: %w", err)
	}

	if err := recoverySvc.Close(); err != nil {
		return fmt.Errorf("failed to close recovery service: %w", err)
	}

	logger.Info("Recovery process completed successfully.")
	return nil
}
