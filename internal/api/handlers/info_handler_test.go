// filepath: internal/api/handlers/info_handler_test.go
package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mediahub/internal/models"
	// <-- IMPORT SERVICES
	"github.com/stretchr/testify/assert"
)

func TestGetInfo(t *testing.T) {
	// Define a fixed start time for a predictable test
	testVersion := "v1.2.3-test"
	testStartTime := time.Now().Add(-10 * time.Minute).Truncate(time.Second)
	testInfo := models.Info{
		ServiceName:     "SWCD MediaHub-API",
		Version:         testVersion,
		UptimeSince:     testStartTime,
		FFmpegAvailable: true,
	}

	// --- REFACTOR: Mock InfoService ---
	infoService := new(MockInfoService) // Use mock
	infoService.On("GetInfo").Return(testInfo)
	// --- END REFACTOR ---

	// Create a minimal handler struct with just the dependencies needed for GetInfo
	h := &Handlers{
		Info: infoService, // <-- INJECT MOCKED SERVICE
	}

	// Create a request and response recorder
	req, err := http.NewRequest("GET", "/api/info", nil)
	assert.NoError(t, err)
	rr := httptest.NewRecorder()

	// Call the handler function
	h.GetInfo(rr, req)

	// --- Assertions ---
	assert.Equal(t, http.StatusOK, rr.Code, "Expected status code 200 OK")

	// 2. Decode the JSON response
	var response models.Info
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err, "Failed to unmarshal JSON response")

	// 3. Check the response fields
	assert.Equal(t, "SWCD MediaHub-API", response.ServiceName, "Service name mismatch")
	assert.Equal(t, testVersion, response.Version, "Version mismatch")
	assert.Equal(t, testStartTime.UTC(), response.UptimeSince.UTC(), "UptimeSince mismatch")
	assert.True(t, response.FFmpegAvailable, "FFmpeg availability mismatch")

	// 4. Verify mock was called
	infoService.AssertExpectations(t)
}
