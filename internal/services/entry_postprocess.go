// filepath: internal/services/entry_postprocess.go
package services

import (
	"mediahub/internal/logging"
	"mediahub/internal/media"
	"mediahub/internal/models"
	"os"
	"strings"
)

// updateMetadataAsync runs ffprobe/etc. on the final file and updates the DB.
// This is launched as a goroutine for *synchronous* (small file) uploads
// to avoid blocking the API response.
func (s *entryService) updateMetadataAsync(db *models.Database, entryID int64, filePath string, contentType string) {
	logging.Log.Debugf("Starting metadata extraction for entry %d (Path: %s, Type: %s)", entryID, filePath, contentType)

	// 1. Extract metadata from the file on disk
	meta, err := media.ExtractMetadata(filePath, contentType)
	if err != nil {
		logging.Log.Errorf("Failed to extract metadata for entry %d: %v", entryID, err)
		return
	}

	// 2. Create the update map conditionally based on content type
	updates := models.Entry{}
	if strings.HasPrefix(contentType, "image/") {
		updates["width"] = meta.Width
		updates["height"] = meta.Height
	} else if strings.HasPrefix(contentType, "audio/") {
		updates["duration_sec"] = meta.DurationSec
		updates["channels"] = meta.Channels
	} else {
		// 'file' type has no tech metadata to update
		logging.Log.Debugf("No technical metadata to update for entry %d (type '%s')", entryID, contentType)
		return
	}

	// 3. Update the database entry
	if _, err := s.UpdateEntry(db.Name, entryID, updates); err != nil {
		logging.Log.Errorf("Failed to update metadata for entry %d: %v", entryID, err)
		return
	}

	logging.Log.Debugf("Successfully updated metadata for entry %d", entryID)
}

// generateAndSavePreview generates a preview for a file and updates the entry status.
// This is launched as a goroutine for *synchronous* (small file) uploads.
func (s *entryService) generateAndSavePreview(mediaFilePath string, db *models.Database, entry models.Entry) {
	id := entry["id"].(int64)
	timestamp := entry["timestamp"].(int64)

	logging.Log.Debugf("Preview generation starting for %s, entry %d", db.Name, id)

	// --- Status Management ---
	// We defer the status update to ensure the entry transitions out of 'processing'
	// even if preview generation fails. The file itself is safe, so we default to 'ready'.
	// If the preview fails, the frontend will just show a placeholder.
	defer func() {
		logging.Log.Debugf("Marking entry %d as 'ready' after preview attempt.", id)
		// We use s.Repo.UpdateEntry directly to allow updating the protected 'status' field
		// (s.UpdateEntry in service layer sanitizes/removes 'status').
		updates := models.Entry{"status": "ready"}
		if err := s.Repo.UpdateEntry(db.Name, id, updates, db.CustomFields); err != nil {
			logging.Log.Errorf("CRITICAL: Failed to update status to 'ready' for entry %d: %v", id, err)
		}
	}()

	previewPath, err := s.Storage.GetPreviewPath(db.Name, timestamp, id)
	if err != nil {
		logging.Log.Errorf("Failed to get preview path for entry %d: %v", id, err)
		return
	}

	// --- Use FFmpeg for image previews if available ---
	if db.ContentType == "image" && media.IsFFmpegAvailable() {
		logging.Log.Debugf("Generating image preview for entry %d using FFmpeg", id)

		// 1. Open the file to pass as stdin
		previewInputFile, err := os.Open(mediaFilePath)
		if err != nil {
			logging.Log.Errorf("Failed to open file for FFmpeg preview: %v", err)
			return
		}
		defer previewInputFile.Close()

		// 2. Call RunFFmpegToFile with the file as the reader
		err = media.RunFFmpegToFile(
			previewInputFile, // Pass the file handle as the reader
			previewPath,
			"mjpeg",
			"-vf", "scale=200:200:force_original_aspect_ratio=decrease",
			"-q:v", "4", // Good quality JPEG
			"-frames:v", "1", // Ensure only one frame is output
		)

		if err != nil {
			logging.Log.Errorf("Failed to create FFmpeg image preview for entry %d: %v", id, err)
		} else {
			logging.Log.Debugf("Preview generation finished successfully for entry %d (FFmpeg)", id)
		}
		return // Stop here, don't use the Go fallback
	}
	// --- End FFmpeg image preview ---

	// Open the file for pure-Go processing
	previewFile, err := os.Open(mediaFilePath)
	if err != nil {
		logging.Log.Errorf("Failed to open final file for preview (entry %d): %v", id, err)
		return
	}
	defer previewFile.Close()

	switch db.ContentType {
	case "image":
		// This is now the pure-Go fallback
		logging.Log.Warnf("FFmpeg not found. Generating image preview for entry %d using pure Go (high memory risk)", id)
		if err := media.CreateImagePreview(previewFile, previewPath); err != nil {
			logging.Log.Errorf("Failed to create image preview for entry %d: %v", id, err)
		} else {
			logging.Log.Debugf("Preview generation finished successfully for entry %d (Pure Go)", id)
		}
	case "audio":
		logging.Log.Debugf("Generating audio preview for entry %d", id)
		if err := media.CreateAudioPreview(previewFile, previewPath); err != nil {
			logging.Log.Errorf("Failed to create audio preview for entry %d: %v", id, err)
		} else {
			logging.Log.Debugf("Preview generation finished successfully for entry %d (Audio)", id)
		}
	}
}
