package processing

import (
	"context"
	"fmt"
	"io"
	"os"

	"mediahub_oss/internal/media"
	repo "mediahub_oss/internal/repository"
)

// StartQueueChecker scans for hanging queued entries on startup and processes them.
func (p *Processor) StartQueueChecker(ctx context.Context) {
	p.Logger.Info("Starting background queue checker to scan for hanging queued entries...")

	databases, err := p.Repo.GetDatabases(ctx)
	if err != nil {
		p.Logger.Error("QueueChecker: Failed to get databases", "error", err)
		return
	}

	for _, db := range databases {
		queuedEntries, err := p.Repo.GetEntriesByStatus(ctx, db.ID, repo.EntryStatusQueued)
		if err != nil {
			p.Logger.Error("QueueChecker: Failed to get queued entries", "database_id", db.ID, "error", err)
			continue
		}

		for _, entry := range queuedEntries {
			if !p.tryAcquireAndSpawn(ctx, db, entry) {
				p.Logger.Info("QueueChecker: Concurrency limits reached, stopping initial queue scan.")
				return
			}
		}
	}
}

func (p *Processor) tryAcquireAndSpawn(ctx context.Context, db repo.Database, entry repo.Entry) bool {
	p.mu.Lock()
	if p.activeAsync >= p.NFfmpegAsync || p.activeTotal >= p.NFfmpegTotal {
		p.mu.Unlock()
		return false
	}
	p.activeAsync++
	p.activeTotal++
	p.mu.Unlock()

	claimed, err := p.Repo.ClaimQueuedEntry(ctx, db.ID, entry.ID)
	if err != nil {
		p.Logger.Error("Failed to claim queued entry", "database_id", db.ID, "entry_id", entry.ID, "error", err)
		p.mu.Lock()
		p.activeAsync--
		p.activeTotal--
		p.mu.Unlock()
		return false
	}

	if !claimed {
		p.mu.Lock()
		p.activeAsync--
		p.activeTotal--
		p.mu.Unlock()
		return true // continue scanning
	}

	go p.runWorkerForClaimedEntry(ctx, db, entry)
	return true
}

func (p *Processor) runWorkerForClaimedEntry(ctx context.Context, db repo.Database, entry repo.Entry) {
	tempFile, err := os.CreateTemp(os.TempDir(), "mh-worker-queued-*")
	if err != nil {
		p.Logger.Error("Worker: Failed to create temp file for queued entry", "entry", entry.ID, "error", err)
		entry.Status = repo.EntryStatusError
		_, _ = p.Repo.UpdateEntry(ctx, db.ID, entry)

		p.mu.Lock()
		p.activeAsync--
		p.activeTotal--
		p.mu.Unlock()
		return
	}
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)

	stream, err := p.Storage.Read(ctx, db.ID, entry.ID, 0, -1)
	if err != nil {
		p.Logger.Error("Worker: Failed to read queued file from storage", "entry", entry.ID, "error", err)
		tempFile.Close()
		entry.Status = repo.EntryStatusError
		_, _ = p.Repo.UpdateEntry(ctx, db.ID, entry)

		p.mu.Lock()
		p.activeAsync--
		p.activeTotal--
		p.mu.Unlock()
		return
	}

	_, err = io.Copy(tempFile, stream)
	stream.Close()
	tempFile.Close()

	if err != nil {
		p.Logger.Error("Worker: Failed to copy queued file to temp path", "entry", entry.ID, "error", err)
		entry.Status = repo.EntryStatusError
		_, _ = p.Repo.UpdateEntry(ctx, db.ID, entry)

		p.mu.Lock()
		p.activeAsync--
		p.activeTotal--
		p.mu.Unlock()
		return
	}

	plan := DeterminePlanForEntry(p.MediaConverter, db, entry)
	p.runConversionAndFinalize(ctx, db, entry, tempFilePath, plan)

	// Now check the queue for next jobs and process them sequentially
	p.runQueueWorkerLoop(ctx, db)
}

func (p *Processor) runQueueWorkerLoop(ctx context.Context, initialDB repo.Database) {
	db := initialDB
	for {
		nextEntry, nextDB, found, err := p.findNextQueuedEntry(ctx)
		if err != nil {
			p.Logger.Error("Worker: Failed to scan for next queued entry", "error", err)
			break
		}
		if !found {
			break
		}

		claimed, err := p.Repo.ClaimQueuedEntry(ctx, nextDB.ID, nextEntry.ID)
		if err != nil {
			p.Logger.Error("Worker: Failed to claim next queued entry", "database_id", nextDB.ID, "entry_id", nextEntry.ID, "error", err)
			break
		}
		if !claimed {
			continue
		}

		db = nextDB
		tempFile, err := os.CreateTemp(os.TempDir(), "mh-worker-queued-*")
		if err != nil {
			p.Logger.Error("Worker: Failed to create temp file for claimed entry", "entry", nextEntry.ID, "error", err)
			nextEntry.Status = repo.EntryStatusError
			_, _ = p.Repo.UpdateEntry(ctx, db.ID, nextEntry)
			continue
		}
		tempFilePath := tempFile.Name()

		stream, err := p.Storage.Read(ctx, db.ID, nextEntry.ID, 0, -1)
		if err != nil {
			p.Logger.Error("Worker: Failed to read claimed file from storage", "entry", nextEntry.ID, "error", err)
			tempFile.Close()
			os.Remove(tempFilePath)
			nextEntry.Status = repo.EntryStatusError
			_, _ = p.Repo.UpdateEntry(ctx, db.ID, nextEntry)
			continue
		}

		_, err = io.Copy(tempFile, stream)
		stream.Close()
		tempFile.Close()

		if err != nil {
			p.Logger.Error("Worker: Failed to copy claimed file to temp path", "entry", nextEntry.ID, "error", err)
			os.Remove(tempFilePath)
			nextEntry.Status = repo.EntryStatusError
			_, _ = p.Repo.UpdateEntry(ctx, db.ID, nextEntry)
			continue
		}

		plan := DeterminePlanForEntry(p.MediaConverter, db, nextEntry)
		p.runConversionAndFinalize(ctx, db, nextEntry, tempFilePath, plan)
		os.Remove(tempFilePath)
	}

	p.mu.Lock()
	p.activeAsync--
	p.activeTotal--
	p.mu.Unlock()
	p.Logger.Debug("Worker: Terminating queue worker.")
}

func (p *Processor) runConversionAndFinalize(
	ctx context.Context,
	db repo.Database,
	entry repo.Entry,
	originalTempPath string,
	plan ProcessingPlan,
) {
	p.Logger.Debug("Worker: Starting conversion and finalize", "entry", entry.ID)

	var processErr error
	var meta map[string]any = map[string]any{}
	var fileSize int64 = 0

	currentPath := originalTempPath
	cleanupPaths := []string{originalTempPath}

	defer func() {
		if processErr != nil {
			p.Logger.Error("Worker: FAILED processing", "entry", entry.ID, "error", processErr)
			entry.Status = repo.EntryStatusError
			if _, updateErr := p.Repo.UpdateEntry(ctx, db.ID, entry); updateErr != nil {
				p.Logger.Error("Worker: CRITICAL: Failed to set status error", "entry", entry.ID, "error", updateErr)
			}
		}
		for _, path := range cleanupPaths {
			os.Remove(path)
		}
	}()

	if plan.WantsConversion && plan.NeedsConversion {
		if !plan.CanConvert {
			processErr = fmt.Errorf("cannot convert %v to the database mime type %v", plan.InitMimeType, db.Config.AutoConversion)
			return
		}

		convertedTempFile, err := os.CreateTemp(os.TempDir(), "mh-converted-*")
		if err != nil {
			processErr = fmt.Errorf("failed to create converted temp file: %w", err)
			return
		}
		convertedTempPath := convertedTempFile.Name()
		convertedTempFile.Close()

		err = p.MediaConverter.ConvertFile(ctx, currentPath, convertedTempPath, plan.InitMimeType, plan.TargetMimeType)
		if err != nil {
			processErr = fmt.Errorf("conversion to file failed: %w", err)
			return
		}

		cleanupPaths = append(cleanupPaths, convertedTempPath)
		currentPath = convertedTempPath
	}

	if mf, err := media.GetMetadataFields(db.ContentType); err == nil && len(mf) > 0 {
		meta, err = p.MediaConverter.ReadMediaFieldsFromFile(ctx, currentPath, db.ContentType)
		if err != nil {
			p.Logger.Warn("Worker: Failed to extract metadata", "entry", entry.ID, "error", err)
		}
	}

	if plan.WantsPreview && plan.CanGenPreview {
		pr, pw := io.Pipe()
		errChan := make(chan error, 1)

		go func() {
			defer pw.Close()
			err := p.MediaConverter.CreatePreviewFromFile(ctx, currentPath, pw, plan.TargetMimeType)
			errChan <- err
		}()

		if previewSize, err := p.Storage.WritePreview(ctx, db.ID, entry.ID, pr); err != nil {
			p.Logger.Error("Worker: Failed to save preview to storage", "entry", entry.ID, "error", err)
		} else if genErr := <-errChan; genErr != nil {
			p.Logger.Error("Worker: Failed to generate preview", "entry", entry.ID, "error", genErr)
		} else {
			entry.PreviewSize = uint64(previewSize)
		}
	}

	finalFile, err := os.Open(currentPath)
	if err != nil {
		processErr = fmt.Errorf("failed to open final file for storage: %w", err)
		return
	}

	fileSize, err = p.Storage.Write(ctx, db.ID, entry.ID, finalFile)
	finalFile.Close()

	if err != nil {
		processErr = fmt.Errorf("failed to stream file to storage: %w", err)
		return
	}

	entry.Status = repo.EntryStatusReady
	entry.Size = uint64(fileSize)
	entry.MimeType = plan.ResultMimeType
	entry.MediaFields = meta

	if _, err := p.Repo.UpdateEntry(ctx, db.ID, entry); err != nil {
		processErr = fmt.Errorf("failed to update final database stats: %w", err)
		return
	}

	p.Logger.Info("Worker: Successfully processed large entry", "entry", entry.ID)
}
