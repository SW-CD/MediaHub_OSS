// filepath: internal/api/handlers/search_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"mediahub/internal/logging"
	"mediahub/internal/models"
	"mediahub/internal/repository"
	"net/http"
)

// @Summary Search for entries in a database (complex)
// @Description Retrieves a paginated list of entries matching complex filter criteria.
// @Tags database
// @Accept  json
// @Produce  json
// @Param   name   query  string  true   "Database Name"
// @Param   search body   models.SearchRequest true "JSON search query"
// @Success 200 {array} models.Entry "Returns an empty array if no entries match"
// @Failure 400 {object} ErrorResponse "Missing name, invalid JSON, missing limit, or invalid filter/sort"
// @Failure 404 {object} ErrorResponse "Database not found"
// @Failure 500 {object} ErrorResponse "Failed to retrieve entries"
// @Security BasicAuth
// @Router /database/entries/search [post]
func (h *Handlers) SearchEntries(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required query parameter: name")
		return
	}

	// --- Call DatabaseService ---
	// Retrieve database details to get custom fields for validation
	// This is still correct.
	db, err := h.Database.GetDatabase(name)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Database not found.")
		return
	}

	// Decode the request body
	var req models.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON request body: "+err.Error())
		return
	}

	// --- CRITICAL: Validate required pagination ---
	if req.Pagination == nil || req.Pagination.Limit == nil {
		respondWithError(w, http.StatusBadRequest, "pagination.limit is a required field")
		return
	}

	// --- Call EntryService ---
	// Call the new entry service method
	entries, err := h.Entry.SearchEntries(name, &req, db.CustomFields)
	if err != nil {
		// Check for user-facing errors (e.g., bad field/operator) vs. server errors
		if errors.Is(err, repository.ErrInvalidFilter) {
			// This is a user error, return 400 and the specific error message
			respondWithError(w, http.StatusBadRequest, err.Error())
		} else {
			// Assume other errors are internal
			logging.Log.Errorf("Failed to search entries for db '%s': %v", name, err)
			respondWithError(w, http.StatusInternalServerError, "Failed to process search request.")
		}
		return
	}

	// Ensure an empty array `[]` is returned instead of `null`
	if entries == nil {
		entries = []models.Entry{}
	}

	respondWithJSON(w, http.StatusOK, entries)
}
