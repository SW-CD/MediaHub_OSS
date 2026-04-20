package entryhandler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"

	repo "mediahub_oss/internal/repository"
)

// Files smaller than or equal to the configured threshold are processed synchronously entirely in RAM.
// handleSmallFileSync handles the synchronous processing path for small files.
func (h *EntryHandler) handleSmallFileSync(ctx context.Context, file multipart.File, db repo.Database, entryMetadata PostPatchEntryRequest, convPlan ProcessingPlan) (EntryResponse, error) {

	// Create preliminary entry
	createdEntry, err := h.createPreliminaryEntry(ctx, db, entryMetadata, convPlan)
	if err != nil {
		return EntryResponse{}, err
	}

	cleanupOnError := func(uploadErr error) {
		h.Logger.Error("Upload failed", "entry", createdEntry.ID, "error", uploadErr)
		createdEntry.Status = repo.EntryStatusError
		_, _ = h.Repo.UpdateEntry(ctx, db.Name, createdEntry)
	}

	// 1. EXTRACT METADATA FIRST!
	// We read from the original uploaded file because it has a fully intact metadata index,
	// unlike piped conversions which lack global duration headers.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		cleanupOnError(err)
		return EntryResponse{}, fmt.Errorf("failed to seek original file for probing: %w", err)
	}

	meta, metaErr := h.MediaConverter.ReadMediaFieldsFromStream(ctx, file, db.ContentType)
	if metaErr == nil {
		createdEntry.MediaFields = meta
	} else {
		h.Logger.Warn("could not extract metadata from original file", "entryID", createdEntry.ID, "error", metaErr)
	}

	// 2. CONVERSION PHASE
	var streamToUpload io.ReadSeeker = file

	if convPlan.WantsConversion && convPlan.NeedsConversion {
		if !convPlan.CanConvert {
			return EntryResponse{}, fmt.Errorf("cannot convert %v to the database mime type %v", convPlan.InitMimeType, db.Config.AutoConversion)
		}

		if _, err := streamToUpload.Seek(0, io.SeekStart); err != nil {
			cleanupOnError(err)
			return EntryResponse{}, fmt.Errorf("failed to seek input file: %w", err)
		}

		convertedBuffer := new(bytes.Buffer)
		err := h.MediaConverter.ConvertStream(ctx, streamToUpload, convertedBuffer, convPlan.InitMimeType, convPlan.ResultMimeType)
		if err != nil {
			cleanupOnError(err)
			return EntryResponse{}, fmt.Errorf("in-memory conversion failed: %w", err)
		}

		streamToUpload = bytes.NewReader(convertedBuffer.Bytes())
	}

	// 3. STORAGE UPLOAD
	if _, err := streamToUpload.Seek(0, io.SeekStart); err != nil {
		cleanupOnError(err)
		return EntryResponse{}, fmt.Errorf("failed to seek file stream before storage: %w", err)
	}

	fileSize, err := h.Storage.Write(ctx, db.Name, createdEntry.ID, streamToUpload)
	if err != nil {
		cleanupOnError(err)
		return EntryResponse{}, fmt.Errorf("failed to write to storage provider: %w", err)
	}
	createdEntry.Size = uint64(fileSize)

	// 4. ASYNC PREVIEW GENERATION
	if convPlan.WantsPreview && convPlan.CanGenPreview {
		streamToUpload.Seek(0, io.SeekStart)

		fileBytes, err := io.ReadAll(streamToUpload)

		if err != nil {
			h.Logger.Error("Failed to read file into memory for preview generation", "entry", createdEntry.ID, "error", err)
			createdEntry.Status = repo.EntryStatusReady
		} else {

			createdEntry.Status = repo.EntryStatusProcessing

			go func(bgEntry repo.Entry) {
				var err error
				var previewSize uint64 = 0

				reader := bytes.NewReader(fileBytes)

				if previewSize, err = h.generateAndStorePreview(context.Background(), db, bgEntry.ID, reader, convPlan.TargetMimeType); err != nil {
					h.Logger.Error("Async preview generation failed", "entry", bgEntry.ID, "error", err)
				}

				bgEntry.Status = repo.EntryStatusReady
				bgEntry.PreviewSize = previewSize

				if _, err := h.Repo.UpdateEntry(context.Background(), db.Name, bgEntry); err != nil {
					h.Logger.Error("Failed to update status to ready after async preview", "entry", bgEntry.ID, "error", err)
				}
			}(createdEntry)
		}
	} else {
		createdEntry.Status = repo.EntryStatusReady
	}

	// 5. FINALIZE DB ENTRY
	finalEntry, err := h.Repo.UpdateEntry(ctx, db.Name, createdEntry)
	if err != nil {
		return EntryResponse{}, fmt.Errorf("failed to finalize entry metadata: %w", err)
	}

	return mapToEntryResponse(db.Name, finalEntry), nil
}
