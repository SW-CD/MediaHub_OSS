// filepath: internal/api/handlers/info_handler_test.go
package handlers

import (
	"encoding/json"
	"mediahub/internal/models"
	"mediahub/internal/services/mocks" // Import shared mocks
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

	// Use shared mock
	infoService := new(mocks.MockInfoService)
	infoService.On("GetInfo").Return(testInfo)

	// Construct Handler
	h := &Handlers{
		Info:      infoService,
		Version:   testInfo.Version,
		StartTime: testInfo.UptimeSince,
	}

	req, err := http.NewRequest("GET", "/api/info", nil)
	assert.NoError(t, err)
	rr := httptest.NewRecorder()

	h.GetInfo(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var response models.Info
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "SWCD MediaHub-API", response.ServiceName)
}
