// filepath: internal/services/storage_service.go
package services

import (
	"fmt"
	"io"
	"mediahub/internal/config"
	"mediahub/internal/logging"
	"mediahub/internal/storage"
	"os"
	"path/filepath"
	"strings"
)

// StorageService provides an interface for interacting with the file system.
// It wraps the 'internal/storage' package to be injectable.
type StorageService struct {
	StorageRoot string
}

// NewStorageService creates a new StorageService.
func NewStorageService(cfg *config.Config) *StorageService {
	return &StorageService{
		StorageRoot: cfg.Database.StorageRoot,
	}
}

// GetEntryPath generates the full, absolute path for an entry file.
func (s *StorageService) GetEntryPath(dbName string, timestamp, entryID int64) (string, error) {
	return storage.GetEntryPath(s.StorageRoot, dbName, timestamp, entryID)
}

// GetPreviewPath generates the full, absolute path for an entry preview file.
func (s *StorageService) GetPreviewPath(dbName string, timestamp, entryID int64) (string, error) {
	return storage.GetPreviewPath(s.StorageRoot, dbName, timestamp, entryID)
}

// SaveFile saves file data from a reader to a specified path.
func (s *StorageService) SaveFile(fileData io.Reader, path string) (int64, error) {
	return storage.SaveFile(fileData, path)
}

// CreateDatabaseFolders creates the main and preview folders for a new database.
func (s *StorageService) CreateDatabaseFolders(dbName string) error {
	// 1. Validate path
	dbPath, err := s.validatePath(dbName)
	if err != nil {
		return err
	}

	// 2. Create main folder
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return err
	}

	// 3. Create preview folder
	previewPath, err := s.validatePath("previews", dbName)
	if err != nil {
		return err // Should be safe, but good to check
	}
	if err := os.MkdirAll(previewPath, 0755); err != nil {
		// Don't fail the operation, just log a warning
		logging.Log.Warnf("Failed to create database previews folder '%s': %v", previewPath, err)
	}
	return nil
}

// DeleteDatabaseFolders deletes the main and preview folders for a database.
func (s *StorageService) DeleteDatabaseFolders(dbName string) error {
	// 1. Delete main folder
	dbPath, err := s.validatePath(dbName)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dbPath); err != nil {
		logging.Log.Warnf("Failed to delete database folder '%s': %v", dbPath, err)
		// Don't return error, just log
	}

	// 2. Delete preview folder
	previewPath, err := s.validatePath("previews", dbName)
	if err != nil {
		logging.Log.Warnf("Failed to validate preview path for deletion '%s': %v", previewPath, err)
		return nil
	}
	if err := os.RemoveAll(previewPath); err != nil {
		logging.Log.Warnf("Failed to delete database previews folder '%s': %v", previewPath, err)
	}
	return nil
}

// DeleteEntryFile deletes a single entry file from storage.
func (s *StorageService) DeleteEntryFile(dbName string, timestamp, entryID int64) error {
	path, err := s.GetEntryPath(dbName, timestamp, entryID)
	if err != nil {
		return fmt.Errorf("could not get entry path for deletion: %w", err)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete entry file %s: %w", path, err)
	}
	return nil
}

// DeletePreviewFile deletes a single preview file from storage.
func (s *StorageService) DeletePreviewFile(dbName string, timestamp, entryID int64) error {
	path, err := s.GetPreviewPath(dbName, timestamp, entryID)
	if err != nil {
		return fmt.Errorf("could not get preview path for deletion: %w", err)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete preview file %s: %w", path, err)
	}
	return nil
}

// validatePath cleans a path and ensures it's within the storage root.
func (s *StorageService) validatePath(subdirs ...string) (string, error) {
	fullPath := filepath.Join(s.StorageRoot, filepath.Join(subdirs...))
	cleanedPath := filepath.Clean(fullPath)
	cleanedRoot := filepath.Clean(s.StorageRoot)

	if !strings.HasPrefix(cleanedPath, cleanedRoot) || cleanedPath == cleanedRoot {
		logging.Log.Warnf("Path traversal attempt blocked for: %s", fullPath)
		return "", fmt.Errorf("invalid path")
	}
	return cleanedPath, nil
}
