package entryhandler

import (
	"context"
	"fmt"
	"io"
	"os"

	"mediahub_oss/internal/media"
	repo "mediahub_oss/internal/repository"
)

// handleLargeFileAsync handles the asynchronous processing path for large files.
// Files larger than the configured threshold are processed on-disk.
func (h *EntryHandler) handleLargeFileAsync(ctx context.Context, file *os.File, db repo.Database, entryMetadata PostPatchEntryRequest, processingPlan ProcessingPlan) (PartialEntryResponse, error) {
	httpTempPath := file.Name()

	// Claim the file
	workerTempFile, err := os.CreateTemp(os.TempDir(), "mh-worker-*")
	if err != nil {
		return PartialEntryResponse{}, fmt.Errorf("failed to create worker temp file: %w", err)
	}
	workerTempPath := workerTempFile.Name()
	workerTempFile.Close()

	file.Close()
	if err := os.Rename(httpTempPath, workerTempPath); err != nil {
		return PartialEntryResponse{}, fmt.Errorf("failed to claim temp file: %w", err)
	}
	h.Logger.Debug("Claimed large file for async processing", "from", httpTempPath, "to", workerTempPath)

	// Create preliminary database entry
	createdEntry, err := h.createPreliminaryEntry(ctx, db, entryMetadata, processingPlan)
	if err != nil {
		return PartialEntryResponse{}, err
	}
	h.Logger.Debug("Created partial entry in database", "entry", createdEntry.ID)

	// Launch Background Worker
	// We pass a background context because the HTTP request context will cancel when the response is sent!
	go h.processEntry(context.Background(), db, createdEntry, workerTempPath, processingPlan)

	// 5. Return Partial Response immediately
	return PartialEntryResponse{
		DBName:       db.Name,
		ID:           createdEntry.ID,
		Status:       "processing", // Translate uint8 to string for frontend
		Timestamp:    createdEntry.Timestamp.UnixMilli(),
		MimeType:     createdEntry.MimeType,
		CustomFields: createdEntry.CustomFields,
	}, nil
}

// processEntry is the background worker that processes large files via Storage Interfaces.
func (h *EntryHandler) processEntry(ctx context.Context, db repo.Database, entry repo.Entry, workerTempPath string, processingPlan ProcessingPlan) {
	h.Logger.Debug("Worker: Starting processing", "entry", entry.ID)

	var processErr error
	var meta map[string]any = map[string]any{}
	var fileSize int64 = 0
	var err error

	// Track files for cleanup to ensure we don't leak temp files on disk
	currentPath := workerTempPath
	cleanupPaths := []string{workerTempPath}

	// Master error handler & cleanup
	defer func() {
		if processErr != nil {
			h.Logger.Error("Worker: FAILED processing", "entry", entry.ID, "error", processErr)
			entry.Status = repo.EntryStatusError
			if _, updateErr := h.Repo.UpdateEntry(ctx, db.Name, entry); updateErr != nil {
				h.Logger.Error("Worker: CRITICAL: Failed to set status error", "entry", entry.ID, "error", updateErr)
			}
		}
		// Clean up any remaining temp files on disk
		for _, path := range cleanupPaths {
			os.Remove(path)
		}
	}()

	// 1. Conversion Phase (Pure Disk-to-Disk)
	if processingPlan.WantsConversion && processingPlan.NeedsConversion {
		if !processingPlan.CanConvert {
			processErr = fmt.Errorf("cannot convert %v to the database mime type %v: %w", processingPlan.InitMimeType, db.Config.AutoConversion, err)
			return
		}

		convertedTempFile, err := os.CreateTemp(os.TempDir(), "mh-converted-*")
		if err != nil {
			processErr = fmt.Errorf("failed to create converted temp file: %w", err)
			return
		}
		convertedTempPath := convertedTempFile.Name()
		convertedTempFile.Close()

		// Always use ConvertFile for large files to utilize direct I/O
		err = h.MediaConverter.ConvertFile(ctx, currentPath, convertedTempPath, processingPlan.InitMimeType, processingPlan.TargetMimeType)
		if err != nil {
			processErr = fmt.Errorf("conversion to file failed: %w", err)
			return
		}

		// Update our working path and track the new file for cleanup
		cleanupPaths = append(cleanupPaths, convertedTempPath)
		currentPath = convertedTempPath
	}

	// 2. Extract Metadata
	if mf, err := media.GetMetadataFields(db.ContentType); err == nil && len(mf) > 0 {
		meta, err = h.MediaConverter.ReadMediaFieldsFromFile(ctx, currentPath, db.ContentType)
		if err != nil {
			h.Logger.Warn("Worker: Failed to extract metadata", "entry", entry.ID, "error", err)
		}
	}

	// 3. Generate Preview (Disk-to-Storage Pipe)
	if processingPlan.WantsPreview && processingPlan.CanGenPreview {
		pr, pw := io.Pipe()
		errChan := make(chan error, 1)

		go func() {
			defer pw.Close()
			// FFmpeg reads from disk natively, and pipes the small thumbnail bytes into the RAM pipe
			err := h.MediaConverter.CreatePreviewFromFile(ctx, currentPath, pw, processingPlan.TargetMimeType)
			errChan <- err
		}()

		if previewSize, err := h.Storage.WritePreview(ctx, db.Name, entry.ID, pr); err != nil {
			h.Logger.Error("Worker: Failed to save preview to storage", "entry", entry.ID, "error", err)
		} else if genErr := <-errChan; genErr != nil {
			h.Logger.Error("Worker: Failed to generate preview", "entry", entry.ID, "error", genErr)
		} else {
			entry.PreviewSize = uint64(previewSize)
		}
	}

	// 4. Upload to Permanent Storage
	finalFile, err := os.Open(currentPath)
	if err != nil {
		processErr = fmt.Errorf("failed to open final file for storage: %w", err)
		return
	}

	fileSize, err = h.Storage.Write(ctx, db.Name, entry.ID, finalFile)
	finalFile.Close()

	if err != nil {
		processErr = fmt.Errorf("failed to stream file to storage: %w", err)
		return
	}

	// 5. Final Database Update
	entry.Status = repo.EntryStatusReady
	entry.Size = uint64(fileSize)
	entry.MediaFields = meta

	if _, err := h.Repo.UpdateEntry(ctx, db.Name, entry); err != nil {
		processErr = fmt.Errorf("failed to update final database stats: %w", err)
		return
	}

	h.Logger.Info("Worker: Successfully processed large entry", "entry", entry.ID)
}
