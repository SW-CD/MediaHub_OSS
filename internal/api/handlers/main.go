// filepath: internal/api/handlers/main.go
package handlers

import (
	"mediahub/internal/config"
	"mediahub/internal/services" // <-- IMPORT SERVICES
	"time"
)

// Handlers provides a struct to hold shared dependencies for API handlers,
// such as the database service.
type Handlers struct {
	// --- Depend on interfaces, not concrete structs ---
	Info         services.InfoService
	User         services.UserService
	Database     services.DatabaseService
	Entry        services.EntryService
	Housekeeping services.HousekeepingService

	Cfg       *config.Config
	Version   string    // Kept for info handler, though it's in InfoService now
	StartTime time.Time // Kept for info handler, though it's in InfoService now
}

// NewHandlers creates a new instance of Handlers with its dependencies.
// --- Accept interfaces as parameters ---
func NewHandlers(
	info services.InfoService,
	user services.UserService,
	database services.DatabaseService,
	entry services.EntryService,
	housekeeping services.HousekeepingService,
	cfg *config.Config,
) *Handlers {
	return &Handlers{
		Info:         info,
		User:         user,
		Database:     database,
		Entry:        entry,
		Housekeeping: housekeeping,
		Cfg:          cfg,
		Version:      info.GetInfo().Version,     // Get from service
		StartTime:    info.GetInfo().UptimeSince, // <-- FIX: Changed from StartTime to UptimeSince
	}
}
