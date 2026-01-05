// filepath: internal/repository/entry_bulk_repo_test.go
package repository

import (
	"encoding/json"
	"mediahub/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeleteEntries(t *testing.T) {
	service, cleanup := setupTestDB(t)
	defer cleanup()

	// 1. Setup Database
	dbModel := models.Database{
		Name:        "BulkDelDB",
		ContentType: "image",
		Config:      json.RawMessage("{}"),
		CustomFields: []models.CustomField{
			{Name: "tag", Type: "TEXT"},
		},
	}
	_, err := service.CreateDatabase(&dbModel)
	assert.NoError(t, err)

	// 2. Create 5 Entries
	ids := make([]int64, 0)
	for i := 0; i < 5; i++ {
		entry := models.Entry{
			"timestamp": 1000 + i,
			"filesize":  100, // 100 bytes each
			"filename":  "test.jpg",
			"status":    "ready",
			"width":     100,
			"height":    100,
			"mime_type": "image/jpeg",
			"tag":       "delete_me",
		}
		created := createTestEntry(t, service, "BulkDelDB", entry)
		ids = append(ids, created["id"].(int64))
	}

	// Verify stats before delete
	statsBefore, err := service.GetDatabaseStats("BulkDelDB")
	assert.NoError(t, err)
	assert.Equal(t, 5, statsBefore.EntryCount)
	assert.Equal(t, int64(500), statsBefore.TotalDiskSpaceBytes)

	// 3. Test Case: Success (Delete 3 existing IDs)
	toDelete := []int64{ids[0], ids[1], ids[2]}
	deletedMeta, err := service.DeleteEntries("BulkDelDB", toDelete)
	assert.NoError(t, err)
	assert.Len(t, deletedMeta, 3)
	assert.Equal(t, int64(100), deletedMeta[0].Filesize)

	// Verify they are gone
	remaining, err := service.GetEntries("BulkDelDB", 10, 0, "asc", 0, 0, dbModel.CustomFields)
	assert.NoError(t, err)
	assert.Len(t, remaining, 2)

	// Verify stats updated
	statsAfter, err := service.GetDatabaseStats("BulkDelDB")
	assert.NoError(t, err)
	assert.Equal(t, 2, statsAfter.EntryCount)
	assert.Equal(t, int64(200), statsAfter.TotalDiskSpaceBytes)

	// 4. Test Case: Partial Existence (Idempotency)
	// Try to delete ids[3] (exists) and ids[0] (already deleted)
	toDeleteMixed := []int64{ids[3], ids[0]}
	deletedMetaMixed, err := service.DeleteEntries("BulkDelDB", toDeleteMixed)
	assert.NoError(t, err)
	assert.Len(t, deletedMetaMixed, 1, "Should only delete the one that exists")
	assert.Equal(t, ids[3], deletedMetaMixed[0].ID)

	// Verify stats updated again
	statsMixed, err := service.GetDatabaseStats("BulkDelDB")
	assert.NoError(t, err)
	assert.Equal(t, 1, statsMixed.EntryCount)

	// 5. Test Case: Empty List
	deletedMetaEmpty, err := service.DeleteEntries("BulkDelDB", []int64{})
	assert.NoError(t, err)
	assert.Empty(t, deletedMetaEmpty)
}

func TestGetEntriesByID(t *testing.T) {
	service, cleanup := setupTestDB(t)
	defer cleanup()

	// 1. Setup
	dbModel := models.Database{
		Name:        "BulkGetDB",
		ContentType: "audio",
		Config:      json.RawMessage("{}"),
		CustomFields: []models.CustomField{
			{Name: "artist", Type: "TEXT"},
		},
	}
	_, err := service.CreateDatabase(&dbModel)
	assert.NoError(t, err)

	entry1 := createTestEntry(t, service, "BulkGetDB", models.Entry{
		"timestamp": 100, "filesize": 10, "filename": "a.wav", "mime_type": "audio/wav",
		"duration_sec": 10.0, "channels": 2, "artist": "Artist A",
	})
	entry2 := createTestEntry(t, service, "BulkGetDB", models.Entry{
		"timestamp": 200, "filesize": 20, "filename": "b.wav", "mime_type": "audio/wav",
		"duration_sec": 20.0, "channels": 2, "artist": "Artist B",
	})

	id1 := entry1["id"].(int64)
	id2 := entry2["id"].(int64)

	// 2. Test Case: Standard Fetch
	results, err := service.GetEntriesByID("BulkGetDB", []int64{id1, id2}, dbModel.CustomFields)
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	// Verify content
	foundMap := make(map[int64]models.Entry)
	for _, e := range results {
		foundMap[e["id"].(int64)] = e
	}
	assert.Equal(t, "Artist A", foundMap[id1]["artist"])
	assert.Equal(t, "Artist B", foundMap[id2]["artist"])

	// 3. Test Case: Filter (One exists, one doesn't)
	results, err = service.GetEntriesByID("BulkGetDB", []int64{id1, 99999}, dbModel.CustomFields)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, id1, results[0]["id"].(int64))
}
