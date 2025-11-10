// filepath: internal/repository/database_repo.go
package repository

import (
	"fmt"
	"mediahub/internal/models"
	"strings"
	"time"
)

// CreateDatabase creates a new database record and a corresponding entry table.
func (s *Repository) CreateDatabase(db *models.Database) (*models.Database, error) {
	// --- SECURITY: Validate database name ---
	if !SafeNameRegex.MatchString(db.Name) {
		return nil, fmt.Errorf("invalid database name: %s", db.Name)
	}

	// Start a transaction
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() // Rollback on any error

	// Insert into the main databases table
	customFieldsJSON, err := db.CustomFields.ToJSON()
	if err != nil {
		return nil, err
	}
	query := `
		INSERT INTO databases (
			name, content_type, config, hk_interval, hk_disk_space, hk_max_age, custom_fields,
			entry_count, total_disk_space_bytes
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0, 0);
	`
	_, err = tx.Exec(query,
		db.Name, db.ContentType, db.Config,
		db.Housekeeping.Interval, db.Housekeeping.DiskSpace, db.Housekeeping.MaxAge,
		customFieldsJSON,
	)
	if err != nil {
		return nil, err
	}

	// --- FIXED: Dynamically create the entry table based on content_type ---
	tableName := fmt.Sprintf("entries_%s", db.Name)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS \"%s\" (", tableName))
	sb.WriteString(`
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER NOT NULL,
		filesize INTEGER NOT NULL,
		filename TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'ready' CHECK(status IN ('processing', 'ready', 'error'))
	`) // <-- ADDED 'status' COLUMN

	switch db.ContentType {
	case "image":
		sb.WriteString(`,
			width INTEGER NOT NULL,
			height INTEGER NOT NULL,
			mime_type TEXT NOT NULL CHECK(mime_type IN ('image/jpeg', 'image/png', 'image/gif', 'image/webp'))
		`)
	case "audio":
		sb.WriteString(`,
			duration_sec REAL NOT NULL,
			channels INTEGER,
			mime_type TEXT NOT NULL CHECK(mime_type IN (
				'audio/mpeg', 
				'audio/wav', 
				'audio/flac', 
				'audio/x-flac', 
				'audio/opus', 
				'audio/ogg',
				'application/ogg'
			))
		`)
	case "file":
		sb.WriteString(`,
			mime_type TEXT NOT NULL
		`)
	default:
		// If content_type is invalid, roll back and return an error
		return nil, fmt.Errorf("invalid content_type: %s", db.ContentType)
	}

	// --- SECURITY: Whitelist custom field types and sanitize names ---
	allowedTypes := map[string]bool{"TEXT": true, "INTEGER": true, "REAL": true, "BOOLEAN": true}
	for _, field := range db.CustomFields {
		if !allowedTypes[field.Type] {
			return nil, fmt.Errorf("invalid custom field type: %s", field.Type)
		}
		if !SafeNameRegex.MatchString(field.Name) {
			return nil, fmt.Errorf("invalid custom field name: %s", field.Name)
		}
		sb.WriteString(fmt.Sprintf(", \"%s\" %s", field.Name, field.Type))
	}
	sb.WriteString(");")

	_, err = tx.Exec(sb.String())
	if err != nil {
		return nil, err
	}

	// Create an index on the timestamp column
	indexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS \"idx_%s_time\" ON \"%s\"(timestamp);", tableName, tableName)
	_, err = tx.Exec(indexQuery)
	if err != nil {
		return nil, err
	}

	// --- ADDED: Create index for status column ---
	indexQueryStatus := fmt.Sprintf("CREATE INDEX IF NOT EXISTS \"idx_%s_status\" ON \"%s\"(status);", tableName, tableName)
	_, err = tx.Exec(indexQueryStatus)
	if err != nil {
		return nil, err
	}
	// --- END ADDED ---

	// Create indexes for custom fields
	for _, field := range db.CustomFields {
		// Name is already sanitized from the loop above
		indexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS \"idx_%s_%s\" ON \"%s\"(\"%s\");", tableName, field.Name, tableName, field.Name)
		_, err = tx.Exec(indexQuery)
		if err != nil {
			return nil, err
		}
	}

	// Commit the transaction
	return db, tx.Commit()
}

// GetDatabase retrieves a single database by name.
func (s *Repository) GetDatabase(name string) (*models.Database, error) {
	query := `
		SELECT
			name, content_type, config,
			hk_interval, hk_disk_space, hk_max_age,
			custom_fields, last_hk_run,
			entry_count, total_disk_space_bytes
		FROM databases WHERE name = ?
	`
	row := s.DB.QueryRow(query, name)

	var db models.Database
	var stats models.Stats
	var customFieldsJSON string
	err := row.Scan(
		&db.Name, &db.ContentType, &db.Config,
		&db.Housekeeping.Interval, &db.Housekeeping.DiskSpace, &db.Housekeeping.MaxAge,
		&customFieldsJSON, &db.LastHkRun,
		&stats.EntryCount, &stats.TotalDiskSpaceBytes,
	)
	if err != nil {
		return nil, err
	}

	if err := db.CustomFields.FromJSON(customFieldsJSON); err != nil {
		return nil, err
	}

	db.Stats = &stats // Assign the pre-calculated stats
	return &db, nil
}

// GetDatabases retrieves all databases.
func (s *Repository) GetDatabases() ([]models.Database, error) {
	query := `
		SELECT
			name, content_type, config,
			hk_interval, hk_disk_space, hk_max_age,
			custom_fields, last_hk_run,
			entry_count, total_disk_space_bytes
		FROM databases
	`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Initialize an empty, non-nil slice to ensure JSON marshals to [] instead of null.
	databases := make([]models.Database, 0)
	for rows.Next() {
		var db models.Database
		var stats models.Stats
		var customFieldsJSON string
		err := rows.Scan(
			&db.Name, &db.ContentType, &db.Config,
			&db.Housekeeping.Interval, &db.Housekeeping.DiskSpace, &db.Housekeeping.MaxAge,
			&customFieldsJSON, &db.LastHkRun,
			&stats.EntryCount, &stats.TotalDiskSpaceBytes,
		)
		if err != nil {
			return nil, err
		}
		if err := db.CustomFields.FromJSON(customFieldsJSON); err != nil {
			return nil, err
		}

		db.Stats = &stats // Assign the pre-calculated stats
		databases = append(databases, db)
	}

	return databases, nil
}

// UpdateDatabase updates the housekeeping settings of a database.
func (s *Repository) UpdateDatabase(db *models.Database) error {
	query := `
		UPDATE databases
		SET config = ?, hk_interval = ?, hk_disk_space = ?, hk_max_age = ?
		WHERE name = ?;
	`
	_, err := s.DB.Exec(query, db.Config, db.Housekeeping.Interval, db.Housekeeping.DiskSpace, db.Housekeeping.MaxAge, db.Name)
	return err
}

// UpdateDatabaseLastHkRun updates the last_hk_run timestamp for a database to the current time.
func (s *Repository) UpdateDatabaseLastHkRun(name string) error {
	query := "UPDATE databases SET last_hk_run = ? WHERE name = ?"
	_, err := s.DB.Exec(query, time.Now(), name)
	return err
}

// DeleteDatabase deletes a database record and its corresponding entry table.
func (s *Repository) DeleteDatabase(name string) error {
	// Start a transaction
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete the database record
	query := "DELETE FROM databases WHERE name = ?"
	_, err = tx.Exec(query, name)
	if err != nil {
		return err
	}

	// Drop the entry table
	tableName := fmt.Sprintf("entries_%s", name)
	dropQuery := fmt.Sprintf("DROP TABLE IF EXISTS \"%s\";", tableName)
	_, err = tx.Exec(dropQuery)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetDatabaseStats calculates and returns the stats for a given database.
func (s *Repository) GetDatabaseStats(dbName string) (*models.Stats, error) {
	query := "SELECT entry_count, total_disk_space_bytes FROM databases WHERE name = ?"
	row := s.DB.QueryRow(query, dbName)

	var stats models.Stats
	err := row.Scan(&stats.EntryCount, &stats.TotalDiskSpaceBytes)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}
