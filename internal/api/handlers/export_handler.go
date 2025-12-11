// filepath: internal/api/handlers/export_handler.go
package handlers

import (
	"encoding/json"
	"fmt"
	"mediahub/internal/logging"
	"mediahub/internal/models"
	"net/http"
)

// @Summary Export entries as ZIP
// @Description Streams a ZIP archive containing the files and metadata (CSV) for the specified entries.
// @Tags database
// @Accept json
// @Produce application/zip
// @Param name query string true "Database Name"
// @Param body body models.ExportRequest true "List of Entry IDs to export"
// @Success 200 {file} file "ZIP Archive"
// @Failure 400 {object} ErrorResponse "Missing name or empty ID list"
// @Failure 404 {object} ErrorResponse "Database not found"
// @Failure 500 {object} ErrorResponse "Server error"
// @Security BasicAuth
// @Router /database/entries/export [post]
func (h *Handlers) ExportEntries(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required query parameter: name")
		return
	}

	var req models.ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if len(req.IDs) == 0 {
		respondWithError(w, http.StatusBadRequest, "No IDs provided for export")
		return
	}

	// Set headers for ZIP download
	w.Header().Set("Content-Type", "application/zip")
	filename := fmt.Sprintf("%s_export.zip", name)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	logging.Log.Infof("ExportEntries: Starting export for %d entries from '%s'", len(req.IDs), name)

	// Call the service to stream data directly to the ResponseWriter.
	// Note: Once the service starts writing to 'w', we cannot change the status code to error.
	// Any error occurring mid-stream will result in a truncated download, which is standard
	// behavior for chunked HTTP streams.
	if err := h.Entry.ExportEntries(r.Context(), name, req.IDs, w); err != nil {
		// Log the error. We can't send a JSON error response if headers were already sent.
		logging.Log.Errorf("ExportEntries: Streaming failed: %v", err)
		return
	}

	// Audit Log (Logged after success or partial success)
	h.Auditor.Log(r.Context(), "entry.export", getUserFromContext(r), name, map[string]interface{}{
		"count": len(req.IDs),
	})
}
