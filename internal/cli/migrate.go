// filepath: internal/cli/migrate.go
package cli

import (
	"context"
	"fmt"
	"mediahub/internal/db/migrations"
	"mediahub/internal/logging"
	"mediahub/internal/repository"

	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration tools",
	Long:  `Manage database schema versions. Use subcommands 'up', 'down', or 'status'.`,
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Migrate the database to the most recent version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMigration("up")
	},
}

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Roll back the database by one version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMigration("down")
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Dump the migration status for the current DB",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMigration("status")
	},
}

func init() {
	RootCmd.AddCommand(migrateCmd)
	migrateCmd.AddCommand(upCmd)
	migrateCmd.AddCommand(downCmd)
	migrateCmd.AddCommand(statusCmd)
}

func runMigration(command string) error {
	// We need to connect to the database. The root command's PersistentPreRunE
	// has already loaded the 'cfg' global variable.

	repo, err := repository.NewRepository(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer repo.Close()

	// Configure Goose
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	// The 'internal/db/migrations' directory is embedded, so we pass "."
	// to tell goose to look at the root of the embedded FS.
	dir := "."

	logging.Log.Infof("Running migration command: %s", command)

	var gooseErr error
	switch command {
	case "up":
		gooseErr = goose.Up(repo.DB, dir)
	case "down":
		gooseErr = goose.Down(repo.DB, dir)
	case "status":
		gooseErr = goose.Status(repo.DB, dir)
	default:
		return fmt.Errorf("unknown migration command: %s", command)
	}

	if gooseErr != nil {
		return fmt.Errorf("migration failed: %w", gooseErr)
	}

	logging.Log.Info("Migration operation completed successfully.")
	return nil
}

// Explicit stub to fix 'context' usage if necessary, though not strictly used in simple RunE.
var _ = context.Background
