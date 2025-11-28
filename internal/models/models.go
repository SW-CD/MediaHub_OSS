// filepath: internal/models/models.go
// Package models contains the core data structures for the application.
package models

import (
	"encoding/json"
	"time"
)

// Info represents general information about the service.
type Info struct {
	ServiceName      string    `json:"service_name"`
	Version          string    `json:"version"`
	UptimeSince      time.Time `json:"uptime_since"`
	FFmpegAvailable  bool      `json:"ffmpeg"`
	FFprobeAvailable bool      `json:"ffprobe"`
}

// MediaMetadata holds extracted technical metadata from a file.
type MediaMetadata struct {
	Width       int
	Height      int
	DurationSec float64
	Title       string
	Artist      string
	Album       string
	Genre       string
	Channels    int `json:"channels,omitempty"`
}

// CustomField defines a custom metadata field for a database.
type CustomField struct {
	Name string `json:"name"`
	Type string `json:"type"` // Supported types: TEXT, INTEGER, REAL, BOOLEAN
}

// CustomFields is a slice of CustomField.
type CustomFields []CustomField

// Housekeeping defines the automated maintenance policies for a database.
type Housekeeping struct {
	Interval  string `json:"interval"`
	DiskSpace string `json:"disk_space"`
	MaxAge    string `json:"max_age"`
}

// Stats provides live statistics for a database.
type Stats struct {
	EntryCount          int   `json:"entry_count"`
	TotalDiskSpaceBytes int64 `json:"total_disk_space_bytes"`
}

// DatabaseConfig holds the parsed JSON config from the databases table.
type DatabaseConfig struct {
	CreatePreview  bool   `json:"create_preview"`
	ConvertToJPEG  bool   `json:"convert_to_jpeg"`
	AutoConversion string `json:"auto_conversion"`
}

// Database represents the configuration and metadata for a single database.
type Database struct {
	Name         string          `json:"name"`
	ContentType  string          `json:"content_type"`
	Config       json.RawMessage `json:"config" swaggertype:"object"`
	Housekeeping Housekeeping    `json:"housekeeping"`
	CustomFields CustomFields    `json:"custom_fields"`
	Stats        *Stats          `json:"stats,omitempty"`
	LastHkRun    time.Time       `json:"last_hk_run"`
}

// DatabaseCreatePayload is used for the POST /api/database request.
type DatabaseCreatePayload struct {
	Name         string                 `json:"name"`
	ContentType  string                 `json:"content_type"`
	Config       map[string]interface{} `json:"config"`
	Housekeeping *Housekeeping          `json:"housekeeping"`
	CustomFields []CustomField          `json:"custom_fields"`
}

// DatabaseUpdatePayload is used for the PUT /api/database request.
type DatabaseUpdatePayload struct {
	Config       map[string]interface{} `json:"config,omitempty"`
	Housekeeping *Housekeeping          `json:"housekeeping,omitempty"`
}

// Entry represents the metadata for a single entry stored in the system.
// It uses a map to hold custom fields, allowing for dynamic properties.
type Entry map[string]interface{}

// PartialEntryResponse is returned for a new entry that is still processing.
type PartialEntryResponse struct {
	ID           int64  `json:"id"`
	Timestamp    int64  `json:"timestamp"`
	DatabaseName string `json:"database_name"`
	Status       string `json:"status"`        // Will be "processing"
	CustomFields Entry  `json:"custom_fields"` // Only fields provided by user
}

// FileJSONResponse is used when clients request a file via Accept: application/json.
// This is used for both /entry/file and /entry/preview endpoints.
type FileJSONResponse struct {
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Data     string `json:"data"` // Base64 encoded string with data URI prefix
}

// User represents a user account in the system.
type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"` // Omit from JSON responses
	CanView      bool   `json:"can_view"`
	CanCreate    bool   `json:"can_create"`
	CanEdit      bool   `json:"can_edit"`
	CanDelete    bool   `json:"can_delete"`
	IsAdmin      bool   `json:"is_admin"`
}

// HousekeepingReport summarizes the results of a housekeeping run.
type HousekeepingReport struct {
	DatabaseName    string `json:"database_name"`
	EntriesDeleted  int    `json:"entries_deleted"`
	SpaceFreedBytes int64  `json:"space_freed_bytes"`
	Message         string `json:"message"`
}

// ToJSON converts a slice of CustomField to its JSON string representation.
func (cf CustomFields) ToJSON() (string, error) {
	if cf == nil {
		return "[]", nil
	}
	b, err := json.Marshal(cf)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// FromJSON populates a slice of CustomField from its JSON string representation.
func (cf *CustomFields) FromJSON(jsonStr string) error {
	if jsonStr == "" {
		*cf = []CustomField{}
		return nil
	}
	return json.Unmarshal([]byte(jsonStr), cf)
}

// --- STRUCTS FOR SEARCH ENDPOINT ---

// SearchFilter defines the structure for a single filter or a nested group.
type SearchFilter struct {
	// For logical grouping ("and", "or")
	Operator   string          `json:"operator,omitempty"`
	Conditions []*SearchFilter `json:"conditions,omitempty"`

	// For single conditions ("=", ">", "<", etc.)
	Field string      `json:"field,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

// SearchSort defines the sorting criteria.
type SearchSort struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // "asc", "desc"
}

// SearchPagination defines the limit and offset for the query.
type SearchPagination struct {
	Offset int  `json:"offset"`
	Limit  *int `json:"limit"` // Pointer to check if it was provided
}

// SearchRequest is the top-level request body for the search endpoint.
type SearchRequest struct {
	Filter     *SearchFilter     `json:"filter"` // Pointer so it can be omitted
	Sort       *SearchSort       `json:"sort"`   // Pointer so it can be omitted
	Pagination *SearchPagination `json:"pagination"`
}
