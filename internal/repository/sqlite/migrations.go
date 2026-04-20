package sqlite

import (
	"context"
	"fmt"

	// Adjust this import path to point to where your EmbedFS is located
	"mediahub_oss/internal/repository/migrations"

	"github.com/pressly/goose/v3"
)

// Ensure goose uses the embedded filesystem and the correct dialect
func init() {
	goose.SetBaseFS(migrations.EmbedFS)
}

// GetMigrationVersion retrieves the current schema version (e.g., 2000).
func (r *SQLiteRepository) GetMigrationVersion(ctx context.Context) (int, error) {
	goose.SetDialect("sqlite3")

	// GetDBVersion returns an int64. We cast it to int to match our interface.
	version, err := goose.GetDBVersion(r.DB)
	if err != nil {
		return 0, fmt.Errorf("failed to get database version: %w", err)
	}

	return int(version), nil
}

// MigrateUp applies all pending migrations.
func (r *SQLiteRepository) MigrateUp(ctx context.Context) error {
	goose.SetDialect("sqlite3")

	// The second argument "sqlite" refers to the subfolder inside your embed.FS
	if err := goose.UpContext(ctx, r.DB, "sqlite"); err != nil {
		return fmt.Errorf("failed to migrate database up: %w", err)
	}
	return nil
}

// MigrateDown rolls back the database by one version.
func (r *SQLiteRepository) MigrateDown(ctx context.Context) error {
	goose.SetDialect("sqlite3")

	if err := goose.DownContext(ctx, r.DB, "sqlite"); err != nil {
		return fmt.Errorf("failed to migrate database down: %w", err)
	}
	return nil
}
