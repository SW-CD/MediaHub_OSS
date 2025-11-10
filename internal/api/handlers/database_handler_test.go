// filepath: internal/api/handlers/database_handler_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"mediahub/internal/config"
	"mediahub/internal/models" // <-- IMPORT SERVICES
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// setupDBHandlerTestAPI creates a new test server for database handlers.
func setupDBHandlerTestAPI(t *testing.T) (*httptest.Server, *MockDatabaseService, *MockInfoService, func()) {
	t.Helper()

	mockDBService := new(MockDatabaseService)
	mockInfoService := new(MockInfoService) // <-- Create mock info service
	dummyCfg := &config.Config{}            // Cfg is not used by these handlers directly

	mockInfoService.On("GetInfo").Return(models.Info{
		Version:     "test",
		UptimeSince: time.Now(),
	})
	h := NewHandlers(
		mockInfoService, // info
		nil,             // user
		mockDBService,
		nil, // entry
		nil, // housekeeping
		dummyCfg,
	)

	r := mux.NewRouter()
	r.HandleFunc("/database", h.CreateDatabase).Methods("POST")
	r.HandleFunc("/database", h.UpdateDatabase).Methods("PUT")
	r.HandleFunc("/databases", h.GetDatabases).Methods("GET")
	r.HandleFunc("/database", h.GetDatabase).Methods("GET")

	server := httptest.NewServer(r)
	cleanup := func() {
		server.Close()
	}

	return server, mockDBService, mockInfoService, cleanup
}

func TestDatabaseAPI(t *testing.T) {
	server, mockDB, _, cleanup := setupDBHandlerTestAPI(t)
	defer cleanup()

	// --- Create Database ---
	createPayload := models.DatabaseCreatePayload{
		Name:        "APITestDB",
		ContentType: "image",
		Config: map[string]interface{}{
			"create_preview":  true,
			"convert_to_jpeg": false,
		},
		CustomFields: []models.CustomField{
			{Name: "location", Type: "TEXT"},
		},
		Housekeeping: &models.Housekeeping{
			Interval: "10h",
		},
	}
	payloadBytes, _ := json.Marshal(createPayload)

	// Mock the service call
	// This is what the service will return
	returnedDBModel := models.Database{
		Name:        "APITestDB",
		Config:      json.RawMessage(`{"convert_to_jpeg":false, "create_preview":true}`), // <-- FIX: Cast to json.RawMessage
		ContentType: "image",
		Housekeeping: models.Housekeeping{
			Interval:  "10h",
			DiskSpace: "100G", // Service filled in default
			MaxAge:    "365d",
		},
		CustomFields: []models.CustomField{
			{Name: "location", Type: "TEXT"},
		},
	}
	mockDB.On("CreateDatabase", createPayload).Return(&returnedDBModel, nil).Once()

	resp, err := http.Post(server.URL+"/database", "application/json", bytes.NewReader(payloadBytes))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdDB models.Database
	err = json.NewDecoder(resp.Body).Decode(&createdDB)
	assert.NoError(t, err)
	assert.Equal(t, "APITestDB", createdDB.Name)
	assert.Equal(t, "10h", createdDB.Housekeeping.Interval)
	assert.Equal(t, "100G", createdDB.Housekeeping.DiskSpace) // Check if default was applied by service
	mockDB.AssertCalled(t, "CreateDatabase", createPayload)

	// --- Get Database ---
	mockDB.On("GetDatabase", "APITestDB").Return(&returnedDBModel, nil).Once()
	resp, err = http.Get(server.URL + "/database?name=APITestDB")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var readDB models.Database
	err = json.NewDecoder(resp.Body).Decode(&readDB)
	assert.NoError(t, err)
	assert.Equal(t, "APITestDB", readDB.Name)
	assert.Contains(t, string(readDB.Config), `"convert_to_jpeg":false`) // <-- FIX: Cast config to string for check
	mockDB.AssertCalled(t, "GetDatabase", "APITestDB")

	// --- Update Database ---
	updatePayload := models.DatabaseUpdatePayload{
		Config: map[string]interface{}{
			"create_preview": false, // Change a value
		},
		Housekeeping: &models.Housekeeping{
			MaxAge: "90d", // Change a different value
		},
	}
	payloadBytes, _ = json.Marshal(updatePayload)

	// This is what the service will return
	finalUpdatedDBModel := returnedDBModel                                   // copy
	finalUpdatedDBModel.Config = json.RawMessage(`{"create_preview":false}`) // <-- FIX: Cast to json.RawMessage
	finalUpdatedDBModel.Housekeeping.MaxAge = "90d"

	mockDB.On("UpdateDatabase", "APITestDB", updatePayload).Return(&finalUpdatedDBModel, nil).Once()

	req, _ := http.NewRequest("PUT", server.URL+"/database?name=APITestDB", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var updatedDB models.Database
	err = json.NewDecoder(resp.Body).Decode(&updatedDB)
	assert.NoError(t, err)
	assert.Equal(t, "90d", updatedDB.Housekeeping.MaxAge)
	assert.Contains(t, string(updatedDB.Config), `"create_preview":false`) // <-- FIX: Cast config to string for check
	mockDB.AssertCalled(t, "UpdateDatabase", "APITestDB", updatePayload)

	// --- Get Databases ---
	mockDB.On("GetDatabases").Return([]models.Database{finalUpdatedDBModel}, nil).Once()
	resp, err = http.Get(server.URL + "/databases")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var dbs []models.Database
	err = json.NewDecoder(resp.Body).Decode(&dbs)
	assert.NoError(t, err)
	assert.Len(t, dbs, 1)
	mockDB.AssertCalled(t, "GetDatabases")
}
