// filepath: internal/services/entry_upload_handlers.go
package services

import (
	"fmt"
	"mediahub/internal/logging"
	"mediahub/internal/media"
	"mediahub/internal/models"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

// handleSmallFileSync handles the synchronous processing path for small, in-memory files.
func (s *entryService) handleSmallFileSync(file multipart.File, header *multipart.FileHeader, db *models.Database, dbConfig *models.DatabaseConfig, entryMetadata models.Entry, originalMime string) (interface{}, int, error) {

	// 3. Determine conversion plan
	originalFilename := header.Filename
	finalMimeType := originalMime
	finalFilename := originalFilename
	needsConversion := false
	ffmpegFormat := ""
	ffmpegArgs := []string{}

	switch db.ContentType {
	case "image":
		if dbConfig.ConvertToJPEG && originalMime != "image/jpeg" {
			needsConversion = true
			finalMimeType = "image/jpeg"
			ffmpegFormat = "mjpeg"
			ffmpegArgs = []string{"-q:v", "3"} // High-quality VBR JPEG
		}
	case "audio":
		targetFormat := dbConfig.AutoConversion
		if targetFormat != "none" && targetFormat != "" {
			if targetFormat == "flac" && !strings.Contains(originalMime, "flac") {
				needsConversion = true
				finalMimeType = "audio/flac" // Native, seekable FLAC
				ffmpegFormat = "flac"
				ffmpegArgs = []string{"-c:a", "flac"}
			} else if targetFormat == "opus" && !strings.Contains(originalMime, "opus") {
				needsConversion = true
				finalMimeType = "audio/opus" // Ogg-Opus is streamable
				ffmpegFormat = "opus"
				ffmpegArgs = []string{"-c:a", "libopus", "-b:a", "96k"}
			}
		}
	}

	// 4. Update filename if conversion happened
	if originalMime != finalMimeType {
		newExtension := getExtensionForMimeType(finalMimeType)
		finalFilename = replaceExtension(originalFilename, newExtension)
	}

	// 5. Start database transaction
	tx, err := s.Repo.BeginTx()
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to start database transaction: %w", err)
	}
	defer tx.Rollback() // Rollback on any error

	// 6. Create preliminary entry record to get the ID
	entryMetadata["mime_type"] = finalMimeType
	entryMetadata["filename"] = finalFilename
	entryMetadata["width"] = 0
	entryMetadata["height"] = 0
	entryMetadata["duration_sec"] = 0
	entryMetadata["channels"] = 0

	// --- STATUS LOGIC UPDATE ---
	// If previews are enabled, we set the status to "processing".
	// The background goroutine will flip it to "ready" when done.
	// If previews are disabled, we are effectively "ready" immediately.
	initialStatus := "ready"
	if dbConfig.CreatePreview {
		initialStatus = "processing"
	}
	entryMetadata["status"] = initialStatus

	preliminaryEntry, err := tx.CreateEntryInTx(db.Name, db.ContentType, entryMetadata, db.CustomFields)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create entry record in transaction: %w", err)
	}
	id := preliminaryEntry["id"].(int64)
	timestamp := preliminaryEntry["timestamp"].(int64)

	// 7. Get the final, permanent file path
	permanentEntryPath, err := s.Storage.GetEntryPath(db.Name, timestamp, id)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to generate entry path: %w", err)
	}

	// 8. Execute file operation (Convert or Save)
	var fileSize int64

	// Define a cleanup function in case of error
	cleanupOnError := func() {
		os.Remove(permanentEntryPath) // Attempt to delete partial file
		tx.Rollback()
	}

	if needsConversion {
		// --- Conversion is needed ---
		if _, err := file.Seek(0, 0); err != nil { // Rewind the input file
			cleanupOnError()
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to seek input file for conversion: %w", err)
		}

		if media.IsFFmpegAvailable() {
			// --- 8a. Convert with FFmpeg ---
			logging.Log.Debugf("Converting file for entry %d using FFmpeg (to file)", id)
			err = media.RunFFmpegToFile(file, permanentEntryPath, ffmpegFormat, ffmpegArgs...)
			if err != nil {
				cleanupOnError()
				return nil, http.StatusInternalServerError, fmt.Errorf("failed to convert (ffmpeg) and save file: %w", err)
			}
		} else if db.ContentType == "image" {
			// --- 8b. Convert Image with Pure Go Fallback ---
			logging.Log.Warnf("FFmpeg not found. Attempting pure Go conversion for image entry %d", id)
			err = media.ConvertImagePureGoToFile(file, permanentEntryPath)
			if err != nil {
				cleanupOnError()
				return nil, http.StatusInternalServerError, fmt.Errorf("failed to convert (pure go) and save file: %w", err)
			}
		} else {
			// --- 8c. Cannot convert (e.g., Audio without FFmpeg) ---
			cleanupOnError()
			logging.Log.Errorf("Cannot convert entry %d: conversion required but FFmpeg is not available", id)
			return nil, http.StatusBadRequest, fmt.Errorf("%w: conversion required for %s but FFmpeg is not available", ErrDependencies, db.ContentType)
		}
	} else {
		// --- 8d. No conversion needed, just save ---
		if _, err := file.Seek(0, 0); err != nil { // Rewind the input file
			cleanupOnError()
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to seek input file for saving: %w", err)
		}
		logging.Log.Debugf("No conversion needed. Saving original file for entry %d", id)
		fileSize, err = s.Storage.SaveFile(file, permanentEntryPath)
		if err != nil {
			cleanupOnError()
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to save file: %w", err)
		}
	}

	// 9. Get filesize (especially important after conversion)
	if fileSize == 0 { // fileSize is only set in 8d
		fileInfo, err := os.Stat(permanentEntryPath)
		if err != nil {
			cleanupOnError()
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to get file info after conversion: %w", err)
		}
		fileSize = fileInfo.Size()
	}

	// 10. Update the entry with the final filesize
	updates := models.Entry{"filesize": fileSize}
	if err := tx.UpdateEntryInTx(db.Name, id, updates, db.CustomFields); err != nil {
		cleanupOnError()
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to update entry metadata: %w", err)
	}

	// 11. Update database stats
	if err := tx.UpdateStatsInTx(db.Name, 1, fileSize); err != nil {
		cleanupOnError()
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to update database stats: %w", err)
	}

	// 12. Commit the transaction
	if err := tx.Commit(); err != nil {
		cleanupOnError()
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 13. Launch async tasks *after* commit
	// --- FIX: Only generate preview if enabled in config ---
	if dbConfig.CreatePreview {
		go s.generateAndSavePreview(permanentEntryPath, db, preliminaryEntry)
	}
	go s.updateMetadataAsync(db, id, permanentEntryPath, finalMimeType)

	// 14. Get the final, complete entry data and return
	finalEntry, err := s.Repo.GetEntry(db.Name, id, db.CustomFields)
	if err != nil {
		logging.Log.Errorf("EntryService: Failed to retrieve final entry data for %d: %v", id, err)
		// Return preliminary as a fallback, but ensure status matches our logic
		preliminaryEntry["status"] = initialStatus
		return preliminaryEntry, http.StatusCreated, nil
	}

	// --- Return 201 Created ---
	return finalEntry, http.StatusCreated, nil
}

// handleLargeFileAsync handles the asynchronous processing path for large, on-disk files.
func (s *entryService) handleLargeFileAsync(file *os.File, header *multipart.FileHeader, db *models.Database, dbConfig *models.DatabaseConfig, entryMetadata models.Entry, originalMime string) (interface{}, int, error) {
	// 1. Get the path of the *http temp file*
	httpTempPath := file.Name()

	// 2. Create a *new* temp file for our worker to "claim"
	workerTempFile, err := os.CreateTemp(os.TempDir(), "fdb-worker-*")
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create worker temp file: %w", err)
	}
	workerTempPath := workerTempFile.Name()
	if err := workerTempFile.Close(); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to close worker temp file: %w", err)
	}

	// 3. "Claim" the file: Close the http file handle and do a fast rename
	if err := file.Close(); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to close http temp file: %w", err)
	}
	if err := os.Rename(httpTempPath, workerTempPath); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to 'claim' temp file: %w", err)
	}
	logging.Log.Debugf("Claimed temp file: %s -> %s", httpTempPath, workerTempPath)

	// 4. Start DB transaction
	tx, err := s.Repo.BeginTx()
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to start database transaction: %w", err)
	}
	defer tx.Rollback()

	// 5. Create the entry with "processing" status and partial metadata
	entryMetadata["mime_type"] = originalMime // Will be updated by worker if converted
	entryMetadata["filename"] = header.Filename
	entryMetadata["width"] = 0
	entryMetadata["height"] = 0
	entryMetadata["duration_sec"] = 0
	entryMetadata["channels"] = 0
	entryMetadata["status"] = "processing" // <-- SET 'processing' STATUS

	preliminaryEntry, err := tx.CreateEntryInTx(db.Name, db.ContentType, entryMetadata, db.CustomFields)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create 'processing' entry: %w", err)
	}
	id := preliminaryEntry["id"].(int64)
	timestamp := preliminaryEntry["timestamp"].(int64)

	// 6. Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to commit 'processing' transaction: %w", err)
	}

	// 7. Launch the background worker (goroutine)
	logging.Log.Infof("Handing off entry %d to background worker. Temp file: %s", id, workerTempPath)
	go s.processEntry(db, dbConfig, id, timestamp, workerTempPath, originalMime, header.Filename)

	// 8. Build and return the PartialEntryResponse
	userCustomFields := models.Entry{}
	for k, v := range entryMetadata {
		if _, isInternal := preliminaryEntry[k]; !isInternal {
			userCustomFields[k] = v
		}
	}

	partialResponse := models.PartialEntryResponse{
		ID:           id,
		Timestamp:    timestamp,
		DatabaseName: db.Name,
		Status:       "processing",
		CustomFields: userCustomFields,
	}

	return partialResponse, http.StatusAccepted, nil
}

// processEntry is the background worker that processes large files.
func (s *entryService) processEntry(db *models.Database, dbConfig *models.DatabaseConfig, id, timestamp int64, workerTempPath, originalMime, originalFilename string) {
	logging.Log.Debugf("Worker: Starting processing for entry %d", id)
	currentPath := workerTempPath
	cleanupPaths := []string{workerTempPath}
	finalMimeType := originalMime
	finalFilename := originalFilename
	var processErr error

	// Defer a master error handler to ensure status is updated and files are cleaned up
	defer func() {
		if processErr != nil {
			logging.Log.Errorf("Worker: FAILED processing for entry %d: %v", id, processErr)
			updateErr := s.Repo.UpdateEntry(db.Name, id, models.Entry{"status": "error"}, db.CustomFields)
			if updateErr != nil {
				logging.Log.Errorf("Worker: CRITICAL: Failed to set status='error' for entry %d: %v", id, updateErr)
			}
		}
		for _, path := range cleanupPaths {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				logging.Log.Warnf("Worker: Failed to clean up temp file %s: %v", path, err)
			}
		}
		logging.Log.Debugf("Worker: Finished cleanup for entry %d", id)
	}()

	// --- 1. Determine Conversion Plan (copied from sync path) ---
	needsConversion := false
	ffmpegFormat := ""
	ffmpegArgs := []string{}

	switch db.ContentType {
	case "image":
		if dbConfig.ConvertToJPEG && originalMime != "image/jpeg" {
			needsConversion = true
			finalMimeType = "image/jpeg"
			ffmpegFormat = "mjpeg"
			ffmpegArgs = []string{"-q:v", "3"}
		}
	case "audio":
		targetFormat := dbConfig.AutoConversion
		if targetFormat != "none" && targetFormat != "" {
			if targetFormat == "flac" && !strings.Contains(originalMime, "flac") {
				needsConversion = true
				finalMimeType = "audio/flac"
				ffmpegFormat = "flac"
				ffmpegArgs = []string{"-c:a", "flac"}
			} else if targetFormat == "opus" && !strings.Contains(originalMime, "opus") {
				needsConversion = true
				finalMimeType = "audio/opus"
				ffmpegFormat = "opus"
				ffmpegArgs = []string{"-c:a", "libopus", "-b:a", "96k"}
			}
		}
	}

	if originalMime != finalMimeType {
		newExtension := getExtensionForMimeType(finalMimeType)
		finalFilename = replaceExtension(originalFilename, newExtension)
	}

	// --- 2. Perform Conversion (if needed) ---
	if needsConversion {
		if !media.IsFFmpegAvailable() {
			processErr = fmt.Errorf("%w: conversion required but FFmpeg is not available", ErrDependencies)
			return
		}

		convertedTempFile, err := os.CreateTemp(os.TempDir(), "fdb-converted-*")
		if err != nil {
			processErr = fmt.Errorf("failed to create converted temp file: %w", err)
			return
		}
		convertedTempPath := convertedTempFile.Name()
		if err := convertedTempFile.Close(); err != nil {
			processErr = fmt.Errorf("failed to close converted temp file: %w", err)
			return
		}

		inputFile, err := os.Open(currentPath)
		if err != nil {
			processErr = fmt.Errorf("failed to open temp file for conversion: %w", err)
			return
		}

		logging.Log.Debugf("Worker: Converting entry %d (%s) -> (%s)", id, currentPath, convertedTempPath)
		err = media.RunFFmpegToFile(inputFile, convertedTempPath, ffmpegFormat, ffmpegArgs...)
		inputFile.Close() // Close input file immediately
		if err != nil {
			processErr = fmt.Errorf("ffmpeg conversion failed: %w", err)
			return
		}

		cleanupPaths = append(cleanupPaths, convertedTempPath)
		currentPath = convertedTempPath
	}

	// --- 3. Generate Preview (if needed) ---
	var previewTempPath string
	if dbConfig.CreatePreview {
		previewTempFile, err := os.CreateTemp(os.TempDir(), "fdb-preview-*")
		if err != nil {
			processErr = fmt.Errorf("failed to create preview temp file: %w", err)
			return
		}
		previewTempPath = previewTempFile.Name()
		if err := previewTempFile.Close(); err != nil {
			processErr = fmt.Errorf("failed to close preview temp file: %w", err)
			return
		}

		inputFile, err := os.Open(currentPath)
		if err != nil {
			processErr = fmt.Errorf("failed to open file for preview: %w", err)
			return
		}

		logging.Log.Debugf("Worker: Generating preview for entry %d -> %s", id, previewTempPath)
		switch db.ContentType {
		case "image":
			err = media.CreateImagePreview(inputFile, previewTempPath)
		case "audio":
			err = media.CreateAudioPreview(inputFile, previewTempPath)
		}
		inputFile.Close() // Close input file immediately
		if err != nil {
			processErr = fmt.Errorf("failed to generate preview: %w", err)
			return
		}
		cleanupPaths = append(cleanupPaths, previewTempPath)
	}

	// --- 4. Extract Metadata ---
	logging.Log.Debugf("Worker: Extracting metadata for entry %d from %s", id, currentPath)
	meta, err := media.ExtractMetadata(currentPath, finalMimeType)
	if err != nil {
		logging.Log.Warnf("Worker: Failed to extract metadata for entry %d: %v", id, err)
		meta = &models.MediaMetadata{} // Use empty metadata struct
	}

	// --- 5. Move Files to Permanent Storage ---
	permanentEntryPath, err := s.Storage.GetEntryPath(db.Name, timestamp, id)
	if err != nil {
		processErr = fmt.Errorf("failed to get permanent entry path: %w", err)
		return
	}
	if err := os.Rename(currentPath, permanentEntryPath); err != nil {
		processErr = fmt.Errorf("failed to move temp file to permanent storage: %w", err)
		return
	}
	logging.Log.Debugf("Worker: Moved entry %d to %s", id, permanentEntryPath)

	if previewTempPath != "" {
		permanentPreviewPath, err := s.Storage.GetPreviewPath(db.Name, timestamp, id)
		if err != nil {
			processErr = fmt.Errorf("failed to get permanent preview path: %w", err)
			return
		}
		if err := os.Rename(previewTempPath, permanentPreviewPath); err != nil {
			processErr = fmt.Errorf("failed to move temp preview to permanent storage: %w", err)
			return
		}
		logging.Log.Debugf("Worker: Moved preview %d to %s", id, permanentPreviewPath)
	}

	// --- 6. Update Database with Final Metadata ---
	fileInfo, err := os.Stat(permanentEntryPath)
	if err != nil {
		processErr = fmt.Errorf("failed to stat final file: %w", err)
		return
	}
	fileSize := fileInfo.Size()

	updates := models.Entry{
		"status":    "ready",
		"filesize":  fileSize,
		"mime_type": finalMimeType,
		"filename":  finalFilename,
	}

	// Use the DATABASE's content type (db.ContentType)
	// to determine which fields to update, not the FILE's mime type (finalMimeType).
	switch db.ContentType {
	case "image":
		updates["width"] = meta.Width
		updates["height"] = meta.Height
	case "audio":
		updates["duration_sec"] = meta.DurationSec
		updates["channels"] = meta.Channels
	case "file":
		// No additional metadata fields to update.
	}

	tx, err := s.Repo.BeginTx()
	if err != nil {
		processErr = fmt.Errorf("failed to start final transaction: %w", err)
		return
	}

	// 6a. Update the entry
	if err := tx.UpdateEntryInTx(db.Name, id, updates, db.CustomFields); err != nil {
		tx.Rollback()
		processErr = fmt.Errorf("failed to update final entry metadata: %w", err)
		return
	}

	// 6b. Update the database stats (Entry count +1, Size +fileSize)
	if err := tx.UpdateStatsInTx(db.Name, 1, fileSize); err != nil {
		tx.Rollback()
		processErr = fmt.Errorf("failed to update final database stats: %w", err)
		return
	}

	// 6c. Commit
	if err := tx.Commit(); err != nil {
		processErr = fmt.Errorf("failed to commit final transaction: %w", err)
		return
	}

	logging.Log.Infof("Worker: Successfully processed entry %d", id)
}
