package sqlite

import (
	"context"
	"fmt"
	"strings"
	"time"

	repo "mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared/customerrors"

	"github.com/Masterminds/squirrel"
)

// GetCustomFields retrieves all custom fields for a specific database.
func (r *SQLiteRepository) GetCustomFields(ctx context.Context, dbID string) ([]repo.CustomFieldDef, error) {
	// First check if database exists to return 404 if not found
	var exists bool
	err := r.DB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM databases WHERE id = ?)", dbID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check database existence: %w", err)
	}
	if !exists {
		return nil, customerrors.ErrNotFound
	}
	return r.getCustomFields(ctx, r.DB, dbID)
}

// getCustomFields retrieves all custom fields for a specific database with cache backing.
func (r *SQLiteRepository) getCustomFields(ctx context.Context, q Queryer, dbID string) ([]repo.CustomFieldDef, error) {
	cacheKey := "cf:" + dbID
	if val, found := r.Cache.Get(cacheKey); found {
		return val.([]repo.CustomFieldDef), nil
	}

	query, args, err := r.Builder.Select("field_id", "name", "type", "is_indexed").
		From("database_custom_fields").
		Where(squirrel.Eq{"database_id": dbID}).
		OrderBy("field_id").
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fields []repo.CustomFieldDef
	for rows.Next() {
		var cf repo.CustomFieldDef
		if err := rows.Scan(&cf.ID, &cf.Name, &cf.Type, &cf.IsIndexed); err != nil {
			return nil, err
		}
		fields = append(fields, cf)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// If empty list, initialize it to avoid returning nil
	if fields == nil {
		fields = []repo.CustomFieldDef{}
	}

	r.Cache.Set(cacheKey, fields, 5*time.Minute)
	return fields, nil
}

// AddCustomField adds a new custom field to an existing database.
func (r *SQLiteRepository) AddCustomField(ctx context.Context, dbID string, field repo.CustomFieldDef) (repo.CustomFieldDef, error) {
	// Check if database exists
	var exists bool
	err := r.DB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM databases WHERE id = ?)", dbID).Scan(&exists)
	if err != nil {
		return repo.CustomFieldDef{}, fmt.Errorf("failed to check database existence: %w", err)
	}
	if !exists {
		return repo.CustomFieldDef{}, customerrors.ErrNotFound
	}

	// Validate name
	if field.Name == "" {
		return repo.CustomFieldDef{}, fmt.Errorf("%w: field name cannot be empty", customerrors.ErrValidation)
	}

	// Validate type
	datatype := strings.ToUpper(field.Type)
	if datatype != "TEXT" && datatype != "INTEGER" && datatype != "REAL" && datatype != "BOOLEAN" {
		return repo.CustomFieldDef{}, fmt.Errorf("%w: unsupported custom field type '%s'", customerrors.ErrValidation, field.Type)
	}

	// Load existing fields
	existingFields, err := r.getCustomFields(ctx, r.DB, dbID)
	if err != nil {
		return repo.CustomFieldDef{}, err
	}

	// Check name uniqueness
	for _, f := range existingFields {
		if strings.EqualFold(f.Name, field.Name) {
			return repo.CustomFieldDef{}, customerrors.ErrConflict
		}
	}

	// Find the next available ID between 0 and 254
	usedIDs := make(map[int]bool)
	for _, f := range existingFields {
		usedIDs[f.ID] = true
	}
	nextID := -1
	for i := 0; i <= 254; i++ {
		if !usedIDs[i] {
			nextID = i
			break
		}
	}
	if nextID == -1 {
		return repo.CustomFieldDef{}, fmt.Errorf("Cannot add field: The maximum limit of 255 custom fields has been reached.")
	}
	field.ID = nextID

	// Begin transaction
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return repo.CustomFieldDef{}, err
	}
	defer tx.Rollback()

	// 1. Insert into database_custom_fields
	query, args, err := r.Builder.Insert("database_custom_fields").
		Columns("database_id", "field_id", "name", "type", "is_indexed").
		Values(dbID, field.ID, field.Name, datatype, field.IsIndexed).
		ToSql()
	if err != nil {
		return repo.CustomFieldDef{}, err
	}

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return repo.CustomFieldDef{}, customerrors.ErrConflict
		}
		return repo.CustomFieldDef{}, fmt.Errorf("failed to insert custom field: %w", err)
	}

	// 2. ALTER TABLE entries_ID ADD COLUMN cf_nextID Type
	tableName := fmt.Sprintf(`"entries_%s"`, dbID)
	alterSQL := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN "%s%d" %s`, tableName, customFieldsPrefix, field.ID, datatype)
	if _, err := tx.ExecContext(ctx, alterSQL); err != nil {
		return repo.CustomFieldDef{}, fmt.Errorf("failed to add column to entries table: %w", err)
	}

	// 3. Create index if is_indexed is true
	if field.IsIndexed {
		indexSQL := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS "idx_entries_%s_%s%d" ON %s("%s%d")`, dbID, customFieldsPrefix, field.ID, tableName, customFieldsPrefix, field.ID)
		if _, err := tx.ExecContext(ctx, indexSQL); err != nil {
			return repo.CustomFieldDef{}, fmt.Errorf("failed to create index on custom field: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return repo.CustomFieldDef{}, err
	}

	// Invalidate cache
	r.Cache.Delete("cf:" + dbID)

	return field, nil
}

// UpdateCustomField updates an existing custom field.
func (r *SQLiteRepository) UpdateCustomField(ctx context.Context, dbID string, fieldID int, name *string, isIndexed *bool) (repo.CustomFieldDef, error) {
	// Check if database exists
	var exists bool
	err := r.DB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM databases WHERE id = ?)", dbID).Scan(&exists)
	if err != nil {
		return repo.CustomFieldDef{}, fmt.Errorf("failed to check database existence: %w", err)
	}
	if !exists {
		return repo.CustomFieldDef{}, customerrors.ErrNotFound
	}

	// Load existing fields
	existingFields, err := r.getCustomFields(ctx, r.DB, dbID)
	if err != nil {
		return repo.CustomFieldDef{}, err
	}

	// Find the field
	var targetField *repo.CustomFieldDef
	for i := range existingFields {
		if existingFields[i].ID == fieldID {
			targetField = &existingFields[i]
			break
		}
	}
	if targetField == nil {
		return repo.CustomFieldDef{}, customerrors.ErrNotFound
	}

	// Validate name update
	newName := targetField.Name
	if name != nil {
		newName = *name
		if newName == "" {
			return repo.CustomFieldDef{}, fmt.Errorf("%w: name cannot be empty", customerrors.ErrValidation)
		}
		// Check name uniqueness if changed
		if !strings.EqualFold(newName, targetField.Name) {
			for _, f := range existingFields {
				if strings.EqualFold(f.Name, newName) {
					return repo.CustomFieldDef{}, customerrors.ErrConflict
				}
			}
		}
	}

	newIsIndexed := targetField.IsIndexed
	if isIndexed != nil {
		newIsIndexed = *isIndexed
	}

	// If no changes, return early
	if newName == targetField.Name && newIsIndexed == targetField.IsIndexed {
		return *targetField, nil
	}

	// Begin transaction
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return repo.CustomFieldDef{}, err
	}
	defer tx.Rollback()

	// Handle index changes
	tableName := fmt.Sprintf(`"entries_%s"`, dbID)
	if newIsIndexed != targetField.IsIndexed {
		if newIsIndexed {
			// Create index
			indexSQL := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS "idx_entries_%s_%s%d" ON %s("%s%d")`, dbID, customFieldsPrefix, fieldID, tableName, customFieldsPrefix, fieldID)
			if _, err := tx.ExecContext(ctx, indexSQL); err != nil {
				return repo.CustomFieldDef{}, fmt.Errorf("failed to create index: %w", err)
			}
		} else {
			// Drop index
			dropIndexSQL := fmt.Sprintf(`DROP INDEX IF EXISTS "idx_entries_%s_%s%d"`, dbID, customFieldsPrefix, fieldID)
			if _, err := tx.ExecContext(ctx, dropIndexSQL); err != nil {
				return repo.CustomFieldDef{}, fmt.Errorf("failed to drop index: %w", err)
			}
		}
	}

	// Update record
	query, args, err := r.Builder.Update("database_custom_fields").
		Set("name", newName).
		Set("is_indexed", newIsIndexed).
		Where(squirrel.Eq{"database_id": dbID, "field_id": fieldID}).
		ToSql()
	if err != nil {
		return repo.CustomFieldDef{}, err
	}

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return repo.CustomFieldDef{}, customerrors.ErrConflict
		}
		return repo.CustomFieldDef{}, fmt.Errorf("failed to update custom field record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return repo.CustomFieldDef{}, err
	}

	// Invalidate cache
	r.Cache.Delete("cf:" + dbID)

	updatedField := repo.CustomFieldDef{
		ID:        fieldID,
		Name:      newName,
		Type:      targetField.Type,
		IsIndexed: newIsIndexed,
	}
	return updatedField, nil
}

// DeleteCustomField deletes a custom field.
func (r *SQLiteRepository) DeleteCustomField(ctx context.Context, dbID string, fieldID int) error {
	// Check if database exists
	var exists bool
	err := r.DB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM databases WHERE id = ?)", dbID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}
	if !exists {
		return customerrors.ErrNotFound
	}

	// Load existing fields to check if this one exists
	existingFields, err := r.getCustomFields(ctx, r.DB, dbID)
	if err != nil {
		return err
	}

	var found bool
	for _, f := range existingFields {
		if f.ID == fieldID {
			found = true
			break
		}
	}
	if !found {
		return customerrors.ErrNotFound
	}

	// Begin transaction
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Drop the index
	dropIndexSQL := fmt.Sprintf(`DROP INDEX IF EXISTS "idx_entries_%s_%s%d"`, dbID, customFieldsPrefix, fieldID)
	if _, err := tx.ExecContext(ctx, dropIndexSQL); err != nil {
		return fmt.Errorf("failed to drop index: %w", err)
	}

	// 2. Drop column from entries table
	tableName := fmt.Sprintf(`"entries_%s"`, dbID)
	dropColSQL := fmt.Sprintf(`ALTER TABLE %s DROP COLUMN "%s%d"`, tableName, customFieldsPrefix, fieldID)
	if _, err := tx.ExecContext(ctx, dropColSQL); err != nil {
		return fmt.Errorf("failed to drop column from entries table: %w", err)
	}

	// 3. Delete from database_custom_fields
	query, args, err := r.Builder.Delete("database_custom_fields").
		Where(squirrel.Eq{"database_id": dbID, "field_id": fieldID}).
		ToSql()
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to delete custom field record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Invalidate cache
	r.Cache.Delete("cf:" + dbID)

	return nil
}
