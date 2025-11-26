// filepath: internal/api/handlers/entry_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"mediahub/internal/logging"
	"mediahub/internal/models"
	"mediahub/internal/services"
	"net/http"
	"os"
	"strconv"
)

// @Summary Upload an entry
// @Description Uploads a new entry to a specified database using multipart/form-data. The metadata part should be a JSON object containing the entry's timestamp, and any custom fields.
// @Description The 'file' part's 'filename' in the Content-Disposition header will be extracted and saved.
// @Description
// @Description This endpoint uses a hybrid model:
// @Description - **Small files (<= 8MB):** Processed synchronously. Returns `201 Created` with the full entry metadata.
// @Description - **Large files (> 8MB):** Processed asynchronously. Returns `202 Accepted` with a partial response. The client should poll `GET /api/entry/meta` until the `status` field is 'ready'.
// @Tags entry
// @Accept  mpfd
// @Produce  json
// @Param   database_name  query  string  true  "Database Name"
// @Param   metadata       formData  string  true  "JSON metadata for the entry (including custom fields)"
// @Param   file           formData  file    true  "Entry file"
// @Success 201 {object} models.Entry "For small files (synchronous processing)"
// @Success 202 {object} models.PartialEntryResponse "For large files (asynchronous processing)"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 404 {object} ErrorResponse "Database not found"
// @Failure 415 {object} ErrorResponse "Unsupported entry format"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /entry [post]
func (h *Handlers) UploadEntry(w http.ResponseWriter, r *http.Request) {

	// 1. Parse database name
	dbName := r.URL.Query().Get("database_name")
	if dbName == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required query parameter: database_name")
		return
	}

	// 2. Parse multipart form
	// ---
	// Use ParseMultipartForm
	// This is the key that enables the (*os.File) type assertion.
	// We set a max in-memory size (e.g., 8MB). Anything larger
	// will be spooled to a temp file on disk by the http server.
	// ---
	if err := r.ParseMultipartForm(8 << 20); err != nil { // 8MB max in-memory
		logging.Log.Warnf("Failed to parse multipart form: %v", err)
		respondWithError(w, http.StatusBadRequest, "Failed to parse multipart form.")
		return
	}

	// 3. Get metadata part
	metadataStr := r.FormValue("metadata")
	if metadataStr == "" {
		logging.Log.Warn("Missing 'metadata' part in multipart form")
		respondWithError(w, http.StatusBadRequest, "Missing 'metadata' part in multipart form.")
		return
	}

	// 4. Get file part
	file, header, err := r.FormFile("file")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Missing 'file' part in multipart form.")
		return
	}
	defer file.Close()

	// 5. Call the EntryService. It returns (body, status, error)
	body, status, err := h.Entry.CreateEntry(dbName, metadataStr, file, header)
	if err != nil {
		// Map service errors to HTTP status codes
		if errors.Is(err, services.ErrNotFound) {
			respondWithError(w, http.StatusNotFound, "Database not found.")
		} else if errors.Is(err, services.ErrValidation) {
			respondWithError(w, http.StatusBadRequest, err.Error())
		} else if errors.Is(err, services.ErrUnsupported) {
			respondWithError(w, http.StatusUnsupportedMediaType, err.Error())
		} else if errors.Is(err, services.ErrDependencies) {
			respondWithError(w, http.StatusBadRequest, err.Error()) // e.g., FFmpeg missing
		} else if status == http.StatusNotImplemented {
			// This is our stub from handleLargeFileAsync
			respondWithError(w, http.StatusNotImplemented, err.Error())
		} else {
			logging.Log.Errorf("UploadEntry: Unhandled error from EntryService: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to create entry.")
		}
		return
	}

	// 6. Respond to the client with the status and body from the service
	//    (This will be 201 or 202)
	respondWithJSON(w, status, body)
}

// @Summary Get an entry file
// @Description Retrieves a raw entry file.
// @Tags entry
// @Produce octet-stream
// @Param   database_name  query  string  true  "Database Name"
// @Param   id             query  int     true  "Entry ID"
// @Success 200 {file} file "The raw file data"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 404 {object} ErrorResponse "Database or entry not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /entry/file [get]
func (h *Handlers) GetEntry(w http.ResponseWriter, r *http.Request) {
	dbName := r.URL.Query().Get("database_name")
	if dbName == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required query parameter: database_name")
		return
	}
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID format.")
		return
	}

	// Call EntryService ---
	entryPath, mimeType, filename, err := h.Entry.GetEntryFile(dbName, id)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			respondWithError(w, http.StatusNotFound, "Database or entry not found.")
		} else {
			respondWithError(w, http.StatusInternalServerError, "Failed to get entry file.")
		}
		return
	}

	w.Header().Set("Content-Type", mimeType)

	// Set Content-Disposition header if filename is available
	if filename != "" {
		// Use fmt.Sprintf to safely quote the filename
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	}

	http.ServeFile(w, r, entryPath)
}

// @Summary Get an entry preview
// @Description Retrieves a 200x200 JPEG preview of an entry.
// @Tags entry
// @Produce jpeg
// @Param   database_name  query  string  true  "Database Name"
// @Param   id             query  int     true  "Entry ID"
// @Success 200 {file} file "The JPEG preview image"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 404 {object} ErrorResponse "Database or entry not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /entry/preview [get]
func (h *Handlers) GetEntryPreview(w http.ResponseWriter, r *http.Request) {
	dbName := r.URL.Query().Get("database_name")
	if dbName == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required query parameter: database_name")
		return
	}
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID format.")
		return
	}

	// Call EntryService ---
	previewPath, err := h.Entry.GetEntryPreview(dbName, id)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			respondWithError(w, http.StatusNotFound, "Database or entry not found.")
		} else {
			respondWithError(w, http.StatusInternalServerError, "Failed to get preview file.")
		}
		return
	}

	if _, err := os.Stat(previewPath); os.IsNotExist(err) {
		respondWithError(w, http.StatusNotFound, "Preview file not found.")
		return
	}

	// Set cache-control headers ---
	// This tells the browser to never cache this response.
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1.
	w.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0.
	w.Header().Set("Expires", "0")                                         // Proxies.

	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeFile(w, r, previewPath)
}

// @Summary Delete an entry
// @Description Deletes an entry file from disk and its metadata from the database.
// @Tags entry
// @Produce json
// @Param   database_name  query  string  true  "Database Name"
// @Param   id             query  int     true  "Entry ID"
// @Success 200 {object} MessageResponse "Success message"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 404 {object} ErrorResponse "Database or entry not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /entry [delete]
func (h *Handlers) DeleteEntry(w http.ResponseWriter, r *http.Request) {
	dbName := r.URL.Query().Get("database_name")
	if dbName == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required query parameter: database_name")
		return
	}
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID format.")
		return
	}

	// --- Call EntryService ---
	if err := h.Entry.DeleteEntry(dbName, id); err != nil {
		if errors.Is(err, services.ErrNotFound) {
			respondWithError(w, http.StatusNotFound, "Database or entry not found.")
		} else {
			respondWithError(w, http.StatusInternalServerError, "Failed to delete entry.")
		}
		return
	}

	logging.Log.Infof("Entry deleted: %s from database %s", idStr, dbName)
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Entry '" + idStr + "' was successfully deleted."})
}

// @Summary Get entry metadata
// @Description Retrieves all metadata for a single entry, including custom fields.
// @Tags entry
// @Produce json
// @Param   database_name  query  string  true  "Database Name"
// @Param   id             query  int     true  "Entry ID"
// @Success 200 {object} models.Entry "The full entry metadata object"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 404 {object} ErrorResponse "Database or entry not found"
// @Security BasicAuth
// @Router /entry/meta [get]
func (h *Handlers) GetEntryMeta(w http.ResponseWriter, r *http.Request) {
	// DISABLE CACHING
	// This is critical for the frontend poller to work correctly on remote servers.
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	dbName := r.URL.Query().Get("database_name")
	if dbName == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required query parameter: database_name")
		return
	}
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID format.")
		return
	}

	// --- Call DatabaseService (GetDatabase) ---
	db, err := h.Database.GetDatabase(dbName)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Database not found.")
		return
	}

	// --- Call EntryService (GetEntry) ---
	entry, err := h.Entry.GetEntry(dbName, id, db.CustomFields)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Entry not found.")
		return
	}

	respondWithJSON(w, http.StatusOK, entry)
}

// @Summary Update entry metadata
// @Description Updates an entry's mutable metadata, including custom fields and the 'filename'.
// @Tags entry
// @Accept json
// @Produce json
// @Param   database_name  query  string  true  "Database Name"
// @Param   id             query  int     true  "Entry ID"
// @Param   updates        body   models.Entry  true  "JSON object with fields to update"
// @Success 200 {object} models.Entry "The full, updated entry metadata object"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 404 {object} ErrorResponse "Database or entry not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /entry [patch]
func (h *Handlers) UpdateEntry(w http.ResponseWriter, r *http.Request) {
	dbName := r.URL.Query().Get("database_name")
	if dbName == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required query parameter: database_name")
		return
	}
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID format.")
		return
	}

	var updates models.Entry
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload.")
		return
	}

	// The service now handles sanitizing the update map
	finalEntry, err := h.Entry.UpdateEntry(dbName, id, updates)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			respondWithError(w, http.StatusNotFound, "Database or entry not found.")
		} else {
			respondWithError(w, http.StatusInternalServerError, "Failed to update entry.")
		}
		return
	}

	respondWithJSON(w, http.StatusOK, finalEntry)
}
