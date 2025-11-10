// filepath: internal/housekeeping/interfaces.go
package housekeeping

import (
	"mediahub/internal/models"
)

// StorageTX defines the storage methods required by the housekeeping service.
type StorageTX interface {
	DeleteEntryFile(dbName string, timestamp, entryID int64) error
	DeletePreviewFile(dbName string, timestamp, entryID int64) error
}

// DBTX is an interface that defines the database methods required by the housekeeping service.
// This decouples the housekeeping logic from the concrete database implementation.
type DBTX interface {
	GetDatabase(name string) (*models.Database, error) // Need this to get config and custom fields
	GetDatabaseStats(dbName string) (*models.Stats, error)
	GetEntriesOlderThan(dbName string, timestamp int64, customFields []models.CustomField) ([]models.Entry, error)
	GetOldestEntries(dbName string, limit, offset int, customFields []models.CustomField) ([]models.Entry, error)
	DeleteEntry(dbName string, id int64) error // This will handle stat updates
	UpdateDatabaseLastHkRun(name string) error
	GetDatabases() ([]models.Database, error)
}
