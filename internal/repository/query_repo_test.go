// filepath: internal/repository/query_repo_test.go
package repository

import (
	"encoding/json"
	"mediahub/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSearchEntries_QueryBuilder tests the internal query builder logic.
func TestSearchEntries_QueryBuilder(t *testing.T) {
	service, cleanup := setupTestDB(t)
	defer cleanup()

	db := models.Database{
		Name:        "SearchQueryTestDB",
		ContentType: "image",
		Config:      json.RawMessage("{}"),
		CustomFields: []models.CustomField{
			{Name: "ml_score", Type: "REAL"},
			{Name: "description", Type: "TEXT"},
			{Name: "is_vehicle", Type: "BOOLEAN"},
		},
	}
	_, err := service.CreateDatabase(&db)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Create entries
	entry1 := models.Entry{
		"timestamp": 1, "width": 1, "height": 1, "filesize": 1, "mime_type": "image/jpeg",
		"filename": "car_photo_01.jpg", "status": "ready",
		"ml_score": 0.8, "description": "Red Car", "is_vehicle": true,
	}
	entry2 := models.Entry{
		"timestamp": 2, "width": 1, "height": 1, "filesize": 1, "mime_type": "image/jpeg",
		"filename": "person_walking.jpg", "status": "processing",
		"ml_score": 0.95, "description": "Person Walking", "is_vehicle": false,
	}
	entry3 := models.Entry{
		"timestamp": 3, "width": 1, "height": 1, "filesize": 1, "mime_type": "image/jpeg",
		"filename": "car_photo_02.png", "status": "ready",
		"ml_score": 0.5, "description": "Blue Car", "is_vehicle": true,
	}
	createTestEntry(t, service, db.Name, entry1)
	createTestEntry(t, service, db.Name, entry2)
	createTestEntry(t, service, db.Name, entry3)

	limit := 10
	pag := models.SearchPagination{Offset: 0, Limit: &limit}

	// Test 1: Simple query (ml_score > 0.9)
	req1 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "ml_score",
			Operator: ">",
			Value:    0.9,
		},
		Pagination: &pag,
	}
	entries, err := service.SearchEntries(db.Name, &req1, db.CustomFields)
	assert.NoError(t, err)
	assert.Len(t, entries, 1, "Test 1 Failed: Expected 1 entry for ml_score > 0.9")
	if len(entries) == 1 {
		assert.Equal(t, 0.95, entries[0]["ml_score"])
	}

	// Test 2: Nested query
	req2 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Operator: "and",
			Conditions: []*models.SearchFilter{
				{Field: "ml_score", Operator: "<", Value: 0.9},
				{Field: "description", Operator: "=", Value: "Blue Car"},
			},
		},
		Pagination: &pag,
	}
	entries, err = service.SearchEntries(db.Name, &req2, db.CustomFields)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	if len(entries) == 1 {
		assert.Equal(t, 0.5, entries[0]["ml_score"])
		assert.Equal(t, "Blue Car", entries[0]["description"])
	}

	// Test 3: Complex nested
	req3 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Operator: "or",
			Conditions: []*models.SearchFilter{
				{Field: "ml_score", Operator: ">", Value: 0.9},
				{Field: "description", Operator: "=", Value: "Red Car"},
			},
		},
		Pagination: &pag,
	}
	entries, err = service.SearchEntries(db.Name, &req3, db.CustomFields)
	assert.NoError(t, err)
	assert.Len(t, entries, 2)

	// Test 4: Invalid field
	req4 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "invalid_field",
			Operator: ">",
			Value:    0.9,
		},
		Pagination: &pag,
	}
	_, err = service.SearchEntries(db.Name, &req4, db.CustomFields)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid filter field: invalid_field")

	// Test 5: Invalid operator
	req5 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "ml_score",
			Operator: "CONTAINS", // Not whitelisted
			Value:    0.9,
		},
		Pagination: &pag,
	}
	_, err = service.SearchEntries(db.Name, &req5, db.CustomFields)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operator 'CONTAINS' is not allowed")

	// Test 6: Sort
	req6 := models.SearchRequest{
		Sort: &models.SearchSort{
			Field:     "ml_score",
			Direction: "asc",
		},
		Pagination: &pag,
	}
	entries, err = service.SearchEntries(db.Name, &req6, db.CustomFields)
	assert.NoError(t, err)
	assert.Len(t, entries, 3)
	assert.Equal(t, 0.5, entries[0]["ml_score"])
	assert.Equal(t, 0.8, entries[1]["ml_score"])
	assert.Equal(t, 0.95, entries[2]["ml_score"])

	// Test 7: LIKE operator
	req7 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "description",
			Operator: "LIKE",
			Value:    "Car",
		},
		Pagination: &pag,
	}
	entries, err = service.SearchEntries(db.Name, &req7, db.CustomFields)
	assert.NoError(t, err)
	assert.Len(t, entries, 2)

	// Test 8: != operator
	req8 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "ml_score",
			Operator: "!=",
			Value:    0.8,
		},
		Pagination: &pag,
	}
	entries, err = service.SearchEntries(db.Name, &req8, db.CustomFields)
	assert.NoError(t, err)
	assert.Len(t, entries, 2)

	// Test 9: != operator string
	req9 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "description",
			Operator: "!=",
			Value:    "Person Walking",
		},
		Pagination: &pag,
	}
	entries, err = service.SearchEntries(db.Name, &req9, db.CustomFields)
	assert.NoError(t, err)
	assert.Len(t, entries, 2)

	// Test 10: LIKE on non-TEXT
	req10 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "ml_score",
			Operator: "LIKE",
			Value:    "0.9",
		},
		Pagination: &pag,
	}
	_, err = service.SearchEntries(db.Name, &req10, db.CustomFields)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operator 'LIKE' is not allowed for field 'ml_score'")

	// Test 11: > on TEXT
	req11 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "description",
			Operator: ">",
			Value:    "Car",
		},
		Pagination: &pag,
	}
	_, err = service.SearchEntries(db.Name, &req11, db.CustomFields)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operator '>' is not allowed for field 'description'")

	// Test 12: = on BOOLEAN
	req12 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "is_vehicle",
			Operator: "=",
			Value:    true,
		},
		Pagination: &pag,
	}
	entries, err = service.SearchEntries(db.Name, &req12, db.CustomFields)
	assert.NoError(t, err)
	assert.Len(t, entries, 2)

	// Test 13: != on BOOLEAN
	req13 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "is_vehicle",
			Operator: "!=",
			Value:    false,
		},
		Pagination: &pag,
	}
	entries, err = service.SearchEntries(db.Name, &req13, db.CustomFields)
	assert.NoError(t, err)
	assert.Len(t, entries, 2)

	// Test 14: > on BOOLEAN
	req14 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "is_vehicle",
			Operator: ">",
			Value:    0,
		},
		Pagination: &pag,
	}
	_, err = service.SearchEntries(db.Name, &req14, db.CustomFields)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operator '>' is not allowed for field 'is_vehicle'")

	// Test 15: Search by filename (LIKE)
	req15 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "filename",
			Operator: "LIKE",
			Value:    "photo",
		},
		Pagination: &pag,
	}
	entries, err = service.SearchEntries(db.Name, &req15, db.CustomFields)
	assert.NoError(t, err)
	assert.Len(t, entries, 2)

	// Test 16: Search by status (=)
	req16 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "status",
			Operator: "=",
			Value:    "processing",
		},
		Pagination: &pag,
	}
	entries, err = service.SearchEntries(db.Name, &req16, db.CustomFields)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	if len(entries) == 1 {
		assert.Equal(t, "processing", entries[0]["status"])
	}
}
