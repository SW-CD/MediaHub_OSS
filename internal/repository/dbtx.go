// filepath: internal/repository/dbtx.go
package repository

import (
	"database/sql"
	"fmt"
	"mediahub/internal/models"
	"strings"
)

// Tx is a wrapper around *sql.Tx that provides transactional database operations.
type Tx struct {
	*sql.Tx
}

// CreateEntryInTx creates a new entry record within a transaction.
func (tx *Tx) CreateEntryInTx(dbName string, contentType string, entry models.Entry, customFields []models.CustomField) (models.Entry, error) {
	if !SafeNameRegex.MatchString(dbName) {
		return nil, fmt.Errorf("invalid database name: %s", dbName)
	}
	tableName := fmt.Sprintf("entries_%s", dbName)

	var sbFields strings.Builder
	var sbValues strings.Builder
	args := []interface{}{}

	// --- UPDATED: Added 'status' field ---
	sbFields.WriteString("timestamp, filesize, mime_type, filename, status")
	sbValues.WriteString("?, 0, ?, ?, ?") // Filesize is 0 initially, updated later
	args = append(args, entry["timestamp"], entry["mime_type"], entry["filename"], entry["status"])
	// --- END UPDATE ---

	switch contentType {
	case "image":
		sbFields.WriteString(", width, height")
		sbValues.WriteString(", ?, ?")
		args = append(args, entry["width"], entry["height"])
	case "audio":
		sbFields.WriteString(", duration_sec, channels")
		sbValues.WriteString(", ?, ?")
		args = append(args, entry["duration_sec"], entry["channels"])
	}

	// Add custom fields
	for _, field := range customFields {
		if val, ok := entry[field.Name]; ok && val != nil {
			sbFields.WriteString(fmt.Sprintf(", \"%s\"", field.Name))
			sbValues.WriteString(", ?")
			args = append(args, val)
		}
	}

	query := fmt.Sprintf("INSERT INTO \"%s\" (%s) VALUES (%s)", tableName, sbFields.String(), sbValues.String())

	res, err := tx.Exec(query, args...)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	entry["id"] = id // Add the generated ID to the returned map

	return entry, nil
}

// UpdateEntryInTx updates an existing entry record within a transaction.
func (tx *Tx) UpdateEntryInTx(dbName string, id int64, updates models.Entry, customFields []models.CustomField) error {
	if !SafeNameRegex.MatchString(dbName) {
		return fmt.Errorf("invalid database name: %s", dbName)
	}
	tableName := fmt.Sprintf("entries_%s", dbName)

	var sbSet strings.Builder
	args := []interface{}{}

	// Build a map of allowed fields (standard + custom) for validation
	allowedUpdateFields := map[string]bool{
		"timestamp": true, "width": true, "height": true,
		"filesize": true, "mime_type": true,
		"duration_sec": true, "channels": true,
		"filename": true,
		"status":   true, // --- ADDED: Allow internal updates to 'status' ---
	}
	for _, cf := range customFields {
		allowedUpdateFields[cf.Name] = true
	}

	for key, val := range updates {
		if !allowedUpdateFields[key] {
			continue
		}
		if sbSet.Len() > 0 {
			sbSet.WriteString(", ")
		}
		sbSet.WriteString(fmt.Sprintf("\"%s\" = ?", key))
		args = append(args, val)
	}

	if sbSet.Len() == 0 {
		return nil // Nothing valid to update
	}

	query := fmt.Sprintf("UPDATE \"%s\" SET %s WHERE id = ?", tableName, sbSet.String())
	args = append(args, id)

	_, err := tx.Exec(query, args...)
	return err
}

// UpdateStatsInTx atomically updates the denormalized stats for a database.
func (tx *Tx) UpdateStatsInTx(dbName string, entryCountDelta int, sizeDelta int64) error {
	query := `
		UPDATE databases
		SET
			entry_count = entry_count + ?,
			total_disk_space_bytes = total_disk_space_bytes + ?
		WHERE name = ?
	`
	_, err := tx.Exec(query, entryCountDelta, sizeDelta, dbName)
	return err
}
