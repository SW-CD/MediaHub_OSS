package config

import (
	"fmt"
	"mediahub_oss/internal/shared/customerrors"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/viper"
)

// LoadConfig leverages Viper to merge TOML, Env Variables, and CLI flags natively.
func LoadConfig(path string, isOSS bool) (*Config, error) {
	// 1. Tell Viper where to find the TOML file
	viper.SetConfigFile(path)

	// 2. Configure Environment Variable automation
	// This replaces your manual bindEnvVars logic!
	viper.SetEnvPrefix("MEDIAHUB")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	// 3. Read the TOML file
	if err := viper.ReadInConfig(); err != nil {
		// If the file simply isn't there, that's fine. But if it's malformed, we should panic/return.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error parsing config file: %w", err)
		}
	}

	// 4. Unmarshal the merged state (TOML + Env + Flags) directly into your struct
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Check that commercial settings are not enabled
	if isOSS {
		if err := config.validateOSS(); err != nil {
			return nil, err
		}
	}

	// Validate settings
	if err := config.ValidateConfig(); err != nil {
		return nil, err
	}

	return &config, nil
}

// ValidateConfig ensures the application doesn't start in an inaccessible state.
func (cfg *Config) ValidateConfig() error {
	// If login page is disabled, but OIDC is also disabled, the user is locked out.
	if cfg.Auth.OIDC.DisableLoginPage && !cfg.Auth.OIDC.Enabled {
		return fmt.Errorf("invalid configuration: login page is disabled but OIDC is not enabled. You must enable at least one authentication method")
	}
	return nil
}

// SaveConfig writes the current configuration back to a TOML file.
// Used to persist the auto-generated JWT secret.
func SaveConfig(path string, cfg *Config) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("trying to save the config: %w", customerrors.ErrorCreateFile)
	}
	defer f.Close()
	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("trying to save the config: %w", customerrors.ErrorEncodeFile)
	}
	return nil
}
