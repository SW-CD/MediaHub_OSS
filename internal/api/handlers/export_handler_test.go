// filepath: internal/api/handlers/export_handler_test.go
package handlers

import (
	"bytes"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestExportEntries(t *testing.T) {
	// Use the shared setup from main_test.go
	server, mockEntryService, _, mockAuditor, cleanup := setupEntryHandlerTestAPI(t)
	defer cleanup()

	t.Run("Success Headers", func(t *testing.T) {
		reqBody := `{"ids": [1, 2, 3]}`
		req, _ := http.NewRequest("POST", server.URL+"/database/entries/export?name=ImageDB", bytes.NewReader([]byte(reqBody)))
		req.Header.Set("Content-Type", "application/json")

		// Mock Service
		mockEntryService.On("ExportEntries", mock.Anything, "ImageDB", []int64{1, 2, 3}, mock.Anything).
			Return(nil).Once()

		// Mock Auditor
		mockAuditor.On("Log", mock.Anything, "entry.export", mock.Anything, "ImageDB", mock.MatchedBy(func(d map[string]interface{}) bool {
			return d["count"] == 3
		})).Return().Once()

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/zip", resp.Header.Get("Content-Type"))
		assert.Contains(t, resp.Header.Get("Content-Disposition"), "ImageDB_export.zip")
	})

	t.Run("Streaming Failure Logged", func(t *testing.T) {
		reqBody := `{"ids": [1]}`
		req, _ := http.NewRequest("POST", server.URL+"/database/entries/export?name=ImageDB", bytes.NewReader([]byte(reqBody)))
		req.Header.Set("Content-Type", "application/json")

		// Mock Service returning error
		mockEntryService.On("ExportEntries", mock.Anything, "ImageDB", []int64{1}, mock.Anything).
			Return(errors.New("zip error")).Once()

		// Auditor should NOT be called on failure

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)

		// Note: Since headers are written before the service is called (streaming),
		// the status code will still be 200 OK. The error is logged server-side.
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
