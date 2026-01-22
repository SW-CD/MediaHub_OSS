package config

import (
	"fmt"
	"mediahub/internal/shared"
	"os"

	"github.com/BurntSushi/toml"
)

// LoadConfig loads the configuration from a TOML file.
func LoadConfig(path string) (*Config, error) {
	// Parse the config
	var config Config
	_, err := toml.DecodeFile(path, &config)
	if err != nil {
		return nil, err
	}

	// Check if values that need to be converted are ok as well
	if _, err := config.GetServerConfig(); err != nil {
		return nil, err
	}
	if _, err := config.GetJWTConfig(); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig writes the current configuration back to a TOML file.
// Used to persist the auto-generated JWT secret.
func SaveConfig(path string, cfg *Config) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("trying to save the config: %w", shared.ErrorCreateFile)
	}
	defer f.Close()
	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("trying to save the config: %w", shared.ErrorEncodeFile)
	}
	return nil
}
