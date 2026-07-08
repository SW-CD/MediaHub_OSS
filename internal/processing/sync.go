package processing

import (
	"bytes"
	"context"
	"fmt"
	"io"

	repo "mediahub_oss/internal/repository"
)

func (p *Processor) handleSmallFileSync(
	ctx context.Context,
	file io.ReadSeeker,
	db repo.Database,
	req EntryRequest,
	plan ProcessingPlan,
) (repo.Entry, error) {
	createdEntry, err := p.createPreliminaryEntry(ctx, db, req, plan, repo.EntryStatusProcessing, true)
	if err != nil {
		return repo.Entry{}, err
	}

	cleanupOnError := func(uploadErr error) {
		p.Logger.Error("Upload failed", "entry", createdEntry.ID, "error", uploadErr)
		createdEntry.Status = repo.EntryStatusError
		_, _ = p.Repo.UpdateEntry(ctx, db.ID, createdEntry)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		cleanupOnError(err)
		return repo.Entry{}, fmt.Errorf("failed to seek original file for probing: %w", err)
	}

	meta, metaErr := p.MediaConverter.ReadMediaFieldsFromStream(ctx, file, db.ContentType)
	if metaErr == nil {
		createdEntry.MediaFields = meta
	} else {
		p.Logger.Warn("could not extract metadata from original file", "entryID", createdEntry.ID, "error", metaErr)
	}

	var streamToUpload io.ReadSeeker = file
	if plan.WantsConversion && plan.NeedsConversion {
		if !plan.CanConvert {
			return repo.Entry{}, fmt.Errorf("cannot convert %v to the database mime type %v", plan.InitMimeType, db.Config.AutoConversion)
		}

		if _, err := streamToUpload.Seek(0, io.SeekStart); err != nil {
			cleanupOnError(err)
			return repo.Entry{}, fmt.Errorf("failed to seek input file: %w", err)
		}

		convertedBuffer := new(bytes.Buffer)
		err := p.MediaConverter.ConvertStream(ctx, streamToUpload, convertedBuffer, plan.InitMimeType, plan.ResultMimeType)
		if err != nil {
			cleanupOnError(err)
			return repo.Entry{}, fmt.Errorf("in-memory conversion failed: %w", err)
		}

		streamToUpload = bytes.NewReader(convertedBuffer.Bytes())
	}

	if _, err := streamToUpload.Seek(0, io.SeekStart); err != nil {
		cleanupOnError(err)
		return repo.Entry{}, fmt.Errorf("failed to seek file stream before storage: %w", err)
	}

	fileSize, err := p.Storage.Write(ctx, db.ID.String(), createdEntry.ID, streamToUpload)
	if err != nil {
		cleanupOnError(err)
		return repo.Entry{}, fmt.Errorf("failed to write to storage provider: %w", err)
	}
	createdEntry.Size = uint64(fileSize)

	if plan.WantsPreview && plan.CanGenPreview {
		streamToUpload.Seek(0, io.SeekStart)
		fileBytes, err := io.ReadAll(streamToUpload)
		if err != nil {
			p.Logger.Error("Failed to read file into memory for preview generation", "entry", createdEntry.ID, "error", err)
			createdEntry.Status = repo.EntryStatusReady
		} else {
			createdEntry.Status = repo.EntryStatusProcessing

			go func(bgEntry repo.Entry) {
				var err error
				var previewSize uint64 = 0

				reader := bytes.NewReader(fileBytes)
				if previewSize, err = p.generateAndStorePreview(context.Background(), db, bgEntry.ID, reader, plan.TargetMimeType); err != nil {
					p.Logger.Error("Async preview generation failed", "entry", bgEntry.ID, "error", err)
				}

				bgEntry.Status = repo.EntryStatusReady
				bgEntry.PreviewSize = previewSize

				if _, err := p.Repo.UpdateEntry(context.Background(), db.ID, bgEntry); err != nil {
					p.Logger.Error("Failed to update status to ready after async preview", "entry", bgEntry.ID, "error", err)
				}
			}(createdEntry)
		}
	} else {
		createdEntry.Status = repo.EntryStatusReady
	}

	finalEntry, err := p.Repo.UpdateEntry(ctx, db.ID, createdEntry)
	if err != nil {
		return repo.Entry{}, fmt.Errorf("failed to finalize entry metadata: %w", err)
	}

	return finalEntry, nil
}
