package initconfig

import (
	"github.com/BurntSushi/toml"
)

// ParseInitConfig reads the TOML initialization file from the given path.
func ParseInitConfig(path string) (InitConfig, error) {
	var config InitConfig

	_, err := toml.DecodeFile(path, &config)
	if err != nil {
		return InitConfig{}, err
	}

	// Assign safe defaults for any omitted fields
	config.PostProcess()

	return config, nil
}

// PostProcess iterates over the parsed config and sets missing default values.
// This ensures that downstream parsing (like shared.ParseSize) doesn't fail on empty strings.
func (config *InitConfig) PostProcess() {
	for i := range config.Databases {
		// 1. Set default Housekeeping values if they are omitted in the TOML
		if config.Databases[i].Housekeeping.Interval == "" {
			config.Databases[i].Housekeeping.Interval = "24h"
		}
		if config.Databases[i].Housekeeping.DiskSpace == "" {
			config.Databases[i].Housekeeping.DiskSpace = "0" // "0" disables the check
		}
		if config.Databases[i].Housekeeping.MaxAge == "" {
			config.Databases[i].Housekeeping.MaxAge = "0" // "0" disables age-based cleanup
		}

		// 2. Set a fallback content type if none is provided
		if config.Databases[i].ContentType == "" {
			config.Databases[i].ContentType = "file"
		}
	}
}
