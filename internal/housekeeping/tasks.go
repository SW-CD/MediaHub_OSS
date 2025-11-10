// filepath: internal/housekeeping/tasks.go
package housekeeping

import (
	"fmt"
	"mediahub/internal/logging"
	"mediahub/internal/models"
	"time"
)

// Dependencies defines the required services for the housekeeping tasks.
type Dependencies struct {
	DB      DBTX
	Storage StorageTX
}

// RunForDatabase executes the housekeeping tasks for a specific database.
func RunForDatabase(deps Dependencies, dbName string) (*models.HousekeepingReport, error) {
	db, err := deps.DB.GetDatabase(dbName)
	if err != nil {
		return nil, fmt.Errorf("database not found: %w", err)
	}

	report := &models.HousekeepingReport{
		DatabaseName: dbName,
	}

	// 1. Cleanup by Age
	ageReport, err := cleanupByAge(deps, db)
	if err != nil {
		logging.Log.Errorf("Housekeeping cleanup by age failed for %s: %v", dbName, err)
	}
	if ageReport != nil {
		report.EntriesDeleted += ageReport.EntriesDeleted
		report.SpaceFreedBytes += ageReport.SpaceFreedBytes
	}

	// 2. Cleanup by Disk Space
	spaceReport, err := cleanupByDiskSpace(deps, db)
	if err != nil {
		logging.Log.Errorf("Housekeeping cleanup by disk space failed for %s: %v", dbName, err)
	}
	if spaceReport != nil {
		report.EntriesDeleted += spaceReport.EntriesDeleted
		report.SpaceFreedBytes += spaceReport.SpaceFreedBytes
	}

	report.Message = fmt.Sprintf("Housekeeping complete for '%s'. %d entries deleted, freeing %s.",
		dbName, report.EntriesDeleted, formatBytes(report.SpaceFreedBytes))

	return report, nil
}

// cleanupByAge deletes entries older than the max_age rule.
func cleanupByAge(deps Dependencies, db *models.Database) (*models.HousekeepingReport, error) {
	maxAgeDuration, err := parseDuration(db.Housekeeping.MaxAge)
	if err != nil {
		return nil, fmt.Errorf("invalid max_age format: %w", err)
	}

	// If duration is 0, skip this check
	if maxAgeDuration == 0 {
		logging.Log.Debugf("Housekeeping cleanup by age is disabled for '%s' (max_age is 0).", db.Name)
		return &models.HousekeepingReport{}, nil
	}

	cutoffTime := time.Now().Add(-maxAgeDuration)
	cutoffTimestamp := cutoffTime.Unix()

	entriesToDelete, err := deps.DB.GetEntriesOlderThan(db.Name, cutoffTimestamp, db.CustomFields)
	if err != nil {
		return nil, fmt.Errorf("could not query for old entries: %w", err)
	}

	if len(entriesToDelete) == 0 {
		return &models.HousekeepingReport{}, nil
	}

	logging.Log.Infof("Found %d entries older than %s for database '%s'. Deleting...", len(entriesToDelete), db.Housekeeping.MaxAge, db.Name)
	return deleteEntries(deps, db, entriesToDelete)
}

// cleanupByDiskSpace deletes the oldest entries if the total disk space exceeds the limit.
func cleanupByDiskSpace(deps Dependencies, db *models.Database) (*models.HousekeepingReport, error) {
	maxDiskSpace, err := parseSize(db.Housekeeping.DiskSpace)
	if err != nil {
		return nil, fmt.Errorf("invalid disk_space format: %w", err)
	}

	// If max disk space is 0, skip this check
	if maxDiskSpace == 0 {
		logging.Log.Debugf("Housekeeping cleanup by disk space is disabled for '%s' (disk_space is 0).", db.Name)
		return &models.HousekeepingReport{}, nil
	}

	stats, err := deps.DB.GetDatabaseStats(db.Name)
	if err != nil {
		return nil, fmt.Errorf("could not get database stats: %w", err)
	}

	bytesToFree := stats.TotalDiskSpaceBytes - maxDiskSpace
	if bytesToFree <= 0 {
		logging.Log.Debugf("Disk space for '%s' is within limits. No cleanup needed.", db.Name)
		return &models.HousekeepingReport{}, nil
	}

	logging.Log.Infof("Database '%s' is over disk space limit by %s. Finding entries to delete...", db.Name, formatBytes(bytesToFree))

	var entriesToDelete []models.Entry
	var spaceFound int64
	offset := 0
	const batchSize = 100

	// Fetch entries in batches until we have enough to free the required space.
	for spaceFound < bytesToFree {
		batch, err := deps.DB.GetOldestEntries(db.Name, batchSize, offset, db.CustomFields)
		if err != nil || len(batch) == 0 {
			// Stop if we run out of entries or encounter a database error.
			logging.Log.Warnf("Stopping disk space cleanup for '%s'; no more entries to delete.", db.Name)
			break
		}

		for _, entry := range batch {
			entriesToDelete = append(entriesToDelete, entry)
			if filesize, ok := entry["filesize"].(int64); ok {
				spaceFound += filesize
			}
			if spaceFound >= bytesToFree {
				break
			}
		}
		offset += len(batch) // Use len(batch) in case the last page is smaller than batchSize
	}

	if len(entriesToDelete) > 0 {
		logging.Log.Infof("Found %d oldest entries in '%s' to free %s. Deleting...", len(entriesToDelete), db.Name, formatBytes(spaceFound))
		return deleteEntries(deps, db, entriesToDelete)
	}

	return &models.HousekeepingReport{}, nil
}

// deleteEntries is a helper function to delete a list of entries and return a report.
func deleteEntries(deps Dependencies, db *models.Database, entries []models.Entry) (*models.HousekeepingReport, error) {
	report := &models.HousekeepingReport{DatabaseName: db.Name}

	for _, entry := range entries {
		id, ok := entry["id"].(int64)
		if !ok {
			continue
		}
		timestamp, ok := entry["timestamp"].(int64)
		if !ok {
			continue
		}
		filesize, ok := entry["filesize"].(int64)
		if !ok {
			continue
		}

		// Delete the file from storage using the Storage service
		if err := deps.Storage.DeleteEntryFile(db.Name, timestamp, id); err != nil {
			logging.Log.Warnf("Housekeeping: Failed to delete entry file: %v", err)
		}

		// Delete the preview file using the Storage service
		if err := deps.Storage.DeletePreviewFile(db.Name, timestamp, id); err != nil {
			logging.Log.Warnf("Housekeeping: Failed to delete preview file: %v", err)
		}

		// Delete the record from the database
		// This single call now handles DB deletion AND stat updates.
		if err := deps.DB.DeleteEntry(db.Name, id); err != nil {
			logging.Log.Errorf("Failed to delete entry record %d from %s: %v", id, db.Name, err)
			continue // Skip to the next entry
		}

		report.EntriesDeleted++
		report.SpaceFreedBytes += filesize
	}

	return report, nil
}
