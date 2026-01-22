package sqlite

import (
	"fmt"
	"mediahub/internal/repository"
	"mediahub/internal/shared"
	"regexp"
	"strings"
)

var safeNameRegex = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")

func (s *SQLiteRepository) CreateDatabase(db *repository.Database) (*repository.Database, error) {
	// validate database name
	if !safeNameRegex.MatchString(db.Name) {
		return nil, fmt.Errorf("sqlite database: %w", shared.ErrInvalidName)
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
		db.Housekeeping.Interval, db.Housekeeping.DiskSpace, db.Housekeeping.MaxAge.String,
		customFieldsJSON,
	)
	if err != nil {
		return nil, err
	}

	// Dynamically create the entry table based on content_type
	tableName := fmt.Sprintf("entries_%s", db.Name)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS \"%s\" (", tableName))
	sb.WriteString(`
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER NOT NULL,
		filesize INTEGER NOT NULL,
		filename TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'ready' CHECK(status IN ('processing', 'ready', 'error'))
	`)

	switch db.ContentType {
	case "image":
		sb.WriteString(`,
			width INTEGER NOT NULL,
			height INTEGER NOT NULL,
			mime_type TEXT NOT NULL CHECK(mime_type IN (
				'image/jpeg', 
				'image/png', 
				'image/gif', 
				'image/webp'
			))
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

	// Security: Whitelist custom field types and sanitize names
	allowedTypes := map[string]bool{"TEXT": true, "INTEGER": true, "REAL": true, "BOOLEAN": true}
	for _, field := range db.CustomFields {
		if !allowedTypes[field.Type] {
			return nil, fmt.Errorf("invalid custom field type: %s", field.Type)
		}
		if !safeNameRegex.MatchString(field.Name) {
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

	// Create index for status column
	indexQueryStatus := fmt.Sprintf("CREATE INDEX IF NOT EXISTS \"idx_%s_status\" ON \"%s\"(status);", tableName, tableName)
	_, err = tx.Exec(indexQueryStatus)
	if err != nil {
		return nil, err
	}

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
