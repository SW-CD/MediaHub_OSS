package housekeeping

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared"
	"mediahub_oss/internal/shared/customerrors"
	"mediahub_oss/internal/storage"
)

// HouseKeeper manages both scheduled and manual housekeeping tasks.
type HouseKeeper struct {
	Repo           repository.Repository
	Storage        storage.StorageProvider
	Logger         *slog.Logger
	InstanceID     string // Unique identifier for the pod/node
	AuditRetention time.Duration
}

// NewHouseKeeper creates a new Housekeeping Service.
func NewHouseKeeper(repo repository.Repository, storage storage.StorageProvider, logger *slog.Logger, auditRetention time.Duration) *HouseKeeper {
	// Use the hostname (Pod name in K8s) as the base instance ID.
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-host"
	}
	// Append startup timestamp to ensure uniqueness even if a pod restarts rapidly
	instanceID := fmt.Sprintf("%s-%d", hostname, time.Now().Unix())

	return &HouseKeeper{
		Repo:           repo,
		Storage:        storage,
		Logger:         logger,
		InstanceID:     instanceID,
		AuditRetention: auditRetention,
	}
}

// StartScheduler launches a background goroutine that periodically checks all databases
// to see if their housekeeping interval has passed.
func (s *HouseKeeper) StartScheduler(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes

	go func() {
		for {
			select {
			case <-ctx.Done():
				s.Logger.Info("Stopping housekeeping scheduler")
				ticker.Stop()
				return
			case <-ticker.C:
				s.runGlobalTasks(ctx)
				s.runDBTasks(ctx)
			}
		}
	}()
}

// runGlobalTasks handles maintenance that is not tied to a specific media database.
func (s *HouseKeeper) runGlobalTasks(ctx context.Context) {
	lockName := "global_tasks"

	// Attempt to acquire the lock for 5 minutes
	acquired, err := s.Repo.AcquireLock(ctx, lockName, s.InstanceID, 5*time.Minute)
	if err != nil {
		s.Logger.Error("Failed to acquire lock for global tasks", "error", err)
		return
	}
	if !acquired {
		s.Logger.Debug("Global tasks are already running on another instance")
		return
	}

	// Ensure the lock is released when we finish
	defer func() {
		if err := s.Repo.ReleaseLock(ctx, lockName, s.InstanceID); err != nil {
			s.Logger.Error("Failed to release global tasks lock", "error", err)
		}
	}()

	// Execute global maintenance

	// 1. Clean up expired refresh tokens
	deletedCount, err := s.Repo.DeleteExpiredRefreshTokens(ctx)
	if err != nil {
		s.Logger.Error("Failed to clean up expired refresh tokens", "error", err)
	} else if deletedCount > 0 {
		s.Logger.Info("Cleaned up expired refresh tokens", "deleted_count", deletedCount)
	}

	// 2. Clean up old audit logs
	if err := s.Repo.DeleteLogs(ctx, s.AuditRetention); err != nil {
		s.Logger.Error("Failed to clean up old audit logs", "error", err)
	} else {
		s.Logger.Debug("Audit log cleanup routine executed successfully")
	}
}

func (s *HouseKeeper) runDBTasks(ctx context.Context) {
	// Fetch ONLY the databases that need housekeeping, relying on the DB server's clock.
	reqDbs, err := s.Repo.HouseKeepingRequired(ctx) //
	if err != nil {
		s.Logger.Error("Housekeeping scheduler failed to fetch required databases", "error", err)
		return
	}

	for _, db := range reqDbs {
		s.Logger.Debug("Triggering scheduled housekeeping", "database_id", db.ID, "database_name", db.Name)

		// Run synchronously to avoid spiking CPU/Disk I/O with concurrent sweeps
		_, _, err := s.RunDBHousekeeping(ctx, db)
		if err != nil {
			if errors.Is(err, customerrors.ErrLockNotAcquired) {
				s.Logger.Debug("Skipping scheduled housekeeping; locked by another instance", "database_id", db.ID, "database_name", db.Name)
			} else {
				s.Logger.Error("Scheduled housekeeping failed", "database_id", db.ID, "database_name", db.Name, "error", err)
			}
		}
	}
}

// RunDBHousekeeping executes the cleanup logic for a single database.
// This can be called by the scheduler or manually via the API.
func (s *HouseKeeper) RunDBHousekeeping(ctx context.Context, db repository.Database) (int, uint64, error) {
	var lockName = "hk_" + db.ID
	var totalDeleted int = 0
	var totalFreed uint64 = 0
	var err error

	// 1. Acquire Distributed Lock (30-minute TTL as a safety net for large deletions)
	acquired, err := s.Repo.AcquireLock(ctx, lockName, s.InstanceID, 30*time.Minute)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to check lock status: %w", err)
	}
	if !acquired {
		return 0, 0, customerrors.ErrLockNotAcquired
	}

	// Ensure lock is released regardless of panics or errors
	defer func() {
		if err := s.Repo.ReleaseLock(ctx, lockName, s.InstanceID); err != nil {
			s.Logger.Error("Failed to release lock after housekeeping", "database", db.Name, "error", err)
		}
	}()

	// If MaxAge is 0, this check is disabled.
	if db.Housekeeping.MaxAge > 0 {
		maxAgeDur := db.Housekeeping.MaxAge

		// 1. Fetch the unified database time
		dbTime, err := s.Repo.GetDBTime(ctx)
		if err != nil {
			s.Logger.Error("Housekeeper failed to get DB time for MaxAge cutoff", "error", err, "database", db.Name)
			return totalDeleted, totalFreed, err // Or handle gracefully depending on your preference
		}

		// 2. Calculate cutoff using DB time and the MaxAge duration
		cutoff := dbTime.Add(-maxAgeDur)

		for {
			// We process in batches of 100 to prevent memory spikes.
			entries, err := s.Repo.GetEntries(ctx, db.ID, 100, 0, "asc", time.Time{}, cutoff)
			if err != nil {
				s.Logger.Error("Housekeeper failed to fetch entries for MaxAge", "error", err, "database_id", db.ID, "database_name", db.Name)
				break
			}

			// If no more entries are returned, we've deleted all old files!
			if len(entries) == 0 {
				break
			}

			delCount, freed, err := s.deleteEntriesBatch(ctx, db.ID, entries)
			totalDeleted += delCount
			totalFreed += freed

			if err != nil {
				s.Logger.Error("Housekeeper failed during MaxAge batch deletion", "error", err, "database_id", db.ID, "database_name", db.Name)
				break
			}
		}
	}

	// If DiskSpace is 0, this check is disabled.
	if db.Housekeeping.DiskSpace > 0 {
		// Calculate current space using the initial stats minus what we just freed
		currentSpace := db.Stats.TotalDiskSpaceBytes - totalFreed
		limit := db.Housekeeping.DiskSpace

		for currentSpace > limit {
			// Fetch the absolute oldest entries in the DB, regardless of age
			entries, err := s.Repo.GetEntries(ctx, db.ID, 100, 0, "asc", time.Time{}, time.Time{})
			if err != nil || len(entries) == 0 {
				break // Cannot fetch or no entries left
			}

			// Accumulate just enough entries to dip below the limit
			var slideEnd int = 0
			var targetSpaceToFree uint64

			for i, e := range entries {
				targetSpaceToFree += e.Size
				slideEnd = i + 1

				// Check if this entry pushes us under the limit
				if currentSpace-targetSpaceToFree <= limit {
					break
				}
			}

			delCount, freed, err := s.deleteEntriesBatch(ctx, db.ID, entries[:slideEnd])
			totalDeleted += delCount
			totalFreed += freed
			currentSpace -= freed // Update our running total to know when to stop

			if err != nil {
				s.Logger.Error("Housekeeper failed during DiskSpace batch deletion", "error", err, "database_id", db.ID, "database_name", db.Name)
				break
			}
		}
	}

	// Update LastHkRun utilizing the new atomic database method to prevent stat overwrites
	_, err = s.Repo.HouseKeepingWasCalled(ctx, db.ID)
	if err != nil {
		s.Logger.Error("Housekeeper failed to update LastHkRun", "error", err, "database_id", db.ID, "database_name", db.Name)
	}

	s.Logger.Info("Housekeeping completed", "database_id", db.ID, "database_name", db.Name, "deleted", totalDeleted, "freed_bytes", totalFreed)
	return totalDeleted, totalFreed, nil
}

// deleteEntriesBatch safely deletes a batch of entries from the DB and storage using a 2-Phase approach.
// returns
// - number of files deleted
// - disk space that was freed
// - error if any
func (s *HouseKeeper) deleteEntriesBatch(ctx context.Context, dbID string, entries []repository.Entry) (int, uint64, error) {
	if len(entries) == 0 {
		return 0, 0, nil
	}

	// 1. Extract IDs
	ids := make([]int64, len(entries))
	for i, e := range entries {
		ids[i] = e.ID
	}

	// 2. Delete the files and entries
	deletedMeta, err := shared.DeleteMultipleSafe(ctx, s.Repo, s.Storage, dbID, ids)

	// 3. Calculate disk space freed
	var freed uint64 = 0
	for _, e := range deletedMeta {
		freed += e.Filesize + e.PreviewSize
	}

	return len(deletedMeta), freed, err
}
