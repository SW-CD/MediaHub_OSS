package recovery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared"
	"mediahub_oss/internal/shared/customerrors"
)

// EntryStatusCorrection scans for entries stuck in 'processing' or 'deleting' and rectifies their state.
func (s *RecoveryService) EntryStatusCorrection(ctx context.Context) error {
	databases, err := s.repo.GetDatabases(ctx)
	if err != nil {
		return fmt.Errorf("could not fetch databases: %w", err)
	}

	for _, db := range databases {
		fmt.Printf("Database %q\n", db.Name)

		stats, err := s.repo.GetDatabaseStats(ctx, db.ID)
		if err != nil {
			s.logger.Error("Failed to get database stats", "database_id", db.ID, "database_name", db.Name, "error", err)
			continue
		}

		totalEntries := stats.EntryCount
		if totalEntries == 0 {
			fmt.Printf("- Step 1: Entry Status Corrections: 100%% (0 entries)\n")
			continue
		}

		// Slices to hold IDs for bulk operations
		var markReadyIDs []int64
		var deleteZombiesIDs []int64
		var deleteStuckIDs []int64

		limit := 1000
		processed := uint64(0)

		// --- PHASE 1: Scanning ---
		for offset := 0; uint64(offset) < totalEntries; offset += limit {
			entries, err := s.repo.GetEntries(ctx, db.ID, limit, offset, "id asc", time.Time{}, time.Time{})
			if err != nil {
				return fmt.Errorf("failed to fetch entries for %s: %w", db.Name, err)
			}

			// Failsafe if totalEntries drifted during run
			if len(entries) == 0 {
				break
			}

			for _, entry := range entries {
				processed++

				// Update console with carriage return (\r) to overwrite the line
				percent := (processed * 100) / totalEntries
				fmt.Printf("\r- Step 1: Entry Status Corrections: %d%% (Scanning...)", percent)

				if entry.Status == repository.EntryStatusProcessing {
					_, err := s.storage.Stat(ctx, db.ID, entry.ID)
					if errors.Is(err, customerrors.ErrNotFound) {
						// File missing -> mark DB entry for deletion
						deleteZombiesIDs = append(deleteZombiesIDs, entry.ID)
					} else if err == nil {
						// File exists -> mark for ready status
						markReadyIDs = append(markReadyIDs, entry.ID)
					} else {
						fmt.Println() // Break the \r progress line so the log prints cleanly
						s.logger.Error("Storage stat failed", "database_id", db.ID, "database_name", db.Name, "id", entry.ID, "error", err)
					}
				} else if entry.Status == repository.EntryStatusDeleting {
					// Entry stuck deleting -> mark for full cleanup
					deleteStuckIDs = append(deleteStuckIDs, entry.ID)
				}
			}
		}

		// Overwrite the scanning line with the final completion state
		fmt.Printf("\r- Step 1: Entry Status Corrections: 100%% (Applying fixes)    \n")

		// --- PHASE 2: Action & Summarize ---
		if !s.dryRun {
			// 1. Mark valid processing files as ready
			if len(markReadyIDs) > 0 {
				_ = s.repo.UpdateEntriesStatus(ctx, db.ID, markReadyIDs, repository.EntryStatusReady)
			}

			// 2. Delete entries where the file was never uploaded (Zombies)
			if len(deleteZombiesIDs) > 0 {
				_, _ = s.repo.DeleteEntries(ctx, db.ID, deleteZombiesIDs)
			}

			// 3. Fix stuck deleting (Attempt storage cleanup, then remove DB entry)
			if len(deleteStuckIDs) > 0 {
				_, _ = shared.DeleteMultipleSafe(ctx, s.repo, s.storage, db.ID, deleteStuckIDs)
			}
		}

		// Print brief summary for this step as requested
		fmt.Printf("\tSummary: %d marked ready, %d entries without files removed from DB, %d stuck in deleting state removed.\n",
			len(markReadyIDs),
			len(deleteZombiesIDs),
			len(deleteStuckIDs),
		)
	}

	return nil
}
