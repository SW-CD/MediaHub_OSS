package processing

import (
	"context"
	"fmt"
	"io"
	"time"

	repo "mediahub_oss/internal/repository"
)

func (p *Processor) createPreliminaryEntry(
	ctx context.Context,
	db repo.Database,
	entryMetadata EntryRequest,
	plan ProcessingPlan,
	status repo.EntryStatus,
	useResultMimeType bool,
) (repo.Entry, error) {
	var err error

	partialEntry := repo.Entry{}
	partialEntry.FileName = plan.FinalFileName
	partialEntry.Timestamp = time.UnixMilli(entryMetadata.Timestamp)
	if useResultMimeType {
		partialEntry.MimeType = plan.ResultMimeType
	} else {
		partialEntry.MimeType = plan.InitMimeType
	}
	partialEntry.Status = status

	partialEntry.MediaFields, err = DefaultMediaFields(db.ContentType)
	if err != nil {
		return repo.Entry{}, fmt.Errorf("failed to create default media fields: %w", err)
	}

	partialEntry.CustomFields = entryMetadata.CustomFields

	createdEntry, err := p.Repo.CreateEntry(ctx, db, partialEntry)
	if err != nil {
		return repo.Entry{}, fmt.Errorf("failed to create partial database entry: %w", err)
	}

	return createdEntry, nil
}

func (p *Processor) generateAndStorePreview(
	ctx context.Context,
	db repo.Database,
	entryID int64,
	inputSeeker io.ReadSeeker,
	mimeType string,
) (uint64, error) {
	pr, pw := io.Pipe()
	errChan := make(chan error, 1)

	go func() {
		defer pw.Close()
		err := p.MediaConverter.CreatePreviewFromStream(ctx, inputSeeker, pw, mimeType)
		errChan <- err
	}()

	previewSize, err := p.Storage.WritePreview(ctx, db.ID, entryID, pr)
	if err != nil {
		return 0, fmt.Errorf("failed to save preview to storage: %w", err)
	}

	if genErr := <-errChan; genErr != nil {
		return 0, fmt.Errorf("failed to generate preview: %w", genErr)
	}

	return uint64(previewSize), nil
}
