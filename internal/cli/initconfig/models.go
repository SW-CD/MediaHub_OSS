// filepath: internal/initconfig/models.go
package initconfig

import "mediahub/internal/shared"

// InitConfig is the root struct for parsing the TOML initialization file.
type InitConfig struct {
	Users     []InitUser     `toml:"user"`
	Databases []InitDatabase `toml:"database"`
}

// InitUser represents a user entry in the TOML config file.
// "DefaultRole" defines the role that the user has for databases not specified in "Roles"
type InitUser struct {
	Name        string        `toml:"name"`
	Password    string        `toml:"password"`
	DefaultRole string        `toml:"default_role"`
	Roles       []shared.Role `toml:"roles"`
}

// initDatabaseInternal represents a database entry in the TOML config file.
// it is not public, as its fields need conversion first
// Config contains, e.g., autoconversion settings (depends on ContentType)
type InitDatabase struct {
	ID                   int                    `toml:"id"`
	Name                 string                 `toml:"name"`
	ContentType          string                 `toml:"content_type"`
	Config               map[string]interface{} `toml:"config"`
	HousekeepingInternal housekeepingInternal   `toml:"housekeeping"`
	CustomFields         shared.CustomFields    `toml:"custom_fields"`
}

type housekeepingInternal struct {
	Interval  string `toml:"interval"`
	DiskSpace string `toml:"disk_space"`
	MaxAge    string `toml:"max_age"`
}

// getter for the Housekeeping struct
func (initdb *InitDatabase) GetHousekeeping() (shared.Housekeeping, error) {
	interval, err := shared.ParseDuration(initdb.HousekeepingInternal.Interval)
	if err != nil {
		return shared.Housekeeping{}, err
	}
	diskSpace, err := shared.ParseSize(initdb.HousekeepingInternal.DiskSpace)
	if err != nil {
		return shared.Housekeeping{}, err
	}
	maxAge, err := shared.ParseDuration(initdb.HousekeepingInternal.MaxAge)
	if err != nil {
		return shared.Housekeeping{}, err
	}
	return shared.Housekeeping{
		Interval:  interval,
		DiskSpace: diskSpace,
		MaxAge:    maxAge,
	}, nil
}
