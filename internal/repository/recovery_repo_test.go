// filepath: internal/repository/recovery_repo_test.go
package repository

import (
	"encoding/json"
	"mediahub/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixZombieEntries(t *testing.T) {
	service, cleanup := setupTestDB(t)
	defer cleanup()

	// 1. Create a Test Database
	dbModel := &models.Database{
		Name:         "RecoveryTestDB",
		ContentType:  "image",
		Config:       json.RawMessage("{}"),
		CustomFields: []models.CustomField{},
	}
	_, err := service.CreateDatabase(dbModel)
	assert.NoError(t, err)

	// 2. Insert Entries with specific statuses manually (bypassing service layer constraints)
	// We use raw SQL to force the status to 'processing'
	tableName := "entries_RecoveryTestDB"

	// Entry 1: Stuck in processing (Zombie)
	_, err = service.DB.Exec("INSERT INTO \""+tableName+"\" (timestamp, filesize, filename, status, width, height, mime_type) VALUES (?, ?, ?, ?, ?, ?, ?)",
		1001, 1024, "zombie.jpg", "processing", 100, 100, "image/jpeg")
	assert.NoError(t, err)

	// Entry 2: Normal ready entry
	_, err = service.DB.Exec("INSERT INTO \""+tableName+"\" (timestamp, filesize, filename, status, width, height, mime_type) VALUES (?, ?, ?, ?, ?, ?, ?)",
		1002, 1024, "ready.jpg", "ready", 100, 100, "image/jpeg")
	assert.NoError(t, err)

	// 3. Run Recovery
	count, err := service.FixZombieEntries()
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Should have fixed exactly 1 entry")

	// 4. Verify Database State
	var status string
	// Check the zombie
	err = service.DB.QueryRow("SELECT status FROM \""+tableName+"\" WHERE filename = ?", "zombie.jpg").Scan(&status)
	assert.NoError(t, err)
	assert.Equal(t, "error", status, "Zombie entry should be marked as error")

	// Check the healthy entry
	err = service.DB.QueryRow("SELECT status FROM \""+tableName+"\" WHERE filename = ?", "ready.jpg").Scan(&status)
	assert.NoError(t, err)
	assert.Equal(t, "ready", status, "Ready entry should remain ready")
}
