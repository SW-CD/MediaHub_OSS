package cli

import (
	"fmt"
	"mediahub/internal/repository"
	"mediahub/internal/repository/sqlite"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewMigrateCommand(globalOptions *GlobalOptions) *cobra.Command {

	repository, err := sqlite.NewRepository(globalOptions.Conf.Database.Path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var migrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Database migration tools",
		Long:  `Manage database schema versions. Use subcommands 'up', 'down', or 'status'.`,
	}

	var upCmd = &cobra.Command{
		Use:   "up",
		Short: "Migrate the database to the most recent version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigration("up", repository, globalOptions.Logger)
		},
	}

	var downCmd = &cobra.Command{
		Use:   "down",
		Short: "Roll back the database by one version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigration("down", repository, globalOptions.Logger)
		},
	}

	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Dump the migration status for the current DB",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigration("status", repository, globalOptions.Logger)
		},
	}

	// Add subcommands
	migrateCmd.AddCommand(upCmd)
	migrateCmd.AddCommand(downCmd)
	migrateCmd.AddCommand(statusCmd)

	return migrateCmd
}

func runMigration(command string, repository repository.Repository, logger *logrus.Logger) error {

	// TODO
	// ...
	return nil
}
