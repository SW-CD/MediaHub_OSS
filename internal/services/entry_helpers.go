// filepath: internal/services/entry_helpers.go
package services

import (
	"encoding/json"
	"fmt"
	"mediahub/internal/logging"
	"mediahub/internal/media"
	"mediahub/internal/models"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"
)

// === Private Helper Methods ===

var mimeToExtension = map[string]string{
	"image/jpeg": "jpg",
	"audio/flac": "flac",
	"audio/ogg":  "ogg",
	"audio/opus": "opus",
}

// getExtensionForMimeType returns the preferred file extension for a given MIME type (e.g., ".opus")
func getExtensionForMimeType(mimeType string) string {
	if ext, ok := mimeToExtension[mimeType]; ok {
		return "." + ext
	}
	// Fallback for unmapped types
	parts := strings.Split(mimeType, "/")
	if len(parts) == 2 {
		if parts[1] == "x-flac" { // Handle "audio/x-flac"
			return ".flac"
		}
		return "." + parts[1] // e.g., "image/png" -> ".png"
	}
	return "" // No extension
}

// replaceExtension replaces the extension of a filename with a new one.
// e.g., ("song.mp3", ".opus") -> "song.opus"
// e.g., ("file_no_ext", ".jpg") -> "file_no_ext.jpg"
func replaceExtension(filename string, newExt string) string {
	if filename == "" {
		return "" // Don't create a filename if one didn't exist
	}
	if newExt == "" {
		return filename // No new extension provided
	}
	ext := filepath.Ext(filename)
	base := filename[:len(filename)-len(ext)]
	return base + newExt
}

// parseUploadMetadata validates the request and parses the 'metadata' JSON part.
func (s *entryService) parseUploadMetadata(dbName, metadataStr string) (*models.Database, *models.DatabaseConfig, models.Entry, error) {
	if dbName == "" {
		return nil, nil, nil, fmt.Errorf("%w: missing database_name", ErrValidation)
	}

	db, err := s.Repo.GetDatabase(dbName)
	if err != nil {
		return nil, nil, nil, ErrNotFound
	}

	// Parse Database Config
	var dbConfig models.DatabaseConfig
	dbConfig.CreatePreview = true // Default
	if len(db.Config) > 0 {
		if err := json.Unmarshal(db.Config, &dbConfig); err != nil {
			logging.Log.Warnf("Could not parse db.Config for %s, using defaults. Error: %v", dbName, err)
		}
	}

	if !media.IsFFmpegAvailable() && dbConfig.AutoConversion != "none" {
		logging.Log.Warnf("Disabling audio auto-conversion for this request (FFmpeg not found).")
		dbConfig.AutoConversion = "none"
	}

	// Parse Metadata
	if metadataStr == "" {
		return nil, nil, nil, fmt.Errorf("%w: missing 'metadata' part in multipart form", ErrValidation)
	}

	var entry models.Entry
	if err := json.Unmarshal([]byte(metadataStr), &entry); err != nil {
		return nil, nil, nil, fmt.Errorf("%w: invalid JSON in 'metadata' part", ErrValidation)
	}

	// Set timestamp if not provided
	if ts, ok := entry["timestamp"]; ok {
		if _, ok := ts.(float64); ok {
			entry["timestamp"] = int64(ts.(float64))
		}
	} else {
		logging.Log.Debug("Timestamp not provided, adding current time")
		entry["timestamp"] = time.Now().Unix()
	}

	return db, &dbConfig, entry, nil
}

// validateMimeType checks the 'Content-Type' header against the db's allowed types.
func (s *entryService) validateMimeType(db *models.Database, header *multipart.FileHeader) (string, error) {
	originalMimeType := header.Header.Get("Content-Type")

	// 1. Run Validation on Original File
	if db.ContentType != "file" {
		var allowedMimeTypes map[string]bool
		switch db.ContentType {
		case "image":
			allowedMimeTypes = map[string]bool{
				"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true,
			}
		case "audio":
			allowedMimeTypes = map[string]bool{
				"audio/mpeg":      true,
				"audio/wav":       true,
				"audio/flac":      true, // Official
				"audio/x-flac":    true, // Browser-reported
				"audio/opus":      true, // Official
				"audio/ogg":       true, // Ogg container (often for Opus/Vorbis)
				"application/ogg": true, // Browser-reported
			}
		}

		if !allowedMimeTypes[originalMimeType] {
			return "", fmt.Errorf("%w: unsupported entry format for this database: %s", ErrUnsupported, originalMimeType)
		}
	}

	return originalMimeType, nil
}
