// filepath: internal/api/handlers/search_handler_test.go
package handlers

import (
	"bytes" // <-- Import bytes
	"encoding/json"
	"errors"
	"fmt"
	"mediahub/internal/config"
	"mediahub/internal/models"
	"mediahub/internal/repository" // <-- IMPORT REPOSITORY for error
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// --- REFACTOR: MockDatabaseService is now defined in main_test.go ---

// setupSearchTestAPI creates a new test server and a cleanup function.
// --- FIX: Return MockEntryService ---
func setupSearchTestAPI(t *testing.T) (*httptest.Server, *MockDatabaseService, *MockEntryService, func()) {
	t.Helper()

	mockDBService := new(MockDatabaseService)
	mockEntryService := new(MockEntryService) // <-- FIX: Create MockEntryService
	dummyCfg := &config.Config{}

	// --- REFACTOR: Mock InfoService ---
	infoSvc := new(MockInfoService)
	infoSvc.On("GetInfo").Return(models.Info{
		Version:     "test",
		UptimeSince: time.Now(),
	})
	h := NewHandlers(
		infoSvc, // info
		nil,     // user
		nil,     // token (Added)
		mockDBService,
		mockEntryService, // <-- FIX: Pass mockEntryService
		nil,              // housekeeping
		dummyCfg,
	)
	// --- END REFACTOR ---

	r := mux.NewRouter()
	r.HandleFunc("/database/entries/search", h.SearchEntries).Methods("POST")

	server := httptest.NewServer(r)

	cleanup := func() {
		server.Close()
	}

	return server, mockDBService, mockEntryService, cleanup
}

// TestSearchEntriesAPI tests the new POST /database/entries/search endpoint.
func TestSearchEntriesAPI(t *testing.T) {
	// --- FIX: Get mockEntrySvc from helper ---
	server, mockDB, mockEntrySvc, cleanup := setupSearchTestAPI(t)
	defer cleanup()

	// Create a database with a custom field
	dbPayload := &models.Database{
		Name:        "SearchAPITestDB",
		ContentType: "image",
		CustomFields: []models.CustomField{
			{Name: "score", Type: "REAL"},
			{Name: "description", Type: "TEXT"},
		},
	}
	// 1. Mock GetDatabase call (this is still correct)
	mockDB.On("GetDatabase", "SearchAPITestDB").Return(dbPayload, nil)

	// Test 1: Valid search
	searchReq1 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "score",
			Operator: ">",
			Value:    0.9,
		},
		Pagination: &models.SearchPagination{Limit: func(i int) *int { return &i }(10)},
	}
	// 2. Mock SearchEntries call
	// --- FIX: Mock mockEntrySvc, not mockDB ---
	// --- FIX: Cast custom fields to slice type ---
	mockEntrySvc.On("SearchEntries", "SearchAPITestDB", &searchReq1, []models.CustomField(dbPayload.CustomFields)).Return([]models.Entry{
		{"id": 2, "score": 0.95},
	}, nil).Once()

	searchBody, _ := json.Marshal(searchReq1)                                                                                           // <-- This is []byte
	resp, err := http.Post(server.URL+"/database/entries/search?name=SearchAPITestDB", "application/json", bytes.NewReader(searchBody)) // <-- FIX: Use bytes.NewReader
	assert.NoError(t, err)

	// Check for panic first
	if resp == nil {
		t.Fatal("Response was nil, indicating a panic occurred in the server")
	}

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var entries []models.Entry
	err = json.NewDecoder(resp.Body).Decode(&entries)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 0.95, entries[0]["score"])
	mockDB.AssertCalled(t, "GetDatabase", "SearchAPITestDB")
	mockEntrySvc.AssertCalled(t, "SearchEntries", "SearchAPITestDB", &searchReq1, []models.CustomField(dbPayload.CustomFields))

	// Test 2: Missing pagination.limit
	searchBody = []byte(`{ "filter": { "field": "score", "operator": ">", "value": 0.9 } }`)                                           // <-- FIX: Assign []byte
	resp, err = http.Post(server.URL+"/database/entries/search?name=SearchAPITestDB", "application/json", bytes.NewReader(searchBody)) // <-- FIX: Use bytes.NewReader
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	var errResp ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	assert.NoError(t, err)
	assert.Equal(t, "pagination.limit is a required field", errResp.Error)

	// Test 3: Invalid field (service returns error)
	searchReq3 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "bad_field",
			Operator: ">",
			Value:    0.9,
		},
		Pagination: &models.SearchPagination{Limit: func(i int) *int { return &i }(10)},
	}
	// 3. Mock SearchEntries call returning a user error
	// --- FIX: Mock mockEntrySvc, not mockDB ---
	// --- FIX: Cast custom fields to slice type ---
	mockEntrySvc.On("SearchEntries", "SearchAPITestDB", &searchReq3, []models.CustomField(dbPayload.CustomFields)).Return(nil, fmt.Errorf("%w: invalid filter field: bad_field", repository.ErrInvalidFilter)).Once()

	searchBody, _ = json.Marshal(searchReq3)                                                                                           // <-- This is []byte
	resp, err = http.Post(server.URL+"/database/entries/search?name=SearchAPITestDB", "application/json", bytes.NewReader(searchBody)) // <-- FIX: Use bytes.NewReader
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp.Error, "invalid filter field: bad_field")
	mockEntrySvc.AssertCalled(t, "SearchEntries", "SearchAPITestDB", &searchReq3, []models.CustomField(dbPayload.CustomFields))

	// Test 4: Database not found
	mockDB.On("GetDatabase", "NonExistentDB").Return(nil, errors.New("not found")).Once()
	searchBodyValid, _ := json.Marshal(models.SearchRequest{Pagination: &models.SearchPagination{Limit: func(i int) *int { return &i }(10)}})
	resp, err = http.Post(server.URL+"/database/entries/search?name=NonExistentDB", "application/json", bytes.NewReader(searchBodyValid)) // <-- FIX: Use bytes.NewReader
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	mockDB.AssertCalled(t, "GetDatabase", "NonExistentDB")
}
