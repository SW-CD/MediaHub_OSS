// filepath: internal/api/handlers/utils.go
package handlers

import (
	"encoding/base64"
	"fmt"
	"mediahub/internal/logging"
	"mediahub/internal/models"
	"net/http"
	"os"
	"strconv"
	"time"
)

// parseTimestamp tries to parse a string into a Unix timestamp.
// It supports RFC3339 and Unix timestamp formats.
func parseTimestamp(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	// Try parsing as Unix timestamp first
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return ts, nil
	}
	// Try parsing as RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Unix(), nil
	}
	return 0, &time.ParseError{Layout: "RFC3339 or Unix", Value: s}
}

// serveFileAsJSON reads a file from disk, encodes it as Base64, and returns it as a JSON object.
// This is used to support clients that cannot handle binary streams with auth headers (e.g., Grafana).
func serveFileAsJSON(w http.ResponseWriter, filePath, filename, mimeType string) {
	// 1. Read the file into memory
	data, err := os.ReadFile(filePath)
	if err != nil {
		logging.Log.Errorf("serveFileAsJSON: Failed to read file '%s': %v", filePath, err)
		respondWithError(w, http.StatusInternalServerError, "Failed to read file content.")
		return
	}

	// 2. Encode to Base64
	b64Str := base64.StdEncoding.EncodeToString(data)

	// 3. Construct the response object
	resp := models.FileJSONResponse{
		Filename: filename,
		MimeType: mimeType,
		Data:     fmt.Sprintf("data:%s;base64,%s", mimeType, b64Str),
	}

	// 4. Send JSON response
	respondWithJSON(w, http.StatusOK, resp)
}
