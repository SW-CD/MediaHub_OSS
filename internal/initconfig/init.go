// filepath: internal/initconfig/init.go
package initconfig

import (
	"bytes"
	"mediahub/internal/logging"
	"mediahub/internal/models"
	"mediahub/internal/repository" // Still need this for UserCreateArgs
	"mediahub/internal/services"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// Run executes the one-time initialization from the config file.
func Run(
	userSvc services.UserService,
	dbSvc services.DatabaseService,
	configPath string,
) {
	logging.Log.Infof("Initialization config file found at: %s. Processing...", configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		logging.Log.Errorf("Failed to read init config file '%s': %v", configPath, err)
		return
	}

	var config InitConfig
	if _, err := toml.Decode(string(data), &config); err != nil {
		logging.Log.Errorf("Failed to parse TOML init config file '%s': %v", configPath, err)
		return
	}

	logging.Log.Infof("Found %d user(s) and %d database(s) in init config.", len(config.Users), len(config.Databases))

	processUsers(userSvc, config.Users)
	processDatabases(dbSvc, config.Databases)

	// After processing, try to clear passwords
	clearPasswords(&config, configPath)
}

// processUsers iterates over the users in the config and creates them if they don't exist.
func processUsers(userSvc services.UserService, users []InitUser) {
	for _, u := range users {
		if u.Name == "" || u.Password == "" {
			logging.Log.Warnf("Skipping user with empty name or password.")
			continue
		}

		_, err := userSvc.GetUserByUsername(u.Name)
		if err == nil {
			// No error means user was found
			logging.Log.Infof("Skipping user: '%s' already exists.", u.Name)
			continue
		}

		// Check if the error was *something other* than "not found"
		if !strings.Contains(err.Error(), "not found") {
			logging.Log.Errorf("Failed to check if user '%s' exists: %v", u.Name, err)
			continue
		}

		// User does not exist, create them
		logging.Log.Infof("Creating user: '%s'...", u.Name)

		// Map roles
		dbUser := &repository.UserCreateArgs{ // <-- USE REPOSITORY STRUCT
			Username: u.Name,
			Password: u.Password,
		}
		for _, role := range u.Roles {
			switch role {
			case "CanView":
				dbUser.CanView = true
			case "CanCreate":
				dbUser.CanCreate = true
			case "CanEdit":
				dbUser.CanEdit = true
			case "CanDelete":
				dbUser.CanDelete = true
			case "IsAdmin":
				dbUser.IsAdmin = true
			}
		}

		if _, err := userSvc.CreateUser(*dbUser); err != nil {
			logging.Log.Errorf("Failed to create user '%s': %v", u.Name, err)
		} else {
			logging.Log.Infof("Successfully created user: '%s'", u.Name)
		}
	}
}

// processDatabases iterates over the databases in the config and creates them if they don't exist.
func processDatabases(dbSvc services.DatabaseService, databases []InitDatabase) { // <-- Use InitDatabase
	for _, d := range databases {
		if d.Name == "" {
			logging.Log.Warnf("Skipping database with empty name.")
			continue
		}

		_, err := dbSvc.GetDatabase(d.Name)
		if err == nil {
			// No error means we found the database
			logging.Log.Infof("Skipping database: '%s' already exists.", d.Name)
			continue
		}

		// Database does not exist, create it
		logging.Log.Infof("Creating database: '%s'...", d.Name)

		// Create the payload for the DatabaseService
		payload := models.DatabaseCreatePayload{
			Name:         d.Name,
			ContentType:  d.ContentType,
			Config:       d.Config,
			Housekeeping: &d.Housekeeping, // The service handles nil
			CustomFields: d.CustomFields,
		}

		if _, err := dbSvc.CreateDatabase(payload); err != nil {
			// The service already did the validation, so we just log the error
			logging.Log.Errorf("Failed to create database '%s': %v", d.Name, err)
		} else {
			logging.Log.Infof("Successfully created database: '%s'", d.Name)
		}
	}
}

// clearPasswords attempts to overwrite the config file with passwords removed.
func clearPasswords(config *InitConfig, configPath string) {
	logging.Log.Info("Attempting to clear passwords from init config file...")

	// Create a buffer to write the new TOML data
	buf := new(bytes.Buffer)

	// Set all user passwords to an empty string
	for i := range config.Users {
		config.Users[i].Password = ""
	}

	// Encode the modified config back to TOML
	if err := toml.NewEncoder(buf).Encode(config); err != nil {
		logging.Log.Warnf("Could not re-encode config to clear passwords: %v", err)
		logging.Log.Warnf("SECURITY: Please manually remove passwords from '%s'", configPath)
		return
	}

	// Try to write the new config back to the original file
	if err := os.WriteFile(configPath, buf.Bytes(), 0644); err != nil {
		logging.Log.Warnf("Failed to write back to config file to clear passwords: %v", err)
		logging.Log.Warnf("SECURITY: Please manually remove passwords from '%s'", configPath)
		return
	}

	logging.Log.Info("Successfully cleared passwords from init config file.")
}
