// filepath: internal/repository/entry_repo.go
package repository

import (
	"database/sql"
	"fmt"
	"mediahub/internal/models"
)

// UpdateEntry updates an entry's metadata in the correct table.
func (s *Repository) UpdateEntry(dbName string, id int64, updates models.Entry, customFields []models.CustomField) error {
	tx, err := s.BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.UpdateEntryInTx(dbName, id, updates, customFields); err != nil {
		return err
	}

	return tx.Commit()
}

// GetEntry retrieves a single entry by its ID from the correct table.
func (s *Repository) GetEntry(dbName string, id int64, customFields []models.CustomField) (models.Entry, error) {
	if !SafeNameRegex.MatchString(dbName) {
		return nil, fmt.Errorf("invalid database name: %s", dbName)
	}
	tableName := fmt.Sprintf("entries_%s", dbName)
	query := fmt.Sprintf("SELECT * FROM \"%s\" WHERE id = ?", tableName)

	rows, err := s.DB.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}

	return scanEntry(rows)
}

// GetEntriesOlderThan retrieves all entries in a database older than a given timestamp.
func (s *Repository) GetEntriesOlderThan(dbName string, timestamp int64, customFields []models.CustomField) ([]models.Entry, error) {
	if !SafeNameRegex.MatchString(dbName) {
		return nil, fmt.Errorf("invalid database name: %s", dbName)
	}
	tableName := fmt.Sprintf("entries_%s", dbName)
	query := fmt.Sprintf("SELECT * FROM \"%s\" WHERE timestamp < ?", tableName)
	rows, err := s.DB.Query(query, timestamp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]models.Entry, 0)
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// GetOldestEntries retrieves a batch of the oldest entries from a database, with an offset.
func (s *Repository) GetOldestEntries(dbName string, limit, offset int, customFields []models.CustomField) ([]models.Entry, error) {
	if !SafeNameRegex.MatchString(dbName) {
		return nil, fmt.Errorf("invalid database name: %s", dbName)
	}
	tableName := fmt.Sprintf("entries_%s", dbName)
	query := fmt.Sprintf("SELECT * FROM \"%s\" ORDER BY timestamp ASC LIMIT ? OFFSET ?", tableName)
	rows, err := s.DB.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]models.Entry, 0)
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// DeleteEntry deletes an entry by its ID from the correct table
// --- UPDATED: This is now transactional and updates stats ---
func (s *Repository) DeleteEntry(dbName string, id int64) error {
	if !SafeNameRegex.MatchString(dbName) {
		return fmt.Errorf("invalid database name: %s", dbName)
	}
	tableName := fmt.Sprintf("entries_%s", dbName)

	tx, err := s.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Get the filesize of the entry *before* deleting it
	var filesize int64
	query := fmt.Sprintf("SELECT filesize FROM \"%s\" WHERE id = ?", tableName)
	err = tx.QueryRow(query, id).Scan(&filesize)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("entry not found") // Already deleted
		}
		return fmt.Errorf("failed to get entry filesize for delete: %w", err)
	}

	// 2. Delete the entry
	query = fmt.Sprintf("DELETE FROM \"%s\" WHERE id = ?", tableName)
	_, err = tx.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete entry record: %w", err)
	}

	// 3. Update the denormalized stats
	if err := tx.UpdateStatsInTx(dbName, -1, -filesize); err != nil {
		return fmt.Errorf("failed to update stats on delete: %w", err)
	}

	return tx.Commit()
}
