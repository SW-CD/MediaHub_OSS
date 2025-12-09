// filepath: internal/services/interfaces.go
package services

import (
	"context"
	"mediahub/internal/config"
	"mediahub/internal/models"
	"mediahub/internal/repository"
	"mime/multipart"
)

// Auditor defines the interface for recording security-relevant events.
type Auditor interface {
	// Log records an event.
	// ctx: context to trace request IDs (if available)
	// action: what happened (e.g., "database.create", "entry.delete")
	// actor: who did it (username)
	// resource: what was affected (e.g., "MyImageDB", "Entry:101")
	// details: structured metadata about the event
	Log(ctx context.Context, action string, actor string, resource string, details map[string]interface{})
}

// InfoService defines the interface for the info service.
type InfoService interface {
	GetInfo() models.Info
}

// UserService defines the interface for the user service.
type UserService interface {
	GetUserByUsername(username string) (*models.User, error)
	GetUserByID(id int) (*models.User, error)
	GetUsers() ([]models.User, error)
	UpdateUserPassword(username, password string) error
	CreateUser(args repository.UserCreateArgs) (*models.User, error)
	UpdateUser(id int, req models.User, newPassword *string) (*models.User, error)
	DeleteUser(id int) error
	InitializeAdminUser(cfg *config.Config) error
}

// DatabaseService defines the interface for the database service.
type DatabaseService interface {
	GetDatabase(name string) (*models.Database, error)
	GetDatabases() ([]models.Database, error)
	CreateDatabase(payload models.DatabaseCreatePayload) (*models.Database, error)
	UpdateDatabase(name string, updates models.DatabaseUpdatePayload) (*models.Database, error)
	DeleteDatabase(name string) error
}

// EntryService defines the interface for the entry service.
type EntryService interface {
	CreateEntry(dbName string, metadataStr string, file multipart.File, header *multipart.FileHeader) (interface{}, int, error)
	DeleteEntry(dbName string, id int64) error
	UpdateEntry(dbName string, id int64, updates models.Entry) (models.Entry, error)
	GetEntryFile(dbName string, id int64) (string, string, string, error)
	GetEntryPreview(dbName string, id int64) (string, error)
	GetEntry(dbName string, id int64, customFields []models.CustomField) (models.Entry, error)
	GetEntries(dbName string, limit, offset int, order string, tstart, tend int64, customFields []models.CustomField) ([]models.Entry, error)
	SearchEntries(dbName string, req *models.SearchRequest, customFields []models.CustomField) ([]models.Entry, error)
}

// HousekeepingService defines the interface for the housekeeping service.
type HousekeepingService interface {
	Start()
	Stop()
	TriggerHousekeeping(dbName string) (*models.HousekeepingReport, error)
}
