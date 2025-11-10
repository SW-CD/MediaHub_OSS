// filepath: internal/initconfig/models.go
package initconfig

import "mediahub/internal/models"

// InitConfig is the root struct for parsing the TOML initialization file.
type InitConfig struct {
	Users     []InitUser     `toml:"user"`
	Databases []InitDatabase `toml:"database"` // <-- Use InitDatabase
}

// InitUser represents a user entry in the TOML config file.
type InitUser struct {
	Name     string   `toml:"name"`
	Roles    []string `toml:"roles"`
	Password string   `toml:"password"`
}

// InitDatabase represents a database entry in the TOML config file.
// It mirrors models.Database but uses a map[string]interface{} for config
// to match the JSON/API payload structure.
type InitDatabase struct {
	Name         string                 `toml:"name"`
	ContentType  string                 `toml:"content_type"`
	Config       map[string]interface{} `toml:"config"`
	Housekeeping models.Housekeeping    `toml:"housekeeping"`
	CustomFields models.CustomFields    `toml:"custom_fields"`
}
