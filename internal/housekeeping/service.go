// filepath: internal/housekeeping/service.go
package housekeeping

import (
	"math"
	"mediahub/internal/logging"
	"time"
)

const (
	// DefaultCheckInterval is used when no databases are found or have valid schedules.
	DefaultCheckInterval = 1 * time.Hour
	// MinCheckInterval is the minimum time between checks to prevent busy-looping.
	MinCheckInterval = 1 * time.Minute
)

// Service provides the background worker for automated housekeeping.
type Service struct {
	Deps   Dependencies // Now includes DB and Storage
	timer  *time.Timer
	stopCh chan struct{}
}

// NewService creates a new housekeeping service instance.
func NewService(deps Dependencies) *Service {
	return &Service{
		Deps:   deps,
		stopCh: make(chan struct{}),
	}
}

// Start kicks off the background housekeeping service.
func (s *Service) Start() {
	logging.Log.Info("Starting background housekeeping service.")
	s.timer = time.NewTimer(0) // Fire immediately on start

	go func() {
		for {
			select {
			case <-s.timer.C:
				s.runChecks()
				nextRun := s.scheduleNextRun()
				s.timer.Reset(nextRun)
				logging.Log.Infof("Next housekeeping check scheduled in %v.", nextRun)
			case <-s.stopCh:
				s.timer.Stop()
				return
			}
		}
	}()
}

// Stop terminates the background housekeeping service.
func (s *Service) Stop() {
	logging.Log.Info("Stopping background housekeeping service.")
	close(s.stopCh)
}

// scheduleNextRun calculates the duration until the next housekeeping event.
func (s *Service) scheduleNextRun() time.Duration {
	databases, err := s.Deps.DB.GetDatabases()
	if err != nil {
		logging.Log.Errorf("Housekeeping could not retrieve databases to schedule next run: %v", err)
		return DefaultCheckInterval
	}

	if len(databases) == 0 {
		return DefaultCheckInterval
	}

	minDuration := time.Duration(math.MaxInt64)
	now := time.Now()

	for _, db := range databases {
		interval, err := parseDuration(db.Housekeeping.Interval)
		if err != nil || interval == 0 {
			continue
		}

		nextRun := db.LastHkRun.Add(interval)
		duration := nextRun.Sub(now)

		if duration < minDuration {
			minDuration = duration
		}
	}

	if minDuration == time.Duration(math.MaxInt64) {
		return DefaultCheckInterval
	}

	if minDuration < MinCheckInterval {
		return MinCheckInterval
	}

	return minDuration
}

// runChecks fetches all databases and runs housekeeping for those whose interval has elapsed.
func (s *Service) runChecks() {
	logging.Log.Debug("Housekeeping service: Checking databases...")
	databases, err := s.Deps.DB.GetDatabases()
	if err != nil {
		logging.Log.Errorf("Housekeeping service could not retrieve databases: %v", err)
		return
	}

	now := time.Now()
	for _, db := range databases {
		interval, err := parseDuration(db.Housekeeping.Interval)
		if err != nil {
			logging.Log.Warnf("Skipping housekeeping for '%s' due to invalid interval: %v", db.Name, err)
			continue
		}
		if interval == 0 {
			continue // Skip if interval is zero
		}

		// Check if the next run time is in the past
		if db.LastHkRun.Add(interval).Before(now) {
			logging.Log.Infof("Housekeeping interval elapsed for '%s'. Running cleanup...", db.Name)

			// Pass the service's dependencies to the task runner
			report, err := RunForDatabase(s.Deps, db.Name)
			if err != nil {
				logging.Log.Errorf("Housekeeping run failed for '%s': %v", db.Name, err)
			} else {
				logging.Log.Infof("Housekeeping run finished for '%s': %s", db.Name, report.Message)
			}

			// Update the last run time to now
			if err := s.Deps.DB.UpdateDatabaseLastHkRun(db.Name); err != nil {
				logging.Log.Errorf("Failed to update last housekeeping run time for '%s': %v", db.Name, err)
			}
		}
	}
}
