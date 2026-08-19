package repository

import (
	"fmt"
	"slices"
	"strings"
)

// IsValidCustomFieldType checks if the given type is a supported canonical type.
func IsValidCustomFieldType(t string) bool {
	var validCustomFieldTypes = []string{"TEXT", "INTEGER", "REAL", "BOOLEAN"}
	return slices.Contains(validCustomFieldTypes, strings.ToUpper(strings.TrimSpace(t)))
}

// NormalizeCustomFieldType converts type aliases into a canonical uppercase type ("TEXT", "INTEGER", "REAL", "BOOLEAN").
// Returns an error if the type cannot be mapped.
func NormalizeCustomFieldType(t string) (string, error) {
	norm := strings.ToUpper(strings.TrimSpace(t))
	switch norm {
	case "TEXT", "STRING", "STR", "VARCHAR":
		return "TEXT", nil
	case "INTEGER", "INT", "INT32", "INT64", "UINT", "UINT32", "UINT64", "BIGINT", "SMALLINT":
		return "INTEGER", nil
	case "REAL", "FLOAT", "FLOAT32", "FLOAT64", "DOUBLE", "DOUBLE PRECISION", "DECIMAL", "NUMERIC":
		return "REAL", nil
	case "BOOLEAN", "BOOL":
		return "BOOLEAN", nil
	default:
		return "", fmt.Errorf("unsupported custom field type '%s', must be TEXT, INTEGER, REAL, or BOOLEAN", t)
	}
}
