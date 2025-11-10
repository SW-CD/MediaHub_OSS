// filepath: internal/api/handlers/housekeeping_handler.go
package handlers

import (
	"errors"
	"mediahub/internal/services" // <-- IMPORT SERVICES
	"net/http"
	"strings"
)

// @Summary Trigger housekeeping for a database
// @Description Manually triggers the housekeeping task for a specific database.
// @Tags database
// @Produce  json
// @Param   name  query  string  true  "Database Name"
// @Success 200 {object} models.HousekeepingReport
// @Failure 400 {object} ErrorResponse "Missing name parameter"
// @Failure 404 {object} ErrorResponse "Database not found"
// @Failure 500 {object} ErrorResponse "Housekeeping failed"
// @Security BasicAuth
// @Router /database/housekeeping [post]
func (h *Handlers) TriggerHousekeeping(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required query parameter: name")
		return
	}

	// --- Call HousekeepingService ---
	report, err := h.Housekeeping.TriggerHousekeeping(name)
	if err != nil {
		// Check for a "not found" style error
		if errors.Is(err, services.ErrNotFound) || strings.Contains(err.Error(), "not found") {
			respondWithError(w, http.StatusNotFound, "Database not found.")
		} else {
			respondWithError(w, http.StatusInternalServerError, "Housekeeping failed.")
		}
		return
	}

	respondWithJSON(w, http.StatusOK, report)
}
