package cli

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewMigrateCommand(globalOptions *GlobalOptions) *cobra.Command {

	var migrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Database migration tools",
		Long:  `Manage database schema versions. Use subcommands 'up', 'down', or 'status'.`,
	}

	var upCmd = &cobra.Command{
		Use:   "up",
		Short: "Migrate the database to the most recent version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigration("up", globalOptions.Logger)
		},
	}

	var downCmd = &cobra.Command{
		Use:   "down",
		Short: "Roll back the database by one version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigration("down", globalOptions.Logger)
		},
	}

	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Dump the migration status for the current DB",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigration("status", globalOptions.Logger)
		},
	}

	// Add subcommands
	migrateCmd.AddCommand(upCmd)
	migrateCmd.AddCommand(downCmd)
	migrateCmd.AddCommand(statusCmd)

	return migrateCmd
}

func runMigration(command string, logger *logrus.Logger) error {

	// TODO
	// ...
	return nil
}
