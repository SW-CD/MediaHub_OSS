package cli

import "github.com/spf13/cobra"

type RecoveryOptions struct {
	DryRun bool // If true, report only without editing
}

func NewRecoveryCommand(globalOptions *GlobalOptions) *cobra.Command {

	recoveryOptions := &RecoveryOptions{DryRun: false}

	recoveryCommand := &cobra.Command{
		Use:   "recovery",
		Short: "Run maintenance tasks to fix database inconsistencies",
		Long: `Scans the database for entries stuck in 'processing' state (e.g., due to a server crash during upload)
and marks them as 'error'. This does not start the HTTP server.`,
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
	// logger := globalOptions.Logger
	// TODO
	return nil
}
