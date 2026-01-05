// filepath: internal/api/handlers/housekeeping_handler_test.go
package handlers

import (
	"encoding/json"
	"errors"
	"mediahub/internal/models"
	"mediahub/internal/services"
	"mediahub/internal/services/mocks"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTriggerHousekeeping(t *testing.T) {
	// 1. Setup Mocks
	mockHKService := new(mocks.MockHousekeepingService)
	mockInfoSvc := new(mocks.MockInfoService)
	mockAuditor := new(mocks.MockAuditor)

	// --- FIX: Add expectation for GetInfo called by NewHandlers ---
	mockInfoSvc.On("GetInfo").Return(models.Info{
		Version:     "test",
		UptimeSince: time.Now(),
	})

	// 2. Initialize Handler
	h := NewHandlers(
		mockInfoSvc,
		nil,
		nil,
		nil,
		nil,
		mockHKService,
		mockAuditor,
		nil,
	)

	t.Run("Successful housekeeping run", func(t *testing.T) {
		report := &models.HousekeepingReport{
			DatabaseName:   "TestDB",
			EntriesDeleted: 10,
			Message:        "Success",
		}
		// Mock the service call
		mockHKService.On("TriggerHousekeeping", "TestDB").Return(report, nil).Once()

		req, _ := http.NewRequest("POST", "/database/housekeeping?name=TestDB", nil)
		rr := httptest.NewRecorder()

		h.TriggerHousekeeping(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var result models.HousekeepingReport
		err := json.Unmarshal(rr.Body.Bytes(), &result)
		assert.NoError(t, err)
		assert.Equal(t, "Success", result.Message)
		assert.Equal(t, 10, result.EntriesDeleted)
		mockHKService.AssertExpectations(t)
	})

	t.Run("Database not found", func(t *testing.T) {
		mockHKService.On("TriggerHousekeeping", "NotFoundDB").Return(nil, services.ErrNotFound).Once()

		req, _ := http.NewRequest("POST", "/database/housekeeping?name=NotFoundDB", nil)
		rr := httptest.NewRecorder()

		h.TriggerHousekeeping(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		mockHKService.AssertExpectations(t)
	})

	t.Run("Database not found (string error)", func(t *testing.T) {
		mockHKService.On("TriggerHousekeeping", "NotFoundDB").Return(nil, errors.New("database not found")).Once()

		req, _ := http.NewRequest("POST", "/database/housekeeping?name=NotFoundDB", nil)
		rr := httptest.NewRecorder()

		h.TriggerHousekeeping(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		mockHKService.AssertExpectations(t)
	})

	t.Run("Missing name parameter", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/database/housekeeping", nil)
		rr := httptest.NewRecorder()

		h.TriggerHousekeeping(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		mockHKService.AssertNotCalled(t, "TriggerHousekeeping")
	})
}
