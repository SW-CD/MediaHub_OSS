// filepath: internal/repository/repository_test.go
package repository

import (
	"encoding/json"
	"mediahub/internal/config"
	"mediahub/internal/db/migrations"
	"mediahub/internal/models"
	"os"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
)

func applyTestMigrations(t *testing.T, repo *Repository) {
	t.Helper()
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("Failed to set goose dialect: %v", err)
	}
	if err := goose.Up(repo.DB, "."); err != nil {
		t.Fatalf("Failed to apply test migrations: %v", err)
	}
}

func setupTestDB(t *testing.T) (*Repository, func()) {
	t.Helper()
	const dbPath = "test_service.db"
	const storageRoot = "test_service_storage"

	os.Remove(dbPath)
	os.RemoveAll(storageRoot)
	os.MkdirAll(storageRoot, 0755)

	dummyCfg := &config.Config{
		Database: config.DatabaseConfig{
			Path:        dbPath,
			StorageRoot: storageRoot,
		},
	}

	repo, err := NewRepository(dummyCfg)
	if err != nil {
		t.Fatalf("Failed to create new repository: %v", err)
	}

	applyTestMigrations(t, repo)

	cleanup := func() {
		repo.Close()
		os.Remove(dbPath)
		os.RemoveAll(storageRoot)
	}

	return repo, cleanup
}

func TestNewRepository(t *testing.T) {
	service, cleanup := setupTestDB(t)
	defer cleanup()
	tables := []string{"databases", "users"}
	for _, table := range tables {
		var name string
		err := service.DB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("Table '%s' was not created: %v", table, err)
		}
	}
}

func TestDatabaseCRUD(t *testing.T) {
	service, cleanup := setupTestDB(t)
	defer cleanup()
	db := models.Database{
		Name:        "TestDB",
		ContentType: "image",
		Config:      json.RawMessage("{}"),
		CustomFields: []models.CustomField{
			{Name: "latitude", Type: "REAL"},
			{Name: "longitude", Type: "REAL"},
		},
	}
	createdDB, err := service.CreateDatabase(&db)
	assert.NoError(t, err)
	assert.Equal(t, db.Name, createdDB.Name)
	readDB, err := service.GetDatabase("TestDB")
	assert.NoError(t, err)
	assert.Equal(t, db.Name, readDB.Name)
	readDB.Housekeeping.Interval = "2h"
	err = service.UpdateDatabase(readDB)
	assert.NoError(t, err)
	updatedDB, err := service.GetDatabase("TestDB")
	assert.NoError(t, err)
	assert.Equal(t, "2h", updatedDB.Housekeeping.Interval)
	err = service.DeleteDatabase("TestDB")
	assert.NoError(t, err)
	_, err = service.GetDatabase("TestDB")
	assert.Error(t, err)
}

func TestEntryCRUD(t *testing.T) {
	service, cleanup := setupTestDB(t)
	defer cleanup()
	db := models.Database{
		Name:        "EntryTestDB",
		ContentType: "image",
		Config:      json.RawMessage("{}"),
		CustomFields: []models.CustomField{
			{Name: "sensor_id", Type: "TEXT"},
			{Name: "ml_score", Type: "REAL"},
			{Name: "description", Type: "TEXT"},
		},
	}
	_, err := service.CreateDatabase(&db)
	assert.NoError(t, err)
	entry := models.Entry{
		"timestamp":   1234567890,
		"width":       1024,
		"height":      768,
		"filesize":    512000,
		"mime_type":   "image/png",
		"filename":    "test_image.png",
		"status":      "ready",
		"sensor_id":   "test-sensor-123",
		"ml_score":    0.95,
		"description": "A test entry",
	}
	createdEntry := createTestEntry(t, service, db.Name, entry)
	id := createdEntry["id"].(int64)
	assert.NotZero(t, id)
	readEntry, err := service.GetEntry(db.Name, id, db.CustomFields)
	assert.NoError(t, err)
	assert.Equal(t, "A test entry", readEntry["description"])
	updates := models.Entry{
		"description": "An updated test entry",
		"ml_score":    0.98,
		"filename":    "updated_image.png",
	}
	err = service.UpdateEntry(db.Name, id, updates, db.CustomFields)
	assert.NoError(t, err)
	updatedEntry, err := service.GetEntry(db.Name, id, db.CustomFields)
	assert.NoError(t, err)
	assert.Equal(t, "An updated test entry", updatedEntry["description"])
	entries, err := service.GetEntries(db.Name, 10, 0, "desc", 0, 0, db.CustomFields)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	err = service.DeleteEntry(db.Name, id)
	assert.NoError(t, err)
	_, err = service.GetEntry(db.Name, id, db.CustomFields)
	assert.Error(t, err)
}

func TestCustomFieldIndexing(t *testing.T) {
	service, cleanup := setupTestDB(t)
	defer cleanup()
	db := models.Database{
		Name:        "IndexTestDB",
		ContentType: "image",
		Config:      json.RawMessage("{}"),
		CustomFields: []models.CustomField{
			{Name: "indexed_field", Type: "TEXT"},
		},
	}
	_, err := service.CreateDatabase(&db)
	assert.NoError(t, err)
	var indexName string
	err = service.DB.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name=?", "idx_entries_IndexTestDB_indexed_field").Scan(&indexName)
	assert.NoError(t, err, "Index was not created")
}

// createTestEntry helper: inserts entry, updates row with filesize, updates DB stats.
func createTestEntry(t *testing.T, service *Repository, dbName string, entry models.Entry) models.Entry {
	t.Helper()
	db, err := service.GetDatabase(dbName)
	assert.NoError(t, err)

	tx, err := service.BeginTx()
	assert.NoError(t, err)
	defer tx.Rollback()

	if _, ok := entry["status"]; !ok {
		entry["status"] = "ready"
	}

	createdEntry, err := tx.CreateEntryInTx(dbName, db.ContentType, entry, db.CustomFields)
	assert.NoError(t, err)
	id := createdEntry["id"].(int64)

	var size int64 = 0
	// 1. Determine size from input (preferred) or output
	if val, ok := entry["filesize"]; ok {
		switch v := val.(type) {
		case int:
			size = int64(v)
		case int64:
			size = v
		case float64:
			size = int64(v)
		}
	} else if val, ok := createdEntry["filesize"]; ok {
		switch v := val.(type) {
		case int:
			size = int64(v)
		case int64:
			size = v
		}
	}

	// 2. Explicitly update the entry row if we have a non-zero size.
	// CreateEntryInTx sets filesize=0 by default.
	if size > 0 {
		err = tx.UpdateEntryInTx(dbName, id, models.Entry{"filesize": size}, db.CustomFields)
		assert.NoError(t, err)
		createdEntry["filesize"] = size // Update returned map
	}

	// 3. Update Database Stats
	err = tx.UpdateStatsInTx(dbName, 1, size)
	assert.NoError(t, err)

	err = tx.Commit()
	assert.NoError(t, err)

	return createdEntry
}
