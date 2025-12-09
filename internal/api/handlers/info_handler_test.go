// filepath: internal/api/handlers/info_handler_test.go
package handlers

import (
	"encoding/json"
	"mediahub/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetInfo(t *testing.T) {
	testVersion := "v1.2.3-test"
	testStartTime := time.Now()
	testInfo := models.Info{
		ServiceName:     "SWCD MediaHub-API",
		Version:         testVersion,
		UptimeSince:     testStartTime,
		FFmpegAvailable: true,
	}

	infoService := new(MockInfoService)
	infoService.On("GetInfo").Return(testInfo)

	// Minimal struct since GetInfo only uses InfoService
	// But since the struct definition changed, we should use NewHandlers or update the struct literal
	// NewHandlers logic copies version/time from service, so we simulate that:
	h := &Handlers{
		Info:      infoService,
		Version:   testInfo.Version,
		StartTime: testInfo.UptimeSince,
		// Auditor: nil, // Field exists but unused here
	}

	req, err := http.NewRequest("GET", "/api/info", nil)
	assert.NoError(t, err)
	rr := httptest.NewRecorder()

	h.GetInfo(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var response models.Info
	json.Unmarshal(rr.Body.Bytes(), &response)
	assert.Equal(t, "SWCD MediaHub-API", response.ServiceName)
}
