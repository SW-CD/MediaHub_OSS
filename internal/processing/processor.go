package processing

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"

	"mediahub_oss/internal/media"
	repo "mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared/customerrors"
	"mediahub_oss/internal/storage"
)

type EntryRequest struct {
	Timestamp    int64
	FileName     string
	CustomFields map[string]any
}

type Processor struct {
	Repo           repo.Repository
	Storage        storage.StorageProvider
	MediaConverter media.MediaConverter
	NFfmpegAsync   int
	NFfmpegTotal   int
	Logger         *slog.Logger

	mu          sync.Mutex
	activeAsync int
	activeTotal int
}

func NewProcessor(
	repository repo.Repository,
	store storage.StorageProvider,
	converter media.MediaConverter,
	nFfmpegAsync int,
	nFfmpegTotal int,
	logger *slog.Logger,
) (*Processor, error) {
	return &Processor{
		Repo:           repository,
		Storage:        store,
		MediaConverter: converter,
		NFfmpegAsync:   nFfmpegAsync,
		NFfmpegTotal:   nFfmpegTotal,
		Logger:         logger,
	}, nil
}

// ProcessEntry is the main entry point to evaluate limits and route files for processing.
func (p *Processor) ProcessEntry(
	ctx context.Context,
	db repo.Database,
	req EntryRequest,
	file io.ReadSeeker,
	originalMimeType string,
	originalFileName string,
) (repo.Entry, bool, error) {
	procPlan, err := DetermineConversionPlan(p.MediaConverter, db, originalMimeType, originalFileName, req.FileName)
	if err != nil {
		return repo.Entry{}, false, err
	}

	var isLarge bool
	var diskFile *os.File
	if f, ok := file.(*os.File); ok {
		isLarge = true
		diskFile = f
	}

	if isLarge {
		// Path A: Large File, Asynchronous
		p.mu.Lock()
		if p.activeAsync < p.NFfmpegAsync && p.activeTotal < p.NFfmpegTotal {
			p.activeAsync++
			p.activeTotal++
			p.mu.Unlock()

			entry, err := p.handleLargeFileAsync(ctx, diskFile, db, req, procPlan)
			if err != nil {
				// TODO, doesnt this decrement the counts while the file is still being converted?
				// decrementing should probably be done in the go routine when it finsihes processing, not here
				// In that case, also incrementing should be done in the go routine when it starts processing, not here above
				p.mu.Lock()
				p.activeAsync--
				p.activeTotal--
				p.mu.Unlock()
				return repo.Entry{}, false, err
			}
			return entry, false, nil
		}
		p.mu.Unlock()

		// Limits reached, evaluate queue limit
		queuedCount, err := p.Repo.CountEntriesByStatus(ctx, db.ID, repo.EntryStatusQueued)
		if err != nil {
			return repo.Entry{}, false, fmt.Errorf("failed to count queued entries: %w", err)
		}

		if int(queuedCount) < db.NMaxQueued {
			entry, err := p.queueLargeFile(ctx, diskFile, db, req, procPlan)
			if err != nil {
				return repo.Entry{}, false, err
			}
			return entry, false, nil
		}

		return repo.Entry{}, false, customerrors.ErrUnavailable
	}

	// Path B: Small File, Synchronous
	p.mu.Lock()
	if p.activeTotal < p.NFfmpegTotal {
		p.activeTotal++
		p.mu.Unlock()

		defer func() {
			p.mu.Lock()
			p.activeTotal--
			p.mu.Unlock()
		}()

		entry, err := p.handleSmallFileSync(ctx, file, db, req, procPlan)
		if err != nil {
			return repo.Entry{}, true, err
		}
		return entry, true, nil
	}
	p.mu.Unlock()

	// Limits reached, evaluate queue limit
	queuedCount, err := p.Repo.CountEntriesByStatus(ctx, db.ID, repo.EntryStatusQueued)
	if err != nil {
		return repo.Entry{}, false, fmt.Errorf("failed to count queued entries: %w", err)
	}

	if int(queuedCount) < db.NMaxQueued {
		entry, err := p.queueSmallFile(ctx, file, db, req, procPlan)
		if err != nil {
			return repo.Entry{}, false, err
		}
		return entry, false, nil
	}

	return repo.Entry{}, false, customerrors.ErrUnavailable
}
