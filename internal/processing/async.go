package processing

import (
	"context"
	"fmt"
	"os"

	repo "mediahub_oss/internal/repository"
)

func (p *Processor) handleLargeFileAsync(
	ctx context.Context,
	file *os.File,
	db repo.Database,
	req EntryRequest,
	plan ProcessingPlan,
) (repo.Entry, error) {
	httpTempPath := file.Name()

	workerTempFile, err := os.CreateTemp(os.TempDir(), "mh-worker-*")
	if err != nil {
		return repo.Entry{}, fmt.Errorf("failed to create worker temp file: %w", err)
	}
	workerTempPath := workerTempFile.Name()
	workerTempFile.Close()

	file.Close()
	if err := os.Rename(httpTempPath, workerTempPath); err != nil {
		return repo.Entry{}, fmt.Errorf("failed to claim temp file: %w", err)
	}
	p.Logger.Debug("Claimed large file for async processing", "from", httpTempPath, "to", workerTempPath)

	createdEntry, err := p.createPreliminaryEntry(ctx, db, req, plan, repo.EntryStatusProcessing, false)
	if err != nil {
		os.Remove(workerTempPath)
		return repo.Entry{}, err
	}
	p.Logger.Debug("Created partial entry in database", "entry", createdEntry.ID)

	go func() {
		// TODO we should probably increment and decrement counters in this block

		// Run conversion and finalize using the local workerTempPath
		p.runConversionAndFinalize(context.Background(), db, createdEntry, workerTempPath, plan)

		// Now check the queue for next jobs and process them sequentially
		p.runQueueWorkerLoop(context.Background(), db)
	}()

	return createdEntry, nil
}
