package repository

import (
	"encoding/json"
	"mediahub/internal/shared"
	"time"
)

type Database struct {
	Name         string              `json:"name"`
	ContentType  string              `json:"content_type"`
	Config       json.RawMessage     `json:"config" swaggertype:"object"`
	Housekeeping shared.Housekeeping `json:"housekeeping"`
	CustomFields shared.CustomFields `json:"custom_fields"`
	Stats        *DatabaseStats      `json:"stats,omitempty"`
	LastHkRun    time.Time           `json:"last_hk_run"`
}

type DatabaseStats struct {
	EntryCount          int   `json:"entry_count"`
	TotalDiskSpaceBytes int64 `json:"total_disk_space_bytes"`
}

type Entry struct {
}

type SearchRequest struct {
}

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Roles        []shared.Role
}
