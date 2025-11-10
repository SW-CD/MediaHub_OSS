// internal/api/handlers/utils.go
package handlers

import (
	"strconv"
	"time"
)

// parseTimestamp tries to parse a string into a Unix timestamp.
// It supports RFC3339 and Unix timestamp formats.
func parseTimestamp(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	// Try parsing as Unix timestamp first
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return ts, nil
	}
	// Try parsing as RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Unix(), nil
	}
	return 0, &time.ParseError{Layout: "RFC3339 or Unix", Value: s}
}
