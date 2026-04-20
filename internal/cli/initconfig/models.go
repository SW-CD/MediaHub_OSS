package initconfig

import (
	"mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared"
)

// InitConfig is the root struct for parsing the TOML initialization file.
type InitConfig struct {
	Users     []InitUser     `toml:"user"`
	Databases []InitDatabase `toml:"database"`
}

// InitUser represents a user entry in the TOML config file.
type InitUser struct {
	Name        string               `toml:"name"`
	IsAdmin     bool                 `toml:"is_admin"`
	Password    string               `toml:"password"`
	Permissions []InitUserPermission `toml:"permissions"`
}

// InitUserPermission defines the explicit database access rights for a user.
type InitUserPermission struct {
	DatabaseName string `toml:"database_name"`
	CanView      bool   `toml:"can_view"`
	CanCreate    bool   `toml:"can_create"`
	CanEdit      bool   `toml:"can_edit"`
	CanDelete    bool   `toml:"can_delete"`
}

// InitDatabase represents a database entry to be created.
type InitDatabase struct {
	Name         string                   `toml:"name"`
	ContentType  string                   `toml:"content_type"`
	Config       InitDatabaseConfig       `toml:"config"`
	Housekeeping InitHousekeeping         `toml:"housekeeping"`
	CustomFields []repository.CustomField `toml:"custom_fields"`
}

// InitDatabaseConfig maps to the repository.DatabaseConfig.
type InitDatabaseConfig struct {
	CreatePreview  bool   `toml:"create_previews"` // Maps to "create_previews" or "create_preview" in TOML
	AutoConversion string `toml:"auto_conversion"`
}

// InitHousekeeping uses strings for values that need parsing (e.g., "100G", "30d").
type InitHousekeeping struct {
	Interval  string `toml:"interval"`
	DiskSpace string `toml:"disk_space"`
	MaxAge    string `toml:"max_age"`
}

// GetHousekeeping converts the string-based TOML values into the required formats.
// It returns a repository.DatabaseHK which uses nanoseconds for durations and bytes for size.
func (initdb *InitDatabase) GetHousekeeping() (repository.DatabaseHK, error) {
	// Parse the interval duration
	interval, err := shared.ParseDuration(initdb.Housekeeping.Interval)
	if err != nil {
		return repository.DatabaseHK{}, err
	}

	// Parse the disk space string into bytes
	diskSpace, err := shared.ParseSize(initdb.Housekeeping.DiskSpace)
	if err != nil {
		return repository.DatabaseHK{}, err
	}

	// Parse the max age duration
	maxAge, err := shared.ParseDuration(initdb.Housekeeping.MaxAge)
	if err != nil {
		return repository.DatabaseHK{}, err
	}

	return repository.DatabaseHK{
		Interval:  interval,
		DiskSpace: diskSpace,
		MaxAge:    maxAge,
	}, nil
}
