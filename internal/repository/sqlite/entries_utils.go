package sqlite

import (
	"database/sql"
	"fmt"
	repo "mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared/customerrors"
	"strconv"
	"strings"
	"time"
)

// entryScanner holds pre-allocated slices and pre-computed field names
type entryScanner struct {
	cols           []string
	colVals        []any
	columnPointers []any
	cleanNames     []string // Pre-trimmed names for Custom/Media fields
	isCustom       []bool   // True if the column is a custom field
}

// newEntryScanner initializes the scanner once per query result.
func newEntryScanner(rows *sql.Rows) (entryScanner, error) {
	cols, err := rows.Columns()
	if err != nil {
		return entryScanner{}, err
	}

	size := len(cols)
	s := entryScanner{
		cols:           cols,
		colVals:        make([]any, size),
		columnPointers: make([]any, size),
		cleanNames:     make([]string, size),
		isCustom:       make([]bool, size),
	}

	for i, colName := range cols {
		s.columnPointers[i] = &s.colVals[i]

		// Pre-compute the prefix checks and string trims once!
		if strings.HasPrefix(colName, customFieldsPrefix) {
			s.isCustom[i] = true
			s.cleanNames[i] = strings.TrimPrefix(colName, customFieldsPrefix)
		} else {
			s.isCustom[i] = false
			s.cleanNames[i] = colName
		}
	}

	return s, nil
}

// scan maps the current row into a repo.Entry, reusing the allocated memory.
func (s entryScanner) scan(rows *sql.Rows) (repo.Entry, error) {
	if err := rows.Scan(s.columnPointers...); err != nil {
		return repo.Entry{}, err
	}

	entry := repo.Entry{
		MediaFields:  make(map[string]any),
		CustomFields: make(map[string]any),
	}

	for i, colName := range s.cols {
		val := s.colVals[i]
		if val == nil {
			continue
		}

		switch colName {
		case "id":
			entry.ID = asInt64(val)
		case "timestamp":
			tsMs := asInt64(val)
			if tsMs > 0 { // Avoid mapping 1970 if the DB returned 0
				entry.Timestamp = time.UnixMilli(tsMs)
			}
		case "created_at", "updated_at":
			// Ignored
		case "filesize":
			entry.Size = uint64(asInt64(val))
		case "preview_filesize":
			entry.PreviewSize = uint64(asInt64(val))
		case "filename":
			entry.FileName = asString(val)
		case "status":
			entry.Status = uint8(asInt64(val))
		case "mime_type":
			entry.MimeType = asString(val)
		default:
			// We MUST convert []byte to string here to prevent Base64 JSON encoding!
			if b, ok := val.([]byte); ok {
				val = string(b)
			}
			if s.isCustom[i] {
				entry.CustomFields[s.cleanNames[i]] = val
			} else {
				entry.MediaFields[s.cleanNames[i]] = val
			}
		}
	}

	return entry, nil
}

// asInt64 is a safe type-assertion helper for SQLite integer scans
func asInt64(val any) int64 {
	switch v := val.(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case []byte:
		parsed, _ := strconv.ParseInt(string(v), 10, 64)
		return parsed
	}
	return 0
}

// Helper to safely extract a string from the database interface
func asString(val any) string {
	switch v := val.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprint(v) // Fallback for safety
	}
}

// Scan a single row
func (r *SQLiteRepository) scanEntryRow(rows *sql.Rows) (repo.Entry, error) {
	scanner, err := newEntryScanner(rows)
	if err != nil {
		return repo.Entry{}, err
	}

	if !rows.Next() {
		return repo.Entry{}, customerrors.ErrNotFound
	}

	return scanner.scan(rows)
}

// Scan multiple rows
func (r *SQLiteRepository) scanEntryRows(rows *sql.Rows) ([]repo.Entry, error) {
	scanner, err := newEntryScanner(rows)
	if err != nil {
		return nil, err
	}

	var entries []repo.Entry
	for rows.Next() {
		entry, err := scanner.scan(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return entries, nil
}

// validateAndFormatSearchField prevents SQL injection by ensuring a field name exists.
func (r *SQLiteRepository) validateAndFormatSearchField(field string, customFields []repo.CustomField) (string, error) {
	// 1. Whitelist Standard Fields
	standardFields := map[string]bool{
		"id": true, "timestamp": true, "created_at": true, "updated_at": true,
		"filesize": true, "preview_filesize": true, "filename": true, "status": true, "mime_type": true,
	}
	if standardFields[field] {
		return fmt.Sprintf(`"%s"`, field), nil
	}

	// 2. Whitelist Known Media Fields (dynamically from the repository)
	for _, fields := range r.MediaFields {
		for _, mediaField := range fields {
			if mediaField.Name == field {
				return fmt.Sprintf(`"%s"`, field), nil
			}
		}
	}

	// 3. Whitelist dynamically generated Custom Fields
	for _, cf := range customFields {
		if cf.Name == field {
			return fmt.Sprintf(`"%s%s"`, customFieldsPrefix, field), nil
		}
	}

	return "", fmt.Errorf("field '%s' is not allowed or does not exist", field)
}

// isValidOperator checks if the requested SQL operator is whitelisted.
func isValidOperator(op string) bool {
	valid := map[string]bool{
		"=": true, "!=": true, ">": true, ">=": true, "<": true, "<=": true, "LIKE": true,
	}
	return valid[strings.ToUpper(op)]
}
