package shared

// CustomField defines a custom metadata field for a database.
type CustomField struct {
	Name string `json:"name"`
	Type string `json:"type"` // Supported types: TEXT, INTEGER, REAL, BOOLEAN
}

// CustomFields is a slice of CustomField.
type CustomFields []CustomField

// defines a role that a user has in a specific database (can_view, can_edit, can_delete, can_admin)
type Role struct {
	DatabaseID uint   `toml:"database_id"`
	Role       string `toml:"role"`
}

// Housekeeping defines the automated maintenance policies for a database.
type Housekeeping struct {
	Interval  string `json:"interval"`
	DiskSpace string `json:"disk_space"`
	MaxAge    string `json:"max_age"`
}
