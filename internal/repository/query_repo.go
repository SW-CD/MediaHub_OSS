// filepath: internal/repository/query_repo.go
package repository

import (
	"errors"
	"fmt"
	"mediahub/internal/logging"
	"mediahub/internal/models"

	"strings"
)

func (s *Repository) GetEntries(dbName string, limit, offset int, order string, tstart, tend int64, customFields []models.CustomField) ([]models.Entry, error) {
	if !SafeNameRegex.MatchString(dbName) {
		return nil, fmt.Errorf("invalid database name: %s", dbName)
	}
	tableName := fmt.Sprintf("entries_%s", dbName)

	var sbQuery strings.Builder
	sbQuery.WriteString(fmt.Sprintf("SELECT * FROM \"%s\" WHERE 1=1", tableName))
	args := []interface{}{}

	// Time range filters
	if tstart > 0 {
		sbQuery.WriteString(" AND timestamp >= ?")
		args = append(args, tstart)
	}
	if tend > 0 {
		sbQuery.WriteString(" AND timestamp <= ?")
		args = append(args, tend)
	}

	// Append ordering and pagination
	if strings.ToLower(order) != "asc" {
		order = "desc"
	}
	sbQuery.WriteString(fmt.Sprintf(" ORDER BY timestamp %s LIMIT ? OFFSET ?", order))
	args = append(args, limit, offset)

	logging.Log.Debugf("Generated SQL for GetEntries: %s", sbQuery.String())
	logging.Log.Debugf("Arguments: %v", args)

	rows, err := s.DB.Query(sbQuery.String(), args...)
	if err != nil {
		logging.Log.Errorf("Error executing GetEntries query: %v", err)
		return nil, err
	}
	defer rows.Close()

	entries := make([]models.Entry, 0)
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			logging.Log.Errorf("Error scanning entry row: %v", err)
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err = rows.Err(); err != nil {
		logging.Log.Errorf("Error during rows iteration: %v", err)
		return nil, err
	}

	return entries, nil
}

// SearchEntries performs a complex, filtered search on a specific database.
func (s *Repository) SearchEntries(dbName string, req *models.SearchRequest, customFields []models.CustomField) ([]models.Entry, error) {
	if !SafeNameRegex.MatchString(dbName) {
		return nil, fmt.Errorf("invalid database name: %s", dbName)
	}
	tableName := fmt.Sprintf("entries_%s", dbName)
	var sbQuery strings.Builder
	var args []interface{}

	// 1. Build Field Whitelist
	allowedFields := map[string]bool{
		"id": true, "timestamp": true, "width": true, "height": true, "filesize": true, "mime_type": true, "duration_sec": true, "channels": true,
	}
	for _, cf := range customFields {
		allowedFields[cf.Name] = true
	}

	// 2. Build Operator Whitelists
	logicalOps := map[string]bool{"and": true, "or": true}
	// --- ADD LIKE and != to comparisonOps ---
	comparisonOps := map[string]bool{
		"=": true, ">": true, ">=": true, "<": true, "<=": true,
		"!=":   true, // Added Not Equals
		"LIKE": true, // Added LIKE for contains
	}

	// 3. Build WHERE clause
	sbQuery.WriteString(fmt.Sprintf("SELECT * FROM \"%s\" WHERE 1=1", tableName))
	if req.Filter != nil {
		// --- PASS customFields INTO buildWhereClause ---
		whereClause, whereArgs, err := buildWhereClause(req.Filter, allowedFields, logicalOps, comparisonOps, customFields)
		if err != nil {
			return nil, err // This error is user-facing (e.g., "invalid field")
		}
		if whereClause != "" {
			sbQuery.WriteString(" AND (")
			sbQuery.WriteString(whereClause)
			sbQuery.WriteString(")")
			args = append(args, whereArgs...)
		}
	}

	// 4. Build ORDER BY clause
	if req.Sort != nil && req.Sort.Field != "" {
		if !allowedFields[req.Sort.Field] {
			return nil, fmt.Errorf("%w: invalid sort field: %s", ErrInvalidFilter, req.Sort.Field) // <-- WRAP
		}
		sortDir := "DESC" // Default
		if strings.ToLower(req.Sort.Direction) == "asc" {
			sortDir = "ASC"
		}
		sbQuery.WriteString(fmt.Sprintf(" ORDER BY \"%s\" %s", req.Sort.Field, sortDir))
	} else {
		// Default sort
		sbQuery.WriteString(" ORDER BY timestamp DESC")
	}

	// 5. Build LIMIT / OFFSET clause (Limit is guaranteed to be non-nil by handler)
	sbQuery.WriteString(" LIMIT ? OFFSET ?")
	args = append(args, *req.Pagination.Limit, req.Pagination.Offset)

	// 6. Execute Query
	logging.Log.Debugf("Generated SQL for SearchEntries: %s", sbQuery.String())
	logging.Log.Debugf("Arguments: %v", args)

	rows, err := s.DB.Query(sbQuery.String(), args...)
	if err != nil {
		logging.Log.Errorf("Error executing SearchEntries query: %v", err)
		return nil, err // This is a 500 error
	}
	defer rows.Close()

	entries := make([]models.Entry, 0)
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			logging.Log.Errorf("Error scanning entry row: %v", err)
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err = rows.Err(); err != nil {
		logging.Log.Errorf("Error during rows iteration: %v", err)
		return nil, err
	}

	return entries, nil
}

// buildWhereClause recursively constructs a parameterized WHERE clause.
// --- ADD customFields parameter ---
func buildWhereClause(filter *models.SearchFilter, fields map[string]bool, logicalOps map[string]bool, compOps map[string]bool, customFields []models.CustomField) (string, []interface{}, error) {
	// Case 1: Logical Operator (AND/OR)
	if logicalOps[strings.ToLower(filter.Operator)] {
		if len(filter.Conditions) == 0 {
			return "", nil, nil // Empty AND/OR group is valid, just returns nothing
		}

		var sbGroup strings.Builder
		var args []interface{}
		sqlOp := strings.ToUpper(filter.Operator) // "AND" or "OR"

		// --- Use _ to ignore the loop index ---
		for _, cond := range filter.Conditions {
			// Recurse
			// --- PASS customFields DOWN ---
			childClause, childArgs, err := buildWhereClause(cond, fields, logicalOps, compOps, customFields)
			if err != nil {
				return "", nil, err // Propagate error (e.g., invalid field deeper down)
			}
			if childClause == "" {
				continue // Skip empty conditions generated by recursion
			}

			if sbGroup.Len() > 0 {
				sbGroup.WriteString(fmt.Sprintf(" %s ", sqlOp))
			}
			sbGroup.WriteString("(") // Add parentheses for correct precedence
			sbGroup.WriteString(childClause)
			sbGroup.WriteString(")")
			args = append(args, childArgs...)
		}
		return sbGroup.String(), args, nil
	}

	// Case 2: Comparison Operator (=, >, !=, LIKE etc.)
	// --- USE ToUpper for case-insensitive operator matching ---
	upperOperator := strings.ToUpper(filter.Operator)
	if compOps[upperOperator] {
		// A comparison *must* have a field
		if filter.Field == "" {
			return "", nil, errors.New("filter condition is missing 'field'")
		}
		// Security Check: Whitelist the field name
		if !fields[filter.Field] {
			// User provided a field name not allowed for this table
			return "", nil, fmt.Errorf("%w: invalid filter field: %s", ErrInvalidFilter, filter.Field) // <-- WRAP
		}

		// --- Check if the operator is allowed for the field type ---
		fieldType := getFieldType(filter.Field, customFields)
		if !isOperatorAllowedForType(upperOperator, fieldType) {
			return "", nil, fmt.Errorf("%w: operator '%s' is not allowed for field '%s' (type %s)", ErrInvalidFilter, filter.Operator, filter.Field, fieldType) // <-- WRAP
		}
		// --- End Type Check ---

		// Prepare value for parameterization
		var queryValue interface{} = filter.Value

		// --- Specific handling for LIKE ---
		if upperOperator == "LIKE" {
			// LIKE requires a string value and wildcards
			stringValue, ok := filter.Value.(string)
			if !ok {
				// Attempt conversion if possible, or return error
				stringValue = fmt.Sprintf("%v", filter.Value)
				// Consider returning an error if strict type matching is desired for LIKE:
				// return "", nil, fmt.Errorf("value for LIKE operator on field '%s' must be a string", filter.Field)
			}
			queryValue = "%" + stringValue + "%" // Add wildcards
			logging.Log.Debugf("LIKE operator detected. Value transformed to: %v", queryValue)
		} else {
			// For boolean fields with = or !=, convert Go bool to SQLite int
			if fieldType == "BOOLEAN" && (upperOperator == "=" || upperOperator == "!=") {
				if boolVal, ok := filter.Value.(bool); ok {
					if boolVal {
						queryValue = 1
					} else {
						queryValue = 0
					}
					logging.Log.Debugf("Boolean operator detected. Value transformed to: %v", queryValue)
				}
				// If the value isn't a bool, pass it through, SQLite might handle it or error
			}
		}

		// Build the clause: Field name is quoted, operator comes from whitelist, value is parameterized
		// Example: `"ml_score" > ?`, `"description" LIKE ?`
		clause := fmt.Sprintf("\"%s\" %s ?", filter.Field, upperOperator)
		args := []interface{}{queryValue} // Value goes into args for parameterization
		return clause, args, nil
	}

	// Case 3: Handle malformed conditions specifically
	if filter.Field != "" && filter.Operator == "" {
		return "", nil, fmt.Errorf("%w: filter condition for field '%s' is missing 'operator'", ErrInvalidFilter, filter.Field) // <-- WRAP
	}

	// Case 4: Invalid operator if none of the above matched
	if filter.Operator != "" {
		return "", nil, fmt.Errorf("%w: invalid or unsupported operator: %s", ErrInvalidFilter, filter.Operator) // <-- WRAP
	}

	// If filter is completely empty or malformed
	return "", nil, fmt.Errorf("%w: malformed filter condition", ErrInvalidFilter) // <-- WRAP

}

// --- NEW HELPER: getFieldType ---
// Helper to determine the type (TEXT, INTEGER, REAL, BOOLEAN) of a field.
func getFieldType(fieldName string, customFields []models.CustomField) string {
	// Standard fields (approximated types for filtering purposes)
	standardTypes := map[string]string{
		"id":           "INTEGER",
		"timestamp":    "INTEGER",
		"width":        "INTEGER",
		"height":       "INTEGER",
		"filesize":     "INTEGER",
		"mime_type":    "TEXT",
		"duration_sec": "REAL",
		"channels":     "INTEGER",
	}

	if stdType, ok := standardTypes[fieldName]; ok {
		return stdType
	}

	// Check custom fields
	for _, cf := range customFields {
		if cf.Name == fieldName {
			return cf.Type // Return the declared type (TEXT, INTEGER, REAL, BOOLEAN)
		}
	}

	return "UNKNOWN" // Should not happen if field was whitelisted
}

// --- NEW HELPER: isOperatorAllowedForType ---
// Checks if a given operator is suitable for a field type.
func isOperatorAllowedForType(operator string, fieldType string) bool {
	// LIKE is only for TEXT
	if operator == "LIKE" {
		return fieldType == "TEXT"
	}

	// =, != are allowed for all types (including BOOLEAN mapped to INTEGER)
	if operator == "=" || operator == "!=" {
		return true
	}

	// >, >=, <, <= are only for numeric types (INTEGER, REAL) and potentially timestamp (INTEGER)
	if operator == ">" || operator == ">=" || operator == "<" || operator == "<=" {
		return fieldType == "INTEGER" || fieldType == "REAL"
	}

	// Should not reach here if operator was whitelisted
	return false
}
