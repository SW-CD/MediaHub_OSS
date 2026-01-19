// filepath: internal/initconfig/models.go
package initconfig

import "mediahub/internal/shared"

// InitConfig is the root struct for parsing the TOML initialization file.
type InitConfig struct {
	Users     []InitUser     `toml:"user"`
	Databases []InitDatabase `toml:"database"` // <-- Use InitDatabase
}

// InitUser represents a user entry in the TOML config file.
// "DefaultRole" defines the role that the user has for databases not specified in "Roles"
type InitUser struct {
	Name        string        `toml:"name"`
	Password    string        `toml:"password"`
	DefaultRole string        `toml:"default_role"`
	Roles       []shared.Role `toml:"roles"`
}

// InitDatabase represents a database entry in the TOML config file.
type InitDatabase struct {
	ID           int                    `toml:"id"`
	Name         string                 `toml:"name"`
	ContentType  string                 `toml:"content_type"`
	Config       map[string]interface{} `toml:"config"`
	Housekeeping shared.Housekeeping    `toml:"housekeeping"`
	CustomFields shared.CustomFields    `toml:"custom_fields"`
}
