// filepath: internal/repository/entry_bulk_repo.go
package repository

import (
	"fmt"
	"mediahub/internal/models"

	"github.com/Masterminds/squirrel"
)

// DeletedEntryMeta holds the minimal metadata needed to cleanup files after a DB deletion.
type DeletedEntryMeta struct {
	ID        int64
	Timestamp int64
	Filesize  int64
}

// DeleteEntries performs a transactional bulk delete of entries.
// It returns the metadata of the deleted entries so the caller can clean up files on disk.
func (s *Repository) DeleteEntries(dbName string, ids []int64) ([]DeletedEntryMeta, error) {
	if !SafeNameRegex.MatchString(dbName) {
		return nil, fmt.Errorf("invalid database name: %s", dbName)
	}
	if len(ids) == 0 {
		return []DeletedEntryMeta{}, nil
	}

	tableName := fmt.Sprintf("entries_%s", dbName)

	tx, err := s.BeginTx()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Pre-calculation: Fetch ID, Timestamp, and Filesize for the requested IDs.
	// We need filesize to update stats, and timestamp/id for file deletion.
	query := s.Builder.Select("id", "timestamp", "filesize").
		From(fmt.Sprintf("\"%s\"", tableName)).
		Where(squirrel.Eq{"id": ids})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	rows, err := tx.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries for deletion: %w", err)
	}
	defer rows.Close()

	var deletedMeta []DeletedEntryMeta
	var totalFilesize int64
	var foundIDs []int64

	for rows.Next() {
		var meta DeletedEntryMeta
		var size int64
		if err := rows.Scan(&meta.ID, &meta.Timestamp, &size); err != nil {
			return nil, fmt.Errorf("failed to scan entry: %w", err)
		}
		// Set the struct field
		meta.Filesize = size

		deletedMeta = append(deletedMeta, meta)
		foundIDs = append(foundIDs, meta.ID)
		totalFilesize += size
	}

	if len(foundIDs) == 0 {
		return []DeletedEntryMeta{}, nil // Nothing found to delete
	}

	// 2. Batch Delete
	// We use the foundIDs instead of the requested ids to ensure we only delete what we found.
	deleteQuery := s.Builder.Delete(fmt.Sprintf("\"%s\"", tableName)).
		Where(squirrel.Eq{"id": foundIDs})

	sqlDelete, argsDelete, err := deleteQuery.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build delete query: %w", err)
	}

	_, err = tx.Exec(sqlDelete, argsDelete...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute bulk delete: %w", err)
	}

	// 3. Update Database Stats
	// Subtract the count and total size of the deleted entries.
	if err := tx.UpdateStatsInTx(dbName, -len(foundIDs), -totalFilesize); err != nil {
		return nil, fmt.Errorf("failed to update db stats: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return deletedMeta, nil
}

// GetEntriesByID retrieves a list of entries matching the provided IDs.
func (s *Repository) GetEntriesByID(dbName string, ids []int64, customFields []models.CustomField) ([]models.Entry, error) {
	if !SafeNameRegex.MatchString(dbName) {
		return nil, fmt.Errorf("invalid database name: %s", dbName)
	}
	if len(ids) == 0 {
		return []models.Entry{}, nil
	}

	tableName := fmt.Sprintf("entries_%s", dbName)

	query := s.Builder.Select("*").
		From(fmt.Sprintf("\"%s\"", tableName)).
		Where(squirrel.Eq{"id": ids})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := s.DB.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	entries := make([]models.Entry, 0)
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}
