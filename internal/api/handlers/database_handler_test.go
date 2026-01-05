// filepath: internal/api/handlers/database_handler_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"mediahub/internal/models"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDatabaseAPI(t *testing.T) {
	// Use the shared setup from main_test.go
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

	mockAuditor.On("Log",
		mock.Anything,
		"database.update",
		mock.Anything,
		"APITestDB",
		mock.Anything,
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

func TestDeleteEntries(t *testing.T) {
	// Use the FULL setup that includes the EntryService mock
	server, _, mockEntryService, _, mockAuditor, cleanup := setupDBHandlerTestAPI_Full(t)
	defer cleanup()

	// 1. Success Case
	t.Run("Success", func(t *testing.T) {
		reqBody := `{"ids": [101, 102]}`
		req, _ := http.NewRequest("POST", server.URL+"/database/entries/delete?name=TestDB", bytes.NewReader([]byte(reqBody)))
		req.Header.Set("Content-Type", "application/json")

		// Mock Service
		mockEntryService.On("DeleteEntries", "TestDB", []int64{101, 102}).
			Return(2, int64(2048), nil).Once()

		// Mock Auditor
		mockAuditor.On("Log", mock.Anything, "entry.bulk_delete", mock.Anything, "TestDB", mock.MatchedBy(func(d map[string]interface{}) bool {
			// json unmarshal of numbers might be float64 in tests sometimes, but here we passed struct args
			return d["count"] == 2
		})).Return().Once()

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var respBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&respBody)
		// Checking float64 because JSON decodes numbers as floats
		assert.Equal(t, float64(2), respBody["deleted_count"])
	})

	// 2. Validation Failure (Empty List)
	t.Run("EmptyList", func(t *testing.T) {
		reqBody := `{"ids": []}`
		req, _ := http.NewRequest("POST", server.URL+"/database/entries/delete?name=TestDB", bytes.NewReader([]byte(reqBody)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		// Ensure mock was NOT called
		mockEntryService.AssertNotCalled(t, "DeleteEntries")
	})
}
