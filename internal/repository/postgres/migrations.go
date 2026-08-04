package postgres

import (
	"context"
	"fmt"

	"mediahub_oss/internal/repository/migrations"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.SetBaseFS(migrations.EmbedFS)
}

// GetMigrationVersion retrieves the current schema version (e.g. 3003).
func (r *PostgresRepository) GetMigrationVersion(ctx context.Context) (int, error) {
	if err := goose.SetDialect("postgres"); err != nil {
		return 0, fmt.Errorf("failed to set goose dialect: %w", err)
	}

	version, err := goose.GetDBVersion(r.DB)
	if err != nil {
		return 0, fmt.Errorf("failed to get database version: %w", err)
	}

	return int(version), nil
}

// MigrateUp applies all pending migrations.
func (r *PostgresRepository) MigrateUp(ctx context.Context) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.UpContext(ctx, r.DB, "postgres"); err != nil {
		return fmt.Errorf("failed to migrate database up: %w", err)
	}
	return nil
}

// MigrateDown rolls back the database by one version.
func (r *PostgresRepository) MigrateDown(ctx context.Context) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.DownContext(ctx, r.DB, "postgres"); err != nil {
		return fmt.Errorf("failed to migrate database down: %w", err)
	}
	return nil
}
