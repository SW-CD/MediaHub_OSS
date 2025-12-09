// filepath: internal/repository/repository_test.go
package repository

import (
	"encoding/json"
	"mediahub/internal/config"
	"mediahub/internal/db/migrations" // Import embedded migrations
	"mediahub/internal/models"
	"os"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
)

// applyTestMigrations applies all embedded migrations to the test DB.
func applyTestMigrations(t *testing.T, repo *Repository) {
	t.Helper()
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("Failed to set goose dialect: %v", err)
	}
	// "Up" applies all available migrations
	if err := goose.Up(repo.DB, "."); err != nil {
		t.Fatalf("Failed to apply test migrations: %v", err)
	}
}

func setupTestDB(t *testing.T) (*Repository, func()) {
	t.Helper()
	const dbPath = "test_service.db"
	const storageRoot = "test_service_storage"

	// Clean up prior runs
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

	// --- CRITICAL FIX: Apply Migrations ---
	applyTestMigrations(t, repo)

	cleanup := func() {
		repo.Close()
		os.Remove(dbPath)
		os.RemoveAll(storageRoot)
	}

	return repo, cleanup
}

// TestNewRepository tests the creation of a new database service.
func TestNewRepository(t *testing.T) {
	service, cleanup := setupTestDB(t)
	defer cleanup()

	// Verify that the tables were created by the migration
	tables := []string{"databases", "users"}
	for _, table := range tables {
		var name string
		err := service.DB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("Table '%s' was not created: %v", table, err)
		}
	}
}

// TestDatabaseCRUD tests the CRUD operations for databases.
func TestDatabaseCRUD(t *testing.T) {
	service, cleanup := setupTestDB(t)
	defer cleanup()

	// Create
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

	// Verify that the entry table was created
	var tableName string
	err = service.DB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", "entries_TestDB").Scan(&tableName)
	assert.NoError(t, err, "Entry table 'entries_TestDB' was not created")

	// Read
	readDB, err := service.GetDatabase("TestDB")
	assert.NoError(t, err)
	assert.Equal(t, db.Name, readDB.Name)
	assert.Len(t, readDB.CustomFields, 2)

	// Update
	readDB.Housekeeping.Interval = "2h"
	err = service.UpdateDatabase(readDB)
	assert.NoError(t, err)

	updatedDB, err := service.GetDatabase("TestDB")
	assert.NoError(t, err)
	assert.Equal(t, "2h", updatedDB.Housekeeping.Interval)

	// Delete
	err = service.DeleteDatabase("TestDB")
	assert.NoError(t, err)

	_, err = service.GetDatabase("TestDB")
	assert.Error(t, err, "Expected database to be deleted, but it still exists")

	// Verify that the entry table was dropped
	err = service.DB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", "entries_TestDB").Scan(&tableName)
	assert.Error(t, err, "Entry table 'entries_TestDB' was not dropped")
}

// TestEntryCRUD tests the CRUD operations for entries.
func TestEntryCRUD(t *testing.T) {
	service, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a database first
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

	// Create Entry
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
	assert.Equal(t, "test_image.png", createdEntry["filename"])

	// Read Entry
	readEntry, err := service.GetEntry(db.Name, id, db.CustomFields)
	assert.NoError(t, err)
	assert.Equal(t, "A test entry", readEntry["description"])
	assert.Equal(t, 0.95, readEntry["ml_score"])
	assert.Equal(t, "test_image.png", readEntry["filename"])

	// Update Entry
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
	assert.Equal(t, 0.98, updatedEntry["ml_score"])
	assert.Equal(t, "updated_image.png", updatedEntry["filename"])

	// Get Database Entries (simple, no filter)
	entries, err := service.GetEntries(db.Name, 10, 0, "desc", 0, 0, db.CustomFields)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)

	// Delete Entry
	err = service.DeleteEntry(db.Name, id)
	assert.NoError(t, err)

	_, err = service.GetEntry(db.Name, id, db.CustomFields)
	assert.Error(t, err, "Expected entry to be deleted")
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
	assert.NoError(t, err, "Index 'idx_entries_IndexTestDB_indexed_field' was not created")
}

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

	err = tx.Commit()
	assert.NoError(t, err)

	return createdEntry
}
