package sqlitemigrations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"mediahub_oss/internal/media"
	"mediahub_oss/internal/repository"
	sqlite "mediahub_oss/internal/repository/sqlite"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(up02001, down02001)
}

func up02001(ctx context.Context, tx *sql.Tx) error {
	// 1. Add n_max_queued column to databases table
	_, err := tx.ExecContext(ctx, "ALTER TABLE databases ADD COLUMN n_max_queued INTEGER NOT NULL DEFAULT 0;")
	if err != nil {
		return fmt.Errorf("failed to add n_max_queued column to databases: %w", err)
	}

	// 2. Migrate dynamic entries tables check constraints to include status 4 (queued)
	return migrateCheckConstraints(ctx, tx, []uint8{0, 1, 2, 3, 4}, false)
}

func down02001(ctx context.Context, tx *sql.Tx) error {
	// 1. Revert dynamic entries tables check constraints back to status 0, 1, 2, 3
	if err := migrateCheckConstraints(ctx, tx, []uint8{0, 1, 2, 3}, true); err != nil {
		return err
	}

	// 2. Drop n_max_queued column from databases table
	_, err := tx.ExecContext(ctx, "ALTER TABLE databases DROP COLUMN n_max_queued;")
	if err != nil {
		return fmt.Errorf("failed to drop n_max_queued column from databases: %w", err)
	}

	return nil
}

func migrateCheckConstraints(ctx context.Context, tx *sql.Tx, allowedStatuses []uint8, isDowngrade bool) error {
	// 1. Query databases
	rows, err := tx.QueryContext(ctx, "SELECT id, content_type, custom_fields FROM databases")
	if err != nil {
		// If databases table doesn't exist yet, skip
		return nil
	}
	defer rows.Close()

	type dbInfo struct {
		ID           string
		ContentType  string
		CustomFields []repository.CustomField
	}
	var databases []dbInfo
	for rows.Next() {
		var id, contentType, customFieldsJSON string
		if err := rows.Scan(&id, &contentType, &customFieldsJSON); err != nil {
			return err
		}
		var customFields []repository.CustomField
		if err := json.Unmarshal([]byte(customFieldsJSON), &customFields); err != nil {
			return err
		}
		databases = append(databases, dbInfo{id, contentType, customFields})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// 2. Build MediaFields map for SQLiteRepository helper
	mediaFields := make(map[string][]sqlite.MediaField)
	for _, contentType := range media.GetContentTypes() {
		fieldDefs, err := media.GetMetadataFields(contentType)
		if err != nil {
			return err
		}
		mediaFieldsOfContent := make([]sqlite.MediaField, len(fieldDefs))
		for i, v := range fieldDefs {
			mediaFieldsOfContent[i] = sqlite.MediaField{Name: v.Name, SQLiteType: v.Type}
		}
		mediaFields[contentType] = mediaFieldsOfContent
	}

	// Create a temporary SQLiteRepository for schema generation
	r := &sqlite.SQLiteRepository{
		AllowedStatuses: allowedStatuses,
		MediaFields:     mediaFields,
	}

	// 3. Migrate each entries table
	for _, db := range databases {
		tableName := fmt.Sprintf(`"entries_%s"`, db.ID)
		oldTableName := fmt.Sprintf(`"entries_%s_old"`, db.ID)

		// Check if table exists
		var sqlSchema string
		err = tx.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type='table' AND name=?`, fmt.Sprintf("entries_%s", db.ID)).Scan(&sqlSchema)
		if err != nil {
			continue // Skip if table doesn't exist
		}

		// Check if we need to migrate
		statusListStr := strings.Join(func() []string {
			var strs []string
			for _, s := range allowedStatuses {
				strs = append(strs, fmt.Sprintf("%d", s))
			}
			return strs
		}(), ", ")
		expectedPattern := fmt.Sprintf("status IN (%s)", statusListStr)

		if strings.Contains(sqlSchema, expectedPattern) {
			continue // Already has the expected constraint, skip
		}

		// If downgrading, update any status = 4 (queued) to status = 2 (error) first
		if isDowngrade {
			updateSQL := fmt.Sprintf(`UPDATE %s SET status = 2 WHERE status = 4`, tableName)
			_, _ = tx.ExecContext(ctx, updateSQL)
		}

		newSchemaSQL, err := r.BuildDynamicTableSchema(db.ID, db.ContentType, db.CustomFields)
		if err != nil {
			return fmt.Errorf("failed to build dynamic schema SQL for db %s: %w", db.ID, err)
		}

		// Rename old table
		renameSQL := fmt.Sprintf(`ALTER TABLE %s RENAME TO %s`, tableName, oldTableName)
		if _, err := tx.ExecContext(ctx, renameSQL); err != nil {
			return fmt.Errorf("failed to rename old table: %w", err)
		}

		// Create new table with updated constraints
		if _, err := tx.ExecContext(ctx, newSchemaSQL); err != nil {
			return fmt.Errorf("failed to create new table: %w", err)
		}

		// Copy data
		copySQL := fmt.Sprintf(`INSERT INTO %s SELECT * FROM %s`, tableName, oldTableName)
		if _, err := tx.ExecContext(ctx, copySQL); err != nil {
			return fmt.Errorf("failed to copy data: %w", err)
		}

		// Drop old table
		dropSQL := fmt.Sprintf(`DROP TABLE %s`, oldTableName)
		if _, err := tx.ExecContext(ctx, dropSQL); err != nil {
			return fmt.Errorf("failed to drop old table: %w", err)
		}

		// Recreate indexes
		indexesSQLs := sqlite.BuildIndexesSQL(db.ID, db.CustomFields)
		for _, indexSQL := range indexesSQLs {
			if _, err := tx.ExecContext(ctx, indexSQL); err != nil {
				return fmt.Errorf("failed to recreate index: %w", err)
			}
		}
	}

	return nil
}
