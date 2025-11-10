// filepath: internal/services/entry_service.go
package services

import (
	"fmt"
	"mediahub/internal/config"
	"mediahub/internal/logging"
	"mediahub/internal/models"
	"mediahub/internal/repository"
	"mime/multipart"
	"net/http"
	"os"
)

var _ EntryService = (*entryService)(nil)

// entryService is the main struct for handling business logic related to entries.
// It acts as a facade, delegating complex logic to specialized handler and helper files.
type entryService struct {
	Repo    *repository.Repository
	Storage *StorageService
	Cfg     *config.Config
}

// NewEntryService creates a new EntryService.
func NewEntryService(repo *repository.Repository, storage *StorageService, cfg *config.Config) *entryService {
	return &entryService{
		Repo:    repo,
		Storage: storage,
		Cfg:     cfg,
	}
}

// === Public Methods (Implementing the EntryService interface) ===

// CreateEntry is the main "router" for the hybrid upload model.
// It detects if a file is small (in-memory) or large (on-disk) and
// delegates to the appropriate handler.
func (s *entryService) CreateEntry(dbName string, metadataStr string, file multipart.File, header *multipart.FileHeader) (interface{}, int, error) {

	// 1. Parse and validate metadata (Common to both paths)
	db, dbConfig, entryMetadata, err := s.parseUploadMetadata(dbName, metadataStr)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	// 2. Validate MIME type (Common to both paths)
	originalMime, err := s.validateMimeType(db, header)
	if err != nil {
		return nil, http.StatusUnsupportedMediaType, err
	}

	// 3. --- The "Hybrid Model" routing logic ---
	// Check if the underlying file is an *os.File. This indicates it's
	// a large file that the http server has spooled to disk.
	if f, ok := file.(*os.File); ok {
		// Path A: Large File, Async
		logging.Log.Debugf("CreateEntry: Detected large file (spooled to disk at %s). Routing to async handler.", f.Name())
		return s.handleLargeFileAsync(f, header, db, dbConfig, entryMetadata, originalMime)
	}

	// Path B: Small File, Sync
	logging.Log.Debug("CreateEntry: Detected small file (in memory). Routing to sync handler.")
	return s.handleSmallFileSync(file, header, db, dbConfig, entryMetadata, originalMime)
}

// DeleteEntry handles the business logic for deleting an entry and its associated files.
func (s *entryService) DeleteEntry(dbName string, id int64) error {
	// 1. Get entry metadata (needed for file paths)
	db, err := s.Repo.GetDatabase(dbName)
	if err != nil {
		return ErrNotFound
	}

	entry, err := s.Repo.GetEntry(dbName, id, db.CustomFields)
	if err != nil {
		return ErrNotFound
	}
	timestamp := entry["timestamp"].(int64)

	// 2. Delete the entry record from the DB (this now updates stats)
	if err := s.Repo.DeleteEntry(dbName, id); err != nil {
		logging.Log.Errorf("EntryService: Failed to delete entry record %d from %s: %v", id, dbName, err)
		return fmt.Errorf("failed to delete entry record: %w", err)
	}

	// 3. Delete the main file
	if err := s.Storage.DeleteEntryFile(dbName, timestamp, id); err != nil {
		logging.Log.Warnf("EntryService: Failed to delete entry file: %v", err)
	}

	// 4. Delete the preview file
	if err := s.Storage.DeletePreviewFile(dbName, timestamp, id); err != nil {
		logging.Log.Warnf("EntryService: Failed to delete preview file: %v", err)
	}

	logging.Log.Infof("EntryService: Entry deleted: %d from database %s", id, dbName)
	return nil
}

// UpdateEntry handles the business logic for updating an entry's metadata.
func (s *entryService) UpdateEntry(dbName string, id int64, updates models.Entry) (models.Entry, error) {
	db, err := s.Repo.GetDatabase(dbName)
	if err != nil {
		return nil, ErrNotFound
	}

	// Sanitize the update map to remove protected fields
	delete(updates, "id")
	delete(updates, "filesize")
	delete(updates, "mime_type")
	delete(updates, "status") // Protect status from manual updates

	if err := s.Repo.UpdateEntry(dbName, id, updates, db.CustomFields); err != nil {
		logging.Log.Errorf("EntryService: Failed to update entry %d: %v", id, err)
		return nil, fmt.Errorf("failed to update entry")
	}

	// Return the full, updated entry
	return s.Repo.GetEntry(dbName, id, db.CustomFields)
}

// GetEntryFile handles the logic for retrieving the path and info for a raw file download.
func (s *entryService) GetEntryFile(dbName string, id int64) (string, string, string, error) {
	db, err := s.Repo.GetDatabase(dbName)
	if err != nil {
		return "", "", "", ErrNotFound
	}

	entry, err := s.Repo.GetEntry(dbName, id, db.CustomFields)
	if err != nil {
		return "", "", "", ErrNotFound
	}

	timestamp := entry["timestamp"].(int64)
	mimeType := entry["mime_type"].(string)

	filename := "" // Default to empty string
	if fn, ok := entry["filename"].(string); ok {
		filename = fn
	}

	entryPath, err := s.Storage.GetEntryPath(dbName, timestamp, id)
	if err != nil {
		return "", "", "", fmt.Errorf("could not get entry path: %w", err)
	}

	return entryPath, mimeType, filename, nil
}

// GetEntryPreview handles the logic for retrieving the path for a preview file download.
func (s *entryService) GetEntryPreview(dbName string, id int64) (string, error) {
	db, err := s.Repo.GetDatabase(dbName)
	if err != nil {
		return "", ErrNotFound
	}

	entry, err := s.Repo.GetEntry(dbName, id, db.CustomFields)
	if err != nil {
		return "", ErrNotFound
	}

	timestamp := entry["timestamp"].(int64)

	previewPath, err := s.Storage.GetPreviewPath(dbName, timestamp, id)
	if err != nil {
		return "", fmt.Errorf("could not get preview path: %w", err)
	}

	return previewPath, nil
}

// === Pass-through Methods to Repository ===

// GetEntry retrieves a single entry's metadata.
func (s *entryService) GetEntry(dbName string, id int64, customFields []models.CustomField) (models.Entry, error) {
	return s.Repo.GetEntry(dbName, id, customFields)
}

// GetEntries retrieves a list of entries with basic time/pagination filters.
func (s *entryService) GetEntries(dbName string, limit, offset int, order string, tstart, tend int64, customFields []models.CustomField) ([]models.Entry, error) {
	return s.Repo.GetEntries(dbName, limit, offset, order, tstart, tend, customFields)
}

// SearchEntries retrieves a list of entries with complex filters.
func (s *entryService) SearchEntries(dbName string, req *models.SearchRequest, customFields []models.CustomField) ([]models.Entry, error) {
	return s.Repo.SearchEntries(dbName, req, customFields)
}
