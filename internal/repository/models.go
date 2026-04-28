package repository

import (
	"time"
)

type Database struct {
	ID           string // ULID
	Name         string
	ContentType  string
	Config       DatabaseConfig
	Housekeeping DatabaseHK
	CustomFields []CustomField
	Stats        DatabaseStats
}

type DatabaseConfig struct {
	CreatePreview  bool
	AutoConversion string
}

// Struct for housekeeping settings
type DatabaseHK struct {
	Interval  time.Duration
	DiskSpace uint64
	MaxAge    time.Duration
	LastHkRun time.Time // timestamp of the last housekeeping run, used to determine when the next run should occur
}

type DatabaseStats struct {
	EntryCount          uint64
	TotalDiskSpaceBytes uint64
}

// CustomField defines a custom metadata field for a database.
type CustomField struct {
	Name string
	Type string
}

type Entry struct {
	ID           int64
	FileName     string
	Size         uint64
	PreviewSize  uint64
	Timestamp    time.Time // The zero value (time.Time{}) indicates a missing timestamp
	MimeType     string
	Status       uint8          // "processing" 0x01 or "ready" 0x00 for now
	MediaFields  map[string]any // contains fields that are related to the filetype, e.g., image size
	CustomFields map[string]any
}

type User struct {
	ID           int64
	Username     string
	IsAdmin      bool
	PasswordHash string
}

// defines a role that a user has in a specific database (CanView, CanCreate, CanEdit, CanDelete)
type UserPermissions struct {
	UserID     int64
	DatabaseID string
	Roles      string // a comma separated list of roles, e.g., "CanView,CanEdit"
}

// Pagination controls the subset of results returned.
type Pagination struct {
	Offset int
	Limit  int
}

// SearchRequest defines the complex, nested filter criteria for database queries.
type SearchRequest struct {
	Filter     *FilterGroup
	Sort       *SortCriteria
	Pagination Pagination
}

// FilterGroup allows chaining multiple conditions together.
type FilterGroup struct {
	Operator   string      // e.g., "and", "or"
	Conditions []Condition // The individual rules
}

// Condition represents a single query filter.
type Condition struct {
	Field    string
	Operator string // e.g., "=", ">", "<", "LIKE"
	Value    any    // 'any' allows for strings, numbers, or booleans
}

// SortCriteria defines how the results should be ordered.
type SortCriteria struct {
	Field     string
	Direction string // "asc" or "desc"
}

// returned upon deleting an entry from the database
type DeletedEntryMeta struct {
	ID          int64
	Filesize    uint64
	PreviewSize uint64
}

type AuditLog struct {
	ID        int64     // created by the database upon writing
	Timestamp time.Time // timestamp created by the database upon writing
	Action    string
	Actor     string
	Resource  string
	Details   map[string]any
}
