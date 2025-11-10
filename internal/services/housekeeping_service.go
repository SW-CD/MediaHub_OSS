// filepath: internal/services/housekeeping_service.go
package services

import (
	"mediahub/internal/housekeeping"
	"mediahub/internal/logging"
	"mediahub/internal/models"
	"mediahub/internal/repository"
)

// --- REFACTOR: Compile-time check to ensure interface is implemented ---
var _ HousekeepingService = (*housekeepingService)(nil)

// --- REFACTOR: Struct renamed to lowercase ---
// housekeepingService manages the lifecycle of the background housekeeping worker
// and provides a method for manual triggering.
type housekeepingService struct {
	Repo       *repository.Repository
	Storage    *StorageService
	worker     *housekeeping.Service
	workerDeps housekeeping.Dependencies
}

// NewHousekeepingService creates a new HousekeepingService.
// --- REFACTOR: Return type is the concrete struct, but it satisfies the interface ---
func NewHousekeepingService(repo *repository.Repository, storage *StorageService) *housekeepingService {
	// Create the dependencies that the background worker will use
	deps := housekeeping.Dependencies{
		DB:      repo,    // The repository satisfies the DBTX interface
		Storage: storage, // The storage service satisfies the StorageTX interface
	}

	return &housekeepingService{
		Repo:       repo,
		Storage:    storage,
		workerDeps: deps,
	}
}

// Start begins the background housekeeping worker.
func (s *housekeepingService) Start() {
	// Create the worker instance using the prepared dependencies
	s.worker = housekeeping.NewService(s.workerDeps)
	s.worker.Start()
}

// Stop terminates the background housekeeping worker.
func (s *housekeepingService) Stop() {
	if s.worker != nil {
		s.worker.Stop()
	}
}

// TriggerHousekeeping manually runs the cleanup tasks for a specific database.
func (s *housekeepingService) TriggerHousekeeping(dbName string) (*models.HousekeepingReport, error) {
	// Manually run the housekeeping tasks using the same dependencies as the worker
	report, err := housekeeping.RunForDatabase(s.workerDeps, dbName)
	if err != nil {
		return nil, err
	}

	// After a manual run, update the timestamp to reset the timer for the automatic service.
	if err := s.Repo.UpdateDatabaseLastHkRun(dbName); err != nil {
		// Log the error but don't fail the report
		logging.Log.Errorf("Failed to update last_hk_run for %s after manual trigger: %v", dbName, err)
	}
	return report, nil
}
