// filepath: internal/services/entry_bulk_test.go
package services

import (
	"mediahub/internal/config"
	"mediahub/internal/db/migrations"
	"mediahub/internal/models"
	"mediahub/internal/repository"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
)

// setupIntegrationTest creates a real Repo and StorageService backed by temp files.
func setupIntegrationTest(t *testing.T) (*entryService, *repository.Repository, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "mediahub_integration_")
	assert.NoError(t, err)

	dbPath := filepath.Join(tmpDir, "test.db")
	storageRoot := filepath.Join(tmpDir, "storage")

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path:        dbPath,
			StorageRoot: storageRoot,
		},
	}

	repo, err := repository.NewRepository(cfg)
	assert.NoError(t, err)

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("Failed to set goose dialect: %v", err)
	}
	if err := goose.Up(repo.DB, "."); err != nil {
		t.Fatalf("Failed to migrate integration DB: %v", err)
	}

	storageSvc := NewStorageService(cfg)
	entrySvc := NewEntryService(repo, storageSvc, cfg)

	cleanup := func() {
		repo.Close()
		os.RemoveAll(tmpDir)
	}

	return entrySvc, repo, storageRoot, cleanup
}

func TestService_DeleteEntries(t *testing.T) {
	service, repo, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// 1. Setup Data
	dbName := "SvcBulkDelDB"
	dbSvc := NewDatabaseService(repo, NewStorageService(service.Cfg))
	_, err := dbSvc.CreateDatabase(models.DatabaseCreatePayload{
		Name:         dbName,
		ContentType:  "file",
		CustomFields: []models.CustomField{},
	})
	assert.NoError(t, err)

	// 2. Create an entry manually
	tx, _ := repo.BeginTx()
	fileSize := int64(123)
	entryMeta := models.Entry{
		"timestamp": time.Now().Unix(),
		"filesize":  fileSize,
		"filename":  "delete_test.bin",
		"mime_type": "application/octet-stream",
		"status":    "ready",
	}

	created, err := tx.CreateEntryInTx(dbName, "file", entryMeta, nil)
	assert.NoError(t, err)
	id := created["id"].(int64)
	ts := created["timestamp"].(int64)

	// --- FIX: Explicitly update the DB row's filesize ---
	// CreateEntryInTx sets filesize=0. DeleteEntries reads this from DB.
	// We must set it to 123 so the freed space calculation matches.
	err = tx.UpdateEntryInTx(dbName, id, models.Entry{"filesize": fileSize}, nil)
	assert.NoError(t, err)

	// Update stats for consistency (though DeleteEntries calculates based on row data)
	err = tx.UpdateStatsInTx(dbName, 1, fileSize)
	assert.NoError(t, err)

	err = tx.Commit()
	assert.NoError(t, err)

	// Create dummy file on disk
	entryPath, _ := service.Storage.GetEntryPath(dbName, ts, id)
	os.MkdirAll(filepath.Dir(entryPath), 0755)
	os.WriteFile(entryPath, []byte("dummy data"), 0644)

	// 3. Call Bulk Delete
	count, freed, err := service.DeleteEntries(dbName, []int64{id})

	// 4. Assertions
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, fileSize, freed)

	// Verify DB deletion
	_, err = repo.GetEntry(dbName, id, nil)
	assert.Error(t, err, "Entry should be deleted from DB")

	time.Sleep(50 * time.Millisecond)

	_, err = os.Stat(entryPath)
	assert.True(t, os.IsNotExist(err), "File should be deleted from disk")
}
