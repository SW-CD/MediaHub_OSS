// filepath: internal/repository/utils.go
package repository

import (
	"database/sql"
	"mediahub/internal/models"
)

// scanEntry is a helper function to scan a row into an Entry map.
// It dynamically handles standard and custom columns.
func scanEntry(rows *sql.Rows) (models.Entry, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// Create slices to hold the pointers to the scanned data.
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}

	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	entry := make(models.Entry)
	for i, col := range columns {
		val := values[i]
		// Convert byte slices (TEXT columns) to strings for easier handling.
		if b, ok := val.([]byte); ok {
			entry[col] = string(b)
		} else {
			entry[col] = val
		}
	}

	return entry, nil
}
