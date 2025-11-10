// filepath: internal/services/interfaces.go
package services

import (
	"mediahub/internal/config"
	"mediahub/internal/models"
	"mediahub/internal/repository"
	"mime/multipart"
)

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
	// --- MOVED: GetEntry, GetEntries, and SearchEntries have been moved to EntryService ---
}

// EntryService defines the interface for the entry service.
type EntryService interface {
	// --- UPDATED SIGNATURE ---
	CreateEntry(dbName string, metadataStr string, file multipart.File, header *multipart.FileHeader) (interface{}, int, error)
	// --- END UPDATE ---

	DeleteEntry(dbName string, id int64) error
	UpdateEntry(dbName string, id int64, updates models.Entry) (models.Entry, error)
	GetEntryFile(dbName string, id int64) (string, string, string, error)
	GetEntryPreview(dbName string, id int64) (string, error)

	// --- ADDED from DatabaseService ---
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
