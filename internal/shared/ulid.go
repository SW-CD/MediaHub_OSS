package shared

import (
	ulid "github.com/oklog/ulid/v2"
)

// GenerateULID generates a universally unique lexicographically sortable identifier.
// It returns the 26-character string representation of the ULID.
func GenerateULID() string {
	// ulid.Make() is safe for concurrent use, automatically captures the current time,
	// and uses secure randomness for the entropy portion.
	return ulid.Make().String()
}
