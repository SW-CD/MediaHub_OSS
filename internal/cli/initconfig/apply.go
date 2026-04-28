package initconfig

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"golang.org/x/crypto/bcrypt"

	"mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared/customerrors"
)

// Apply executes the initialization configuration against the repository.
func Apply(ctx context.Context, config *InitConfig, repo repository.Repository, logger *slog.Logger, filePath string) error {
	// 0. Pre-fetch existing databases to build a Name -> ID resolution map.
	// This is required because the config uses names, but the DB uses ULIDs.
	existingDBs, err := repo.GetDatabases(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch existing databases for init config resolution: %w", err)
	}

	dbNameToID := make(map[string]string)
	for _, db := range existingDBs {
		dbNameToID[db.Name] = db.ID
	}

	// 1. Initialize Databases
	for _, dbInit := range config.Databases {
		if _, exists := dbNameToID[dbInit.Name]; !exists {
			hk, err := dbInit.GetHousekeeping()
			if err != nil {
				logger.Error("Failed to parse housekeeping config", "database", dbInit.Name, "error", err)
				continue
			}

			db := repository.Database{
				Name:        dbInit.Name,
				ContentType: dbInit.ContentType,
				Config: repository.DatabaseConfig{
					CreatePreview:  dbInit.Config.CreatePreview,
					AutoConversion: dbInit.Config.AutoConversion,
				},
				Housekeeping: hk,
				CustomFields: dbInit.CustomFields,
			}

			createdDB, err := repo.CreateDatabase(ctx, db)
			if err != nil {
				logger.Error("Failed to create init database", "database", dbInit.Name, "error", err)
			} else {
				logger.Info("Created database from init config", "database", dbInit.Name)
				// Add the newly generated ULID to our resolution map so user permissions can use it!
				dbNameToID[createdDB.Name] = createdDB.ID
			}
		} else {
			logger.Debug("Database from init config already exists, skipping", "database", dbInit.Name)
		}
	}

	// 2. Initialize Users
	passwordsRedacted := false
	for i, userInit := range config.Users {
		_, err := repo.GetUserByUsername(ctx, userInit.Name)
		if errors.Is(err, customerrors.ErrNotFound) {

			// Hash the password securely
			hash, err := bcrypt.GenerateFromPassword([]byte(userInit.Password), bcrypt.DefaultCost)
			if err != nil {
				logger.Error("Failed to hash password for user", "user", userInit.Name, "error", err)
				continue
			}

			user := repository.User{
				Username:     userInit.Name,
				IsAdmin:      userInit.IsAdmin,
				PasswordHash: string(hash),
			}

			createdUser, err := repo.CreateUser(ctx, user)
			if err != nil {
				logger.Error("Failed to create init user", "user", userInit.Name, "error", err)
				continue
			}
			logger.Info("Created user from init config", "user", userInit.Name)

			// Assign Permissions
			for _, permInit := range userInit.Permissions {
				// Resolve the database name to its ULID
				dbID, ok := dbNameToID[permInit.DatabaseName]
				if !ok {
					logger.Error("Cannot set permission, database not found", "user", userInit.Name, "database_name", permInit.DatabaseName)
					continue
				}

				var roles []string
				if permInit.CanView {
					roles = append(roles, "CanView")
				}
				if permInit.CanCreate {
					roles = append(roles, "CanCreate")
				}
				if permInit.CanEdit {
					roles = append(roles, "CanEdit")
				}
				if permInit.CanDelete {
					roles = append(roles, "CanDelete")
				}

				err := repo.SetUserPermissions(ctx, repository.UserPermissions{
					UserID:     createdUser.ID,
					DatabaseID: dbID, // Use the resolved ULID here!
					Roles:      strings.Join(roles, ","),
				})

				if err != nil {
					logger.Error("Failed to set permissions for user", "user", userInit.Name, "database", permInit.DatabaseName, "error", err)
				}
			}

			// Clear the plaintext password in memory
			config.Users[i].Password = ""
			passwordsRedacted = true

		} else if err != nil {
			logger.Error("Error checking user existence", "user", userInit.Name, "error", err)
		} else {
			logger.Debug("User from init config already exists, skipping", "user", userInit.Name)
		}
	}

	// 3. Redact passwords in the TOML file if any new users were created
	if passwordsRedacted {
		if err := redactPasswordsInFile(filePath, config); err != nil {
			logger.Warn("Failed to overwrite init config file to remove passwords", "path", filePath, "error", err)
		} else {
			logger.Info("Successfully removed plaintext passwords from init config file", "path", filePath)
		}
	}

	return nil
}

// redactPasswordsInFile encodes the sanitized configuration back to the file system.
func redactPasswordsInFile(filePath string, config *InitConfig) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	return encoder.Encode(config)
}
