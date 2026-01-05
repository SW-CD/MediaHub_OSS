// filepath: internal/services/entry_bulk.go
package services

import (
	"mediahub/internal/logging"
)

// DeleteEntries performs the bulk deletion of entries.
// It deletes records from the DB and then asynchronously cleans up files from disk.
// Returns count of deleted entries and total bytes freed.
func (s *entryService) DeleteEntries(dbName string, ids []int64) (int, int64, error) {
	// 1. Transactional DB Delete
	deletedEntries, err := s.Repo.DeleteEntries(dbName, ids)
	if err != nil {
		logging.Log.Errorf("EntryService: Bulk delete failed for DB '%s': %v", dbName, err)
		return 0, 0, err
	}

	count := len(deletedEntries)
	if count == 0 {
		return 0, 0, nil
	}

	// Calculate total space freed for the report
	var totalSpaceFreed int64
	for _, meta := range deletedEntries {
		totalSpaceFreed += meta.Filesize
	}

	// 2. Async File Cleanup
	// We launch a goroutine so the API response is fast.
	// Orphaned files are better than a slow API or blocked DB.
	go func() {
		logging.Log.Infof("EntryService: Starting async file cleanup for %d entries in '%s'", count, dbName)
		for _, meta := range deletedEntries {
			// Delete main file
			if err := s.Storage.DeleteEntryFile(dbName, meta.Timestamp, meta.ID); err != nil {
				// Log but continue
				logging.Log.Warnf("EntryService: Failed to delete file for entry %d: %v", meta.ID, err)
			}
			// Delete preview file
			if err := s.Storage.DeletePreviewFile(dbName, meta.Timestamp, meta.ID); err != nil {
				logging.Log.Warnf("EntryService: Failed to delete preview for entry %d: %v", meta.ID, err)
			}
		}
		logging.Log.Infof("EntryService: Async file cleanup completed for '%s'", dbName)
	}()

	return count, totalSpaceFreed, nil
}
