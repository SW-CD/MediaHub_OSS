// filepath: internal/repository/query_repo_test.go
package repository

import (
	"encoding/json" // <-- ADDED
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
		Config:      json.RawMessage("{}"), // <-- FIX: Initialize Config
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
	// ---
	// FIX: Added "status": "ready" to all test entries
	// ---
	entry1 := models.Entry{"timestamp": 1, "width": 1, "height": 1, "filesize": 1, "mime_type": "image/jpeg", "filename": "", "status": "ready", "ml_score": 0.8, "description": "Red Car", "is_vehicle": true}
	entry2 := models.Entry{"timestamp": 2, "width": 1, "height": 1, "filesize": 1, "mime_type": "image/jpeg", "filename": "", "status": "ready", "ml_score": 0.95, "description": "Person Walking", "is_vehicle": false}
	entry3 := models.Entry{"timestamp": 3, "width": 1, "height": 1, "filesize": 1, "mime_type": "image/jpeg", "filename": "", "status": "ready", "ml_score": 0.5, "description": "Blue Car", "is_vehicle": true}
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

	// Test 2: Nested query: (ml_score < 0.9) AND (description = "Blue Car")
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
	assert.Len(t, entries, 1, "Test 2 Failed: Expected 1 entry for (ml_score < 0.9) AND (description = 'Blue Car')")
	if len(entries) == 1 {
		assert.Equal(t, 0.5, entries[0]["ml_score"])
		assert.Equal(t, "Blue Car", entries[0]["description"])
	}

	// Test 3: Complex nested: (ml_score > 0.9) OR (description = "Red Car")
	req3 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Operator: "or",
			Conditions: []*models.SearchFilter{
				{Field: "ml_score", Operator: ">", Value: 0.9},          // entry2
				{Field: "description", Operator: "=", Value: "Red Car"}, // entry1
			},
		},
		Pagination: &pag,
	}
	entries, err = service.SearchEntries(db.Name, &req3, db.CustomFields)
	assert.NoError(t, err)
	assert.Len(t, entries, 2, "Test 3 Failed: Expected 2 entries for (ml_score > 0.9) OR (description = 'Red Car')")

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
	assert.Contains(t, err.Error(), "invalid or unsupported operator: CONTAINS")

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

	// --- NEW TESTS for LIKE and != ---

	// Test 7: LIKE operator (contains "Car")
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
	assert.Len(t, entries, 2, "Test 7 Failed: Expected 2 entries for description LIKE '%Car%'") // entry1, entry3

	// Test 8: != operator (ml_score != 0.8)
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
	assert.Len(t, entries, 2, "Test 8 Failed: Expected 2 entries for ml_score != 0.8") // entry2, entry3

	// Test 9: != operator (description != "Person Walking")
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
	assert.Len(t, entries, 2, "Test 9 Failed: Expected 2 entries for description != 'Person Walking'") // entry1, entry3

	// Test 10: LIKE operator on non-TEXT field (should fail validation)
	req10 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "ml_score",
			Operator: "LIKE",
			Value:    "0.9",
		},
		Pagination: &pag,
	}
	_, err = service.SearchEntries(db.Name, &req10, db.CustomFields)
	assert.Error(t, err, "Test 10 Failed: Expected error for LIKE on non-TEXT field")
	if err != nil {
		assert.Contains(t, err.Error(), "operator 'LIKE' is not allowed for field 'ml_score'")
	}

	// Test 11: > operator on TEXT field (should fail validation)
	req11 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "description",
			Operator: ">",
			Value:    "Car",
		},
		Pagination: &pag,
	}
	_, err = service.SearchEntries(db.Name, &req11, db.CustomFields)
	assert.Error(t, err, "Test 11 Failed: Expected error for > on TEXT field")
	if err != nil {
		assert.Contains(t, err.Error(), "operator '>' is not allowed for field 'description'")
	}

	// Test 12: = operator on BOOLEAN field (true)
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
	assert.Len(t, entries, 2, "Test 12 Failed: Expected 2 entries for is_vehicle = true") // entry1, entry3

	// Test 13: != operator on BOOLEAN field (false -> means is_vehicle != 0 -> is_vehicle == 1)
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
	assert.Len(t, entries, 2, "Test 13 Failed: Expected 2 entries for is_vehicle != false") // entry1, entry3

	// Test 14: > operator on BOOLEAN field (should fail validation)
	req14 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "is_vehicle",
			Operator: ">",
			Value:    0, // Even comparing to 0/1 shouldn't work with >
		},
		Pagination: &pag,
	}
	_, err = service.SearchEntries(db.Name, &req14, db.CustomFields)
	assert.Error(t, err, "Test 14 Failed: Expected error for > on BOOLEAN field")
	if err != nil {
		assert.Contains(t, err.Error(), "operator '>' is not allowed for field 'is_vehicle'")
	}
}
