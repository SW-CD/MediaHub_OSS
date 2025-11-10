// filepath: internal/services/database_service.go
package services

import (
	"encoding/json"
	"fmt"
	"mediahub/internal/logging"
	"mediahub/internal/media"
	"mediahub/internal/models"
	"mediahub/internal/repository" // Depends on the repository
)

// --- Compile-time check to ensure interface is implemented ---
var _ DatabaseService = (*databaseService)(nil)

// --- Struct renamed to lowercase ---
// databaseService handles business logic for database management.
type databaseService struct {
	Repo    *repository.Repository
	Storage *StorageService // Depends on the new StorageService
}

// NewDatabaseService creates a new DatabaseService.
// --- Return type is the concrete struct, but it satisfies the interface ---
func NewDatabaseService(repo *repository.Repository, storage *StorageService) *databaseService {
	return &databaseService{
		Repo:    repo,
		Storage: storage,
	}
}

// === Pass-through Repository Methods ===

func (s *databaseService) GetDatabase(name string) (*models.Database, error) {
	return s.Repo.GetDatabase(name)
}

func (s *databaseService) GetDatabases() ([]models.Database, error) {
	return s.Repo.GetDatabases()
}

// --- MOVED: GetEntries, SearchEntries, and GetEntry have been moved to EntryService ---

// === Business Logic Methods ===

// CreateDatabase orchestrates creating a database record and its folders.
// --- Signature changed to accept payload ---
func (s *databaseService) CreateDatabase(payload models.DatabaseCreatePayload) (*models.Database, error) {
	// 1. Validate 'config' and check FFmpeg dependencies
	configJSON, err := s.validateAndMarshalConfig(payload.Config) // <-- FIX: Changed var name
	if err != nil {
		return nil, err
	}

	// 2. Set default housekeeping rules if not provided
	hk := models.Housekeeping{}
	if payload.Housekeeping != nil {
		hk = *payload.Housekeeping
	}
	if hk.Interval == "" {
		hk.Interval = "1h"
	}
	if hk.DiskSpace == "" {
		hk.DiskSpace = "100G"
	}
	if hk.MaxAge == "" {
		hk.MaxAge = "365d"
	}

	// --- Build the models.Database struct here ---
	dbModel := &models.Database{
		Name:         payload.Name,
		ContentType:  payload.ContentType,
		Config:       configJSON, // <-- FIX: Use the marshaled json.RawMessage
		Housekeeping: hk,
		CustomFields: payload.CustomFields,
	}

	// 3. Create the database record
	createdDB, err := s.Repo.CreateDatabase(dbModel)
	if err != nil {
		logging.Log.Errorf("DatabaseService: Failed to create database record: %v", err)
		return nil, err // Pass up repository errors (e.g., unique constraint)
	}

	// 4. Create the storage folders
	if err := s.Storage.CreateDatabaseFolders(createdDB.Name); err != nil {
		logging.Log.Errorf("DatabaseService: Failed to create storage folders for '%s': %v", createdDB.Name, err)
		// Attempt to roll back the database creation
		if delErr := s.Repo.DeleteDatabase(createdDB.Name); delErr != nil {
			logging.Log.Errorf("CRITICAL: Failed to rollback database creation for '%s' after folder error: %v", createdDB.Name, delErr)
		}
		return nil, fmt.Errorf("failed to create database storage folders")
	}

	logging.Log.Infof("DatabaseService: Database created successfully: %s", createdDB.Name)
	return createdDB, nil
}

// UpdateDatabase updates a database's config and housekeeping rules.
// --- Signature changed to accept payload ---
func (s *databaseService) UpdateDatabase(name string, updates models.DatabaseUpdatePayload) (*models.Database, error) {
	// 1. Get existing DB data
	existingDB, err := s.Repo.GetDatabase(name)
	if err != nil {
		return nil, fmt.Errorf("database not found")
	}

	// 2. Apply 'config' updates if provided
	if updates.Config != nil {
		// Marshal and validate the new config
		configJSON, err := s.validateAndMarshalConfig(updates.Config) // <-- FIX: Changed var name
		if err != nil {
			return nil, err
		}
		existingDB.Config = configJSON // <-- FIX: Assign json.RawMessage
	}

	// 3. Apply 'housekeeping' updates if provided
	if updates.Housekeeping != nil {
		// Check each field individually, as user might send a partial object
		if updates.Housekeeping.Interval != "" {
			existingDB.Housekeeping.Interval = updates.Housekeeping.Interval
		}
		if updates.Housekeeping.DiskSpace != "" {
			existingDB.Housekeeping.DiskSpace = updates.Housekeeping.DiskSpace
		}
		if updates.Housekeeping.MaxAge != "" {
			existingDB.Housekeeping.MaxAge = updates.Housekeeping.MaxAge
		}
	}

	// 4. Persist the updated model
	if err := s.Repo.UpdateDatabase(existingDB); err != nil {
		logging.Log.Errorf("DatabaseService: Failed to update database '%s': %v", name, err)
		return nil, fmt.Errorf("failed to update database")
	}

	return existingDB, nil
}

// DeleteDatabase orchestrates deleting a database record and its folders.
func (s *databaseService) DeleteDatabase(name string) error {
	// 1. Attempt to delete the database record first
	if err := s.Repo.DeleteDatabase(name); err != nil {
		logging.Log.Errorf("DatabaseService: Failed to delete database record '%s': %v", name, err)
		return err // Pass up errors (e.g., not found)
	}

	// 2. Delete the associated storage folders
	if err := s.Storage.DeleteDatabaseFolders(name); err != nil {
		// Log the error but don't fail the request, as the DB record is already gone
		logging.Log.Warnf("DatabaseService: Failed to delete database folders for '%s': %v", name, err)
	}

	logging.Log.Infof("DatabaseService: Database deleted successfully: %s", name)
	return nil
}

// validateAndMarshalConfig checks FFmpeg dependencies and marshals the config map to a string.
func (s *databaseService) validateAndMarshalConfig(configMap map[string]interface{}) (json.RawMessage, error) { // <-- FIX: Return json.RawMessage
	if configMap == nil {
		return []byte("{}"), nil // <-- FIX: Return []byte
	}

	configBytes, err := json.Marshal(configMap)
	if err != nil {
		return nil, fmt.Errorf("invalid 'config' object: %w", err) // <-- FIX: Return nil
	}

	var dbConfig models.DatabaseConfig
	if err := json.Unmarshal(configBytes, &dbConfig); err != nil {
		return nil, fmt.Errorf("failed to parse 'config' object: %w", err) // <-- FIX: Return nil
	}

	// Perform the FFmpeg dependency check
	if dbConfig.AutoConversion != "" && dbConfig.AutoConversion != "none" {
		if !media.IsFFmpegAvailable() {
			return nil, fmt.Errorf("%w: cannot enable 'auto_conversion': FFmpeg is not available on the server", ErrDependencies) // <-- FIX: Return nil
		}
	}

	return configBytes, nil
}
