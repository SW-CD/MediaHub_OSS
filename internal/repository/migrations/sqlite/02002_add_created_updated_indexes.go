// Migration: Add CreatedAt & UpdatedAt Indexes to dynamic entry tables
// Description: Upgrades the database schema from v2.1 to v2.2 by adding indexes for created_at and updated_at.
// Work In Progress file.
package sqlitemigrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(up02002, down02002)
}

func up02002(ctx context.Context, tx *sql.Tx) error {
	// Query all existing database IDs
	rows, err := tx.QueryContext(ctx, "SELECT id FROM databases")
	if err != nil {
		// If databases table does not exist, there are no databases, skip
		return nil
	}
	defer rows.Close()

	var dbIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("failed to scan database ID: %w", err)
		}
		dbIDs = append(dbIDs, id)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating database rows: %w", err)
	}
	rows.Close()

	// Create indexes for each database
	for _, dbID := range dbIDs {
		idxCreated := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS "idx_entries_%s_created" ON "entries_%s"(created_at);`, dbID, dbID)
		idxUpdated := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS "idx_entries_%s_updated" ON "entries_%s"(updated_at);`, dbID, dbID)

		if _, err := tx.ExecContext(ctx, idxCreated); err != nil {
			return fmt.Errorf("failed to create created_at index for db %s: %w", dbID, err)
		}
		if _, err := tx.ExecContext(ctx, idxUpdated); err != nil {
			return fmt.Errorf("failed to create updated_at index for db %s: %w", dbID, err)
		}
	}

	return nil
}

func down02002(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, "SELECT id FROM databases")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var dbIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("failed to scan database ID: %w", err)
		}
		dbIDs = append(dbIDs, id)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating database rows: %w", err)
	}
	rows.Close()

	// Drop indexes for each database
	for _, dbID := range dbIDs {
		idxCreated := fmt.Sprintf(`DROP INDEX IF EXISTS "idx_entries_%s_created";`, dbID)
		idxUpdated := fmt.Sprintf(`DROP INDEX IF EXISTS "idx_entries_%s_updated";`, dbID)

		if _, err := tx.ExecContext(ctx, idxCreated); err != nil {
			return fmt.Errorf("failed to drop created_at index for db %s: %w", dbID, err)
		}
		if _, err := tx.ExecContext(ctx, idxUpdated); err != nil {
			return fmt.Errorf("failed to drop updated_at index for db %s: %w", dbID, err)
		}
	}

	return nil
}
