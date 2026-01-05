// filepath: internal/api/handlers/main.go
package handlers

import (
	"mediahub/internal/config"
	"mediahub/internal/services"
	"mediahub/internal/services/auth"
	"time"
)

// Handlers provides a struct to hold shared dependencies for API handlers.
type Handlers struct {
	Info         services.InfoService
	User         services.UserService
	Token        auth.TokenService
	Database     services.DatabaseService
	Entry        services.EntryService
	Housekeeping services.HousekeepingService
	Auditor      services.Auditor // <-- Added Auditor

	Cfg       *config.Config
	Version   string
	StartTime time.Time
}

// NewHandlers creates a new instance of Handlers with its dependencies.
func NewHandlers(
	info services.InfoService,
	user services.UserService,
	token auth.TokenService,
	database services.DatabaseService,
	entry services.EntryService,
	housekeeping services.HousekeepingService,
	auditor services.Auditor, // <-- Added Auditor
	cfg *config.Config,
) *Handlers {
	return &Handlers{
		Info:         info,
		User:         user,
		Token:        token,
		Database:     database,
		Entry:        entry,
		Housekeeping: housekeeping,
		Auditor:      auditor,
		Cfg:          cfg,
		Version:      info.GetInfo().Version,
		StartTime:    info.GetInfo().UptimeSince,
	}
}
