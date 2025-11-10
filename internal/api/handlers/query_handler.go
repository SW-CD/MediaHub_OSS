// filepath: internal/api/handlers/query_handler.go
package handlers

import (
	"mediahub/internal/logging"
	"mediahub/internal/models"
	"net/http"
	"strconv"
	"strings"
)

// @Summary Get entries from a database (basic)
// @Description Retrieves a paginated list of entries from a specific database. Only supports time-based filters.
// @Tags database
// @Produce  json
// @Param   name   query  string  true   "Database Name"
// @Param   limit  query  int     false  "Number of entries to return (default 30)"
// @Param   offset query  int     false  "Offset for pagination (default 0)"
// @Param   order  query  string  false  "Sort order ('asc' or 'desc', default 'desc')"
// @Param   tstart query  string  false  "Start timestamp (RFC3339 or Unix seconds)"
// @Param   tend   query  string  false  "End timestamp (RFC3339 or Unix seconds)"
// @Success 200 {array} models.Entry "Returns an empty array if no entries match"
// @Failure 400 {object} ErrorResponse "Missing name parameter"
// @Failure 404 {object} ErrorResponse "Database not found"
// @Failure 500 {object} ErrorResponse "Failed to retrieve entries"
// @Security BasicAuth
// @Router /database/entries [get]
func (h *Handlers) QueryEntries(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required query parameter: name")
		return
	}

	// --- Call DatabaseService to get DB (for custom fields) ---
	// This is still correct, we need the custom fields from the DB definition
	db, err := h.Database.GetDatabase(name)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Database not found.")
		return
	}

	// Parse pagination and sorting parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 30 // Default limit
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0 // Default offset
	}
	order := strings.ToLower(r.URL.Query().Get("order"))
	if order != "asc" {
		order = "desc" // Default order
	}

	// Parse timestamp parameters
	tstart, _ := parseTimestamp(r.URL.Query().Get("tstart")) // Returns 0 if empty or invalid
	tend, _ := parseTimestamp(r.URL.Query().Get("tend"))     // Returns 0 if empty or invalid

	logging.Log.Debug("QueryEntries: Calling entry service for simple query.")

	// --- Call EntryService ---
	// Call the entry service with parsed parameters.
	entries, err := h.Entry.GetEntries(name, limit, offset, order, tstart, tend, db.CustomFields)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve entries.")
		return
	}

	// Ensure an empty array `[]` is returned instead of `null` if no entries match
	if entries == nil {
		entries = []models.Entry{}
	}

	respondWithJSON(w, http.StatusOK, entries)
}
