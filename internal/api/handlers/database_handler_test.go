// filepath: internal/api/handlers/database_handler_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"mediahub/internal/config"
	"mediahub/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// setupDBHandlerTestAPI creates a new test server for database handlers.
func setupDBHandlerTestAPI(t *testing.T) (*httptest.Server, *MockDatabaseService, *MockInfoService, *MockAuditor, func()) {
	t.Helper()

	mockDBService := new(MockDatabaseService)
	mockInfoService := new(MockInfoService)
	mockAuditor := new(MockAuditor)
	dummyCfg := &config.Config{}

	mockInfoService.On("GetInfo").Return(models.Info{
		Version:     "test",
		UptimeSince: time.Now(),
	})
	h := NewHandlers(
		mockInfoService,
		nil,
		nil,
		mockDBService,
		nil,
		nil,
		mockAuditor,
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

	return server, mockDBService, mockInfoService, mockAuditor, cleanup
}

func TestDatabaseAPI(t *testing.T) {
	server, mockDB, _, mockAuditor, cleanup := setupDBHandlerTestAPI(t)
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

	returnedDBModel := models.Database{
		Name:        "APITestDB",
		ContentType: "image",
		Config:      json.RawMessage(`{"convert_to_jpeg":false, "create_preview":true}`),
		Housekeeping: models.Housekeeping{
			Interval:  "10h",
			DiskSpace: "100G",
			MaxAge:    "365d",
		},
		CustomFields: []models.CustomField{
			{Name: "location", Type: "TEXT"},
		},
	}
	mockDB.On("CreateDatabase", createPayload).Return(&returnedDBModel, nil).Once()

	// Audit Log Expectation
	mockAuditor.On("Log",
		mock.Anything, // Context
		"database.create",
		mock.Anything, // Actor
		"APITestDB",
		mock.MatchedBy(func(details map[string]interface{}) bool {
			return details["content_type"] == "image"
		}),
	).Return().Once()

	resp, err := http.Post(server.URL+"/database", "application/json", bytes.NewReader(payloadBytes))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdDB models.Database
	err = json.NewDecoder(resp.Body).Decode(&createdDB)
	assert.NoError(t, err)
	assert.Equal(t, "APITestDB", createdDB.Name)
	assert.Equal(t, "10h", createdDB.Housekeeping.Interval)

	mockAuditor.AssertExpectations(t)

	// --- Get Database ---
	mockDB.On("GetDatabase", "APITestDB").Return(&returnedDBModel, nil).Once()
	resp, err = http.Get(server.URL + "/database?name=APITestDB")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var readDB models.Database
	err = json.NewDecoder(resp.Body).Decode(&readDB)
	assert.NoError(t, err)
	assert.Equal(t, "APITestDB", readDB.Name)

	// --- Update Database ---
	updatePayload := models.DatabaseUpdatePayload{
		Config: map[string]interface{}{
			"create_preview": false,
		},
		Housekeeping: &models.Housekeeping{
			MaxAge: "90d",
		},
	}
	payloadBytes, _ = json.Marshal(updatePayload)

	finalUpdatedDBModel := returnedDBModel
	finalUpdatedDBModel.Config = json.RawMessage(`{"create_preview":false}`)
	finalUpdatedDBModel.Housekeeping.MaxAge = "90d"

	mockDB.On("UpdateDatabase", "APITestDB", updatePayload).Return(&finalUpdatedDBModel, nil).Once()

	// --- FIX: Audit Log Expectation to allow nil details ---
	mockAuditor.On("Log",
		mock.Anything,
		"database.update",
		mock.Anything,
		"APITestDB",
		mock.Anything, // Accepts nil or map[string]interface{}(nil)
	).Return().Once()

	req, _ := http.NewRequest("PUT", server.URL+"/database?name=APITestDB", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	mockAuditor.AssertExpectations(t)

	// --- Get Databases ---
	mockDB.On("GetDatabases").Return([]models.Database{finalUpdatedDBModel}, nil).Once()
	resp, err = http.Get(server.URL + "/databases")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
