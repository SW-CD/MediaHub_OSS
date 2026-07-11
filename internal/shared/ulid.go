package shared

import (
	ulid "github.com/oklog/ulid/v2"
)

// GenerateULID generates a universally unique lexicographically sortable identifier.
func GenerateULID() string {
	// ulid.Make() is safe for concurrent use, automatically captures the current time,
	// and uses secure randomness for the entropy portion.
	return ulid.Make().String()
}

// IsValidULID checks if the string is a valid ULID.
func IsValidULID(s string) bool {
	_, err := ulid.ParseStrict(s)
	return err == nil
}
