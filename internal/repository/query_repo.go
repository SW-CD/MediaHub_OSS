// filepath: internal/repository/query_repo.go
package repository

import (
	"fmt"
	"mediahub/internal/logging"
	"mediahub/internal/models"
	"strings"

	"github.com/Masterminds/squirrel"
)

func (s *Repository) GetEntries(dbName string, limit, offset int, order string, tstart, tend int64, customFields []models.CustomField) ([]models.Entry, error) {
	if !SafeNameRegex.MatchString(dbName) {
		return nil, fmt.Errorf("invalid database name: %s", dbName)
	}
	tableName := fmt.Sprintf("entries_%s", dbName)

	// Use Squirrel Builder
	query := s.Builder.Select("*").From(fmt.Sprintf("\"%s\"", tableName))

	// Time range filters
	if tstart > 0 {
		query = query.Where(squirrel.GtOrEq{"timestamp": tstart})
	}
	if tend > 0 {
		query = query.Where(squirrel.LtOrEq{"timestamp": tend})
	}

	// Append ordering and pagination
	if strings.ToLower(order) != "asc" {
		order = "DESC"
	} else {
		order = "ASC"
	}
	query = query.OrderBy(fmt.Sprintf("timestamp %s", order))
	query = query.Limit(uint64(limit)).Offset(uint64(offset))

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	logging.Log.Debugf("Generated SQL for GetEntries: %s", sql)
	logging.Log.Debugf("Arguments: %v", args)

	rows, err := s.DB.Query(sql, args...)
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

	// 1. Build Field Whitelist
	allowedFields := map[string]bool{
		"id":           true,
		"timestamp":    true,
		"width":        true,
		"height":       true,
		"filesize":     true,
		"mime_type":    true,
		"duration_sec": true,
		"channels":     true,
		"filename":     true,
		"status":       true,
	}
	for _, cf := range customFields {
		allowedFields[cf.Name] = true
	}

	// 2. Start Query Builder
	query := s.Builder.Select("*").From(fmt.Sprintf("\"%s\"", tableName))

	// 3. Build WHERE clause recursively
	if req.Filter != nil {
		whereExpr, err := buildWhereExpr(req.Filter, allowedFields, customFields)
		if err != nil {
			return nil, err
		}
		if whereExpr != nil {
			query = query.Where(whereExpr)
		}
	}

	// 4. Build ORDER BY clause
	if req.Sort != nil && req.Sort.Field != "" {
		if !allowedFields[req.Sort.Field] {
			return nil, fmt.Errorf("%w: invalid sort field: %s", ErrInvalidFilter, req.Sort.Field)
		}
		sortDir := "DESC" // Default
		if strings.ToLower(req.Sort.Direction) == "asc" {
			sortDir = "ASC"
		}
		query = query.OrderBy(fmt.Sprintf("\"%s\" %s", req.Sort.Field, sortDir))
	} else {
		// Default sort
		query = query.OrderBy("timestamp DESC")
	}

	// 5. Build LIMIT / OFFSET clause
	query = query.Limit(uint64(*req.Pagination.Limit)).Offset(uint64(req.Pagination.Offset))

	// 6. Generate SQL
	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL: %w", err)
	}

	logging.Log.Debugf("Generated SQL for SearchEntries: %s", sql)
	logging.Log.Debugf("Arguments: %v", args)

	// 7. Execute Query
	rows, err := s.DB.Query(sql, args...)
	if err != nil {
		logging.Log.Errorf("Error executing SearchEntries query: %v", err)
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

// buildWhereExpr recursively constructs a Squirrel Sqlizer expression.
func buildWhereExpr(filter *models.SearchFilter, fields map[string]bool, customFields []models.CustomField) (squirrel.Sqlizer, error) {
	op := strings.ToLower(filter.Operator)

	// Case 1: Logical Operator (AND/OR)
	if op == "and" || op == "or" {
		if len(filter.Conditions) == 0 {
			return nil, nil // Empty logic group ignores
		}

		var conditions []squirrel.Sqlizer
		for _, cond := range filter.Conditions {
			expr, err := buildWhereExpr(cond, fields, customFields)
			if err != nil {
				return nil, err
			}
			if expr != nil {
				conditions = append(conditions, expr)
			}
		}

		if len(conditions) == 0 {
			return nil, nil
		}

		if op == "and" {
			return squirrel.And(conditions), nil
		}
		return squirrel.Or(conditions), nil
	}

	// Case 2: Comparison Operator
	// Whitelist check
	if !fields[filter.Field] {
		return nil, fmt.Errorf("%w: invalid filter field: %s", ErrInvalidFilter, filter.Field)
	}

	fieldType := getFieldType(filter.Field, customFields)
	upperOp := strings.ToUpper(filter.Operator)

	if !isOperatorAllowedForType(upperOp, fieldType) {
		return nil, fmt.Errorf("%w: operator '%s' is not allowed for field '%s' (type %s)", ErrInvalidFilter, filter.Operator, filter.Field, fieldType)
	}

	// Value preparation
	val := filter.Value
	if fieldType == "BOOLEAN" && (upperOp == "=" || upperOp == "!=") {
		if boolVal, ok := val.(bool); ok {
			if boolVal {
				val = 1
			} else {
				val = 0
			}
		}
	}

	// Squirrel Expressions
	switch upperOp {
	case "=":
		return squirrel.Eq{filter.Field: val}, nil
	case "!=":
		return squirrel.NotEq{filter.Field: val}, nil
	case ">":
		return squirrel.Gt{filter.Field: val}, nil
	case ">=":
		return squirrel.GtOrEq{filter.Field: val}, nil
	case "<":
		return squirrel.Lt{filter.Field: val}, nil
	case "<=":
		return squirrel.LtOrEq{filter.Field: val}, nil
	case "LIKE":
		return squirrel.Like{filter.Field: fmt.Sprintf("%%%v%%", val)}, nil // Auto-wrap wildcards
	default:
		return nil, fmt.Errorf("%w: invalid operator: %s", ErrInvalidFilter, filter.Operator)
	}
}

// getFieldType determines the type (TEXT, INTEGER, REAL, BOOLEAN) of a field.
func getFieldType(fieldName string, customFields []models.CustomField) string {
	standardTypes := map[string]string{
		"id":           "INTEGER",
		"timestamp":    "INTEGER",
		"width":        "INTEGER",
		"height":       "INTEGER",
		"filesize":     "INTEGER",
		"mime_type":    "TEXT",
		"duration_sec": "REAL",
		"channels":     "INTEGER",
		"filename":     "TEXT",
		"status":       "TEXT",
	}

	if stdType, ok := standardTypes[fieldName]; ok {
		return stdType
	}

	for _, cf := range customFields {
		if cf.Name == fieldName {
			return cf.Type
		}
	}
	return "UNKNOWN"
}

// isOperatorAllowedForType checks if an operator is valid for a given type.
func isOperatorAllowedForType(operator string, fieldType string) bool {
	if operator == "LIKE" {
		return fieldType == "TEXT"
	}
	if operator == "=" || operator == "!=" {
		return true
	}
	if operator == ">" || operator == ">=" || operator == "<" || operator == "<=" {
		return fieldType == "INTEGER" || fieldType == "REAL"
	}
	return false
}
