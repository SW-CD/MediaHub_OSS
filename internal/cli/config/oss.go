package config

import (
	"fmt"
)

// validate the conig with regards to the open source version
// throw errors on commercial functionality, i.e., S3, PostgreSQL, OIDC
func (cfg *Config) validateOSS() error {
	if cfg.Auth.OIDC.Enabled {
		return fmt.Errorf("OIDC is only available in the commercial version of this software.")
	}
	if cfg.Storage.Type == "s3" {
		return fmt.Errorf("S3 storage is only available in the commercial version of this software.")
	}
	if cfg.Database.Driver == "postgres" {
		return fmt.Errorf("PostgreSQL support is only available in the commercial version of this software.")
	}
	return nil
}
