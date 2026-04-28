package recovery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared/customerrors"
	"mediahub_oss/internal/storage"
)

// IntegrityCheck ensures that database records and physical files match.
func (s *RecoveryService) IntegrityCheck(ctx context.Context) error {
	databases, err := s.repo.GetDatabases(ctx)
	if err != nil {
		return fmt.Errorf("could not fetch databases: %w", err)
	}

	for _, db := range databases {
		fmt.Printf("Database %q\n", db.Name)
		stats := db.Stats

		// --- PHASE 2.1: DB -> Storage (Find missing files) ---
		missingFileIDs, entryCount, err := s.checkMissingFiles(ctx, db)
		if err != nil {
			return err
		}

		// --- PHASE 2.2: Storage -> DB (Find orphan main files & calculate sizes) ---
		var calculatedTotalBytes uint64 = 0
		orphanFileIDs, calculatedTotalBytes, err := s.checkOrphanFiles(ctx, db, stats)
		if err != nil {
			fmt.Println() // Break the \r progress line
			s.logger.Error("Failed walking main storage", "database_id", db.ID, "database_name", db.Name, "error", err)
			return err // Abort on error to prevent saving corrupted stats
		}

		// --- PHASE 2.3: Storage -> DB (Find orphan preview files) ---
		orphanPreviewIDs, calculatedPreviewBytes, err := s.checkOrphanPreviewFiles(ctx, db, stats)
		calculatedTotalBytes += calculatedPreviewBytes
		if err != nil {
			fmt.Println() // Break the \r progress line
			s.logger.Error("Failed walking preview storage", "database_id", db.ID, "database_name", db.Name, "error", err)
			return err // Abort on error to prevent saving corrupted stats
		}

		// --- PHASE 2.4: entry DB --> database DB (verify entry count stat)
		if entryCount != db.Stats.EntryCount {
			fmt.Printf("Found discrepance in entry count. Stats: %v, Database: %v\n", db.Stats.EntryCount, entryCount)
		}

		fmt.Printf("\r- Step 2: Integrity check: 100%% (Applying DB<->Disk fixes)            \n")

		// --- PHASE 2.5: Action ---
		if !s.dryRun {
			// Delete the database entries for files that no longer exist on disk
			if len(missingFileIDs) > 0 {
				_, _ = s.storage.DeleteMultiplePreviews(ctx, db.ID, missingFileIDs)
				_, _ = s.repo.DeleteEntries(ctx, db.ID, missingFileIDs)
			}

			// Delete physical files that have no database record
			if len(orphanFileIDs) > 0 {
				_, _ = s.storage.DeleteMultiple(ctx, db.ID, orphanFileIDs)
			}

			// Delete physical preview files that have no database record
			if len(orphanPreviewIDs) > 0 {
				_, _ = s.storage.DeleteMultiplePreviews(ctx, db.ID, orphanPreviewIDs)
			}

			// --- PHASE 2.5: Sync Statistics ---
			trueEntryCount := entryCount - uint64(len(missingFileIDs))

			// Remove the flawed comparison. Always apply the freshly calculated,
			// mathematically correct stats to fix any mutations caused by DeleteEntries.
			db.Stats.EntryCount = trueEntryCount
			db.Stats.TotalDiskSpaceBytes = calculatedTotalBytes

			// Save it back
			_, err = s.repo.UpdateDatabase(ctx, db)
			if err != nil {
				s.logger.Error("Failed to sync database stats", "database_id", db.ID, "database_name", db.Name, "error", err)
			}
		}

		// Print the final summary, including stats differences
		fmt.Printf("\tSummary: %d missing files removed from DB, %d orphan media files and %d orphan previews removed from disk.\n",
			len(missingFileIDs), len(orphanFileIDs), len(orphanPreviewIDs))
	}

	return nil
}

// Loop over entries and check if a file exists in the storage. Return entry IDs without a file, as well as total entry count.
func (s *RecoveryService) checkMissingFiles(ctx context.Context, db repository.Database) ([]int64, uint64, error) {

	var missingFileIDs []int64
	var totalEntries uint64 = 0
	var expectedEntryCount = db.Stats.EntryCount

	// avoid division by 0
	if expectedEntryCount == 0 {
		expectedEntryCount = 1
	}

	limit := 1000
	processedDB := uint64(0)

	// DB -> Storage (Find missing files)
	for offset := 0; true; offset += limit {
		entries, err := s.repo.GetEntries(ctx, db.ID, limit, offset, "id asc", time.Time{}, time.Time{})
		if err != nil {
			return missingFileIDs, totalEntries, fmt.Errorf("failed to fetch entries for %s: %w", db.Name, err)
		}
		totalEntries += uint64(len(entries))

		// stop if no more entries returned (offset too large)
		if len(entries) == 0 {
			break
		}

		for _, entry := range entries {
			processedDB++
			percent := (processedDB * 100) / expectedEntryCount
			fmt.Printf("\r- Step 2: Integrity check: %d%% (Scanning DB->Disk...)", percent)

			// Only verify "ready" files, as Step 1 already handled "processing" and "deleting"
			if entry.Status == repository.EntryStatusReady {
				info, err := s.storage.Stat(ctx, db.ID, entry.ID)

				if errors.Is(err, customerrors.ErrNotFound) {
					missingFileIDs = append(missingFileIDs, entry.ID)
				} else if err != nil {
					fmt.Println() // Break the \r progress line
					s.logger.Error("Storage stat failed", "database_id", db.ID, "database_name", db.Name, "id", entry.ID, "error", err)
				} else {
					// File exists! Cross-check the recorded size against the actual physical size
					if entry.Size != uint64(info.Size) {
						fmt.Println() // Break the \r progress line
						s.logger.Warn("File size mismatch detected (Possible corruption)",
							"database_id", db.ID,
							"database_name", db.Name,
							"id", entry.ID,
							"repo_size_bytes", entry.Size,
							"storage_size_bytes", info.Size,
						)
					}
				}
			}
		}
	}
	return missingFileIDs, totalEntries, nil
}

// walk over files in the storage and check if a database entry exists. Return file ids without entry,
// the accumulated file sizes and possible errors.
func (s *RecoveryService) checkOrphanFiles(ctx context.Context, db repository.Database, stats repository.DatabaseStats) ([]int64, uint64, error) {
	var orphanFileIDs []int64
	var calculatedTotalBytes uint64 = 0

	// number of entries for calculating progress
	divisor := stats.EntryCount
	if divisor == 0 {
		divisor = 1
	}

	processedStorage := uint64(0)
	err := s.storage.Walk(ctx, db.ID, func(id int64, info storage.FileInfo) error {
		processedStorage++
		percent := (processedStorage * 100) / divisor
		if percent > 99 {
			percent = 99
		} // Cap at 99% until finished
		fmt.Printf("\r- Step 2: Integrity check: %d%% (Scanning Disk->DB Main)...", percent)

		// Check if the file exists in the database
		_, err := s.repo.GetEntry(ctx, db.ID, id)
		if errors.Is(err, customerrors.ErrNotFound) {
			orphanFileIDs = append(orphanFileIDs, id)
		} else if err != nil {
			fmt.Println() // Break the \r progress line
			s.logger.Error("Database lookup failed during main file walk", "database_id", db.ID, "database_name", db.Name, "id", id, "error", err)
			// Return the error to abort! If we ignore it, calculatedTotalBytes
			// will be artificially low and corrupt the database stats.
			return err
		} else {
			// Valid file! Add to our true physical size calculation
			calculatedTotalBytes += uint64(info.Size)
		}
		return nil
	})
	return orphanFileIDs, calculatedTotalBytes, err
}

func (s *RecoveryService) checkOrphanPreviewFiles(ctx context.Context, db repository.Database, stats repository.DatabaseStats) ([]int64, uint64, error) {
	var orphanFileIDs []int64
	var calculatedTotalBytes uint64 = 0

	// number of entries for calculating progress
	divisor := stats.EntryCount
	if divisor == 0 {
		divisor = 1
	}

	processedStorage := uint64(0)
	err := s.storage.WalkPreview(ctx, db.ID, func(id int64, info storage.FileInfo) error {
		processedStorage++
		percent := (processedStorage * 100) / divisor
		if percent > 99 {
			percent = 99
		} // Cap at 99% until finished
		fmt.Printf("\r- Step 2: Integrity check: %d%% (Scanning Disk->DB Previews)...", percent)

		// Check if the file exists in the database
		_, err := s.repo.GetEntry(ctx, db.ID, id)
		if errors.Is(err, customerrors.ErrNotFound) {
			orphanFileIDs = append(orphanFileIDs, id)
		} else if err != nil {
			fmt.Println() // Break the \r progress line
			s.logger.Error("Database lookup failed during preview file walk", "database_id", db.ID, "database_name", db.Name, "id", id, "error", err)
			// Return the error to abort!
			return err
		} else {
			// Valid file! Add to our true physical size calculation
			calculatedTotalBytes += uint64(info.Size)
		}
		return nil
	})
	return orphanFileIDs, calculatedTotalBytes, err
}
