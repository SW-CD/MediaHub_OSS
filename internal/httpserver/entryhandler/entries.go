package entryhandler

import (
	"archive/zip"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mediahub_oss/internal/httpserver/utils"
	repo "mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared"
	"mediahub_oss/internal/shared/customerrors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// @Summary Upload an entry
// @Description Uploads a new entry to a specified database using multipart/form-data. The metadata part should be a JSON object containing the entry's timestamp, and any custom fields.
// @Description The 'file' part's 'filename' in the Content-Disposition header will be extracted and saved.
// @Description
// @Description This endpoint uses a hybrid model:
// @Description - **Small files (<= Configured Limit):** Processed synchronously. Returns `201 Created` with the full entry metadata.
// @Description - **Large files (> Configured Limit):** Processed asynchronously. Returns `202 Accepted` with a partial response. The client should poll `GET /api/entry/meta` until the `status` field is 'ready'.
// @Tags entry
// @Accept  mpfd
// @Produce  json
// @Param   database_id  path  string  true  "Database ID"
// @Param   metadata      formData  string  true  "JSON metadata for the entry"
// @Param   file          formData  file    true  "Entry file"
// @Success 201 {object} EntryResponse "For small files (synchronous processing)"
// @Success 202 {object} PartialEntryResponse "For large files (asynchronous processing)"
// @Failure 400 {object} utils.ErrorResponse "Invalid request"
// @Failure 404 {object} utils.ErrorResponse "Database not found"
// @Failure 415 {object} utils.ErrorResponse "Unsupported entry format"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /database/{database_id}/entry [post]
func (h *EntryHandler) PostEntry(w http.ResponseWriter, r *http.Request) {

	dbID := r.PathValue("database_id")
	if dbID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Missing required path parameter: database_id")
		return
	}

	// Get user and db
	user := utils.GetUserFromContext(r.Context())
	if user == nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "User not found")
		return
	}
	db, err := h.Repo.GetDatabase(r.Context(), dbID)
	if err != nil {
		if errors.Is(err, customerrors.ErrNotFound) {
			utils.RespondWithError(w, http.StatusNotFound, "Database not found.")
		} else {
			h.Logger.Error("Failed to fetch database", "database_id", dbID, "error", err)
			utils.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database. Error: %v", err))
		}
		return
	}

	// Read file into memory or store it on the file system
	maxMemory := h.MaxSyncUploadSizeBytes
	if maxMemory <= 0 {
		maxMemory = 8 << 20
	}

	if err := r.ParseMultipartForm(maxMemory); err != nil {
		h.Logger.Warn("Failed to parse multipart form", "error", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Failed to parse multipart form.")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Missing 'file' part in multipart form.")
		return
	}
	defer file.Close()

	// Parse and validate metadata
	metadataStr := r.FormValue("metadata")
	if metadataStr == "" {
		h.Logger.Warn("Missing 'metadata' part in multipart form")
		utils.RespondWithError(w, http.StatusBadRequest, "Missing 'metadata' part in multipart form.")
		return
	}

	entry_request, err := parseUploadMetadata(metadataStr)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Error parsing file metadata: "+err.Error())
		return
	}

	err = validateCustomFields(entry_request.CustomFields, db.CustomFields)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Error validating custom fields: "+err.Error())
		return
	}

	// Create entry in the database and store file in the storage
	entry, wasSync, err := h.createEntryWithFile(r.Context(), db, entry_request, file, header)

	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	status := 201
	if !wasSync {
		status = 202
	}

	// Audit & Response
	h.Auditor.Log(r.Context(), "entry.post", user.Username, fmt.Sprintf("%s:%d", dbID, entry.GetID()), map[string]any{"database_name": db.Name})

	utils.RespondWithJSON(w, status, entry)
}

// @Summary Delete an entry
// @Description Deletes an entry file from disk and its metadata from the database.
// @Tags entry
// @Produce json
// @Param   database_id  path  string  true  "Database ID"
// @Param   id      path  int     true  "Entry ID"
// @Success 200 {object} utils.MessageResponse "Success message"
// @Failure 400 {object} utils.ErrorResponse "Invalid request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden"
// @Failure 404 {object} utils.ErrorResponse "Database or entry not found"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /database/{database_id}/entry/{id} [delete]
func (h *EntryHandler) DeleteEntry(w http.ResponseWriter, r *http.Request) {

	// 1. Validate Inputs
	dbID := r.PathValue("database_id")
	if dbID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Missing required path parameter: database_id")
		return
	}
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid ID format.")
		return
	}

	user := utils.GetUserFromContext(r.Context())
	if user == nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "User not found")
		return
	}

	// 2. Delete using the Safe 2-Phase Approach
	_, err = shared.DeleteSafe(r.Context(), h.Repo, h.Storage, dbID, id)
	if err != nil {
		if errors.Is(err, customerrors.ErrNotFound) {
			utils.RespondWithError(w, http.StatusNotFound, "Database or entry not found.")
		} else {
			h.Logger.Error("Failed to safely delete entry", "database_id", dbID, "id", id, "error", err)
			utils.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete the entry data. Error: %v", err))
		}
		return
	}

	// 3. Audit & Response
	h.Auditor.Log(r.Context(), "entry.delete", user.Username, fmt.Sprintf("%s:%d", dbID, id), nil)

	h.Logger.Info("Entry deleted", "id", idStr, "database_id", dbID)
	utils.RespondWithJSON(w, http.StatusOK, utils.MessageResponse{Message: fmt.Sprintf("Entry '%s' from database '%s' was successfully deleted.", idStr, dbID)})
}

// @Summary Get an entry file
// @Description Retrieves a raw entry file. Supports Content Negotiation (JSON vs Binary) and HTTP Range Requests (Streaming).
// @Tags entry
// @Produce octet-stream
// @Produce json
// @Param   database_id  path    string  true  "Database ID"
// @Param   id      path    int64   true  "Entry ID"
// @Param   Range   header  string  false "Byte range request (e.g., bytes=0-1023)"
// @Success 200 {file} file "The full raw file data (default)"
// @Success 200 {object} FileJSONResponse "Base64 encoded file data (if Accept: application/json)"
// @Success 206 {file} file "Partial content (streaming response)"
// @Failure 400 {object} utils.ErrorResponse "Invalid request or ID format"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden"
// @Failure 404 {object} utils.ErrorResponse "Database or entry not found"
// @Failure 409 {object} utils.ErrorResponse "File is currently processing"
// @Failure 416 {object} utils.ErrorResponse "Range Not Satisfiable"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Header 200,206 {string} Accept-Ranges "bytes"
// @Header 206 {string} Content-Range "bytes start-end/total"
// @Security BasicAuth
// @Security BearerAuth
// @Router /database/{database_id}/entry/{id}/file [get]
func (h *EntryHandler) GetEntryFile(w http.ResponseWriter, r *http.Request) {
	dbID := r.PathValue("database_id")
	idStr := r.PathValue("id")
	user := utils.GetUserFromContext(r.Context())
	if user == nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "User not found")
		return
	}

	// 1. Validate Input
	if dbID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Missing required path parameter: database_id")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid ID format.")
		return
	}

	// 2. Get Metadata (Crucial for File Size)
	filemeta, err := h.Repo.GetEntry(r.Context(), dbID, id)
	if err != nil {
		if errors.Is(err, customerrors.ErrNotFound) {
			utils.RespondWithError(w, http.StatusNotFound, "Database or entry not found.")
		} else {
			utils.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get entry metadata. Error: %v", err))
		}
		return
	}

	// Do not serve files that are still processing!
	if filemeta.Status == repo.EntryStatusProcessing {
		utils.RespondWithError(w, http.StatusConflict, "File is currently being processed. Try again later.")
		return
	}

	// Case A: JSON / Base64 Response
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		// Read full file (offset 0, length -1)
		fileStream, err := h.Storage.Read(r.Context(), dbID, filemeta.ID, 0, -1)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "File content not found.")
			return
		}
		defer fileStream.Close()

		if filemeta.FileName == "" {
			filemeta.FileName = fmt.Sprintf("%d", id)
		}

		resp, err := encodeReaderAsJSON(fileStream, filemeta.FileName, filemeta.MimeType)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to encode file. Error: %v", err))
			return
		}
		utils.RespondWithJSON(w, http.StatusOK, resp)
		return
	}

	// Determine Range (Streaming vs Full)
	rangeHeader := r.Header.Get("Range")
	fileSize := int64(filemeta.Size)

	var offset int64 = 0
	var length int64 = -1 // Read to end
	isPartial := false

	if rangeHeader != "" {
		// Simple parser for "bytes=start-end"
		ranges, err := parseRange(rangeHeader, fileSize)
		if err != nil {
			// 416 Range Not Satisfiable
			w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", fileSize))
			utils.RespondWithError(w, http.StatusRequestedRangeNotSatisfiable, "Invalid Range Header")
			return
		}

		// We only support the first range requested (multipart ranges are rare for this use case)
		if len(ranges) > 0 {
			isPartial = true
			offset = ranges[0].start
			length = ranges[0].length
		}
	}

	// 3. Open Stream (Partial or Full)
	fileStream, err := h.Storage.Read(r.Context(), dbID, filemeta.ID, offset, length)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "File content not found.")
		return
	}
	defer fileStream.Close()

	// 4. Set Response Headers
	w.Header().Set("Content-Type", filemeta.MimeType)
	w.Header().Set("Accept-Ranges", "bytes") // Advertise support

	if isPartial {
		// Case B: 206 Partial Content
		end := offset + length - 1
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", offset, end, fileSize))
		w.Header().Set("Content-Length", strconv.FormatInt(length, 10))

		// Spec: "inline" allows playback
		if filemeta.FileName != "" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filemeta.FileName))
		}
		w.WriteHeader(http.StatusPartialContent)

	} else {
		// Case C: 200 OK (Full Download)
		w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))

		if filemeta.FileName != "" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filemeta.FileName))
		}
		w.WriteHeader(http.StatusOK)
	}

	// Auditor logging
	h.Auditor.Log(r.Context(), "entry.download", user.Username, fmt.Sprintf("%s:%d", dbID, id), nil)

	// 5. Stream Data
	_, err = io.Copy(w, fileStream)
	if err != nil {
		// Stream interrupted
		return
	}
}

// @Summary Get entry metadata
// @Description Retrieves all metadata for a single entry, including custom fields.
// @Tags entry
// @Produce json
// @Param   database_id  path  string  true  "Database ID"
// @Param   id      path  int64   true  "Entry ID"
// @Success 200 {object} EntryResponse "The full entry metadata object"
// @Failure 400 {object} utils.ErrorResponse "Invalid request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden"
// @Failure 404 {object} utils.ErrorResponse "Database or entry not found"
// @Security BasicAuth
// @Router /database/{database_id}/entry/{id} [get]
func (h *EntryHandler) GetEntryMeta(w http.ResponseWriter, r *http.Request) {
	dbID := r.PathValue("database_id")
	idStr := r.PathValue("id")
	user := utils.GetUserFromContext(r.Context())
	if user == nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "User not found")
		return
	}

	// 1. Validate Input
	if dbID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Missing required path parameter: database_id")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid ID format.")
		return
	}

	// 2. Get Metadata from Database
	filemeta, err := h.Repo.GetEntry(r.Context(), dbID, id)
	if err != nil {
		if errors.Is(err, customerrors.ErrNotFound) {
			utils.RespondWithError(w, http.StatusNotFound, "Database or entry not found.")
		} else {
			h.Logger.Error("Failed to get entry metadata", "entry", id, "error", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get entry metadata.")
		}
		return
	}

	// 3. Map to API Response Model!
	responseObject := mapToEntryResponse(dbID, filemeta)

	// 4. Set anti-caching headers before sending the JSON
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// 5. Auditor logging
	h.Auditor.Log(r.Context(), "entry.read_meta", user.Username, fmt.Sprintf("%s:%d", dbID, id), nil)

	// 6. Return the mapped response
	utils.RespondWithJSON(w, http.StatusOK, responseObject)
}

// @Summary Get an entry preview
// @Description Retrieves a 200x200 WebP preview of an entry. Supports Content Negotiation via Accept header.
// @Tags entry
// @Produce image/webp
// @Produce json
// @Param   database_id   path   string   true  "Database ID"
// @Param   id       path   int64    true  "Entry ID"
// @Success 200 {file} file "The WebP preview image (default)"
// @Success 200 {object} FileJSONResponse "Base64 encoded preview data (if Accept: application/json)"
// @Failure 400 {object} utils.ErrorResponse "Invalid request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden"
// @Failure 404 {object} utils.ErrorResponse "Database, entry, or preview not found"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /database/{database_id}/entry/{id}/preview [get]
func (h *EntryHandler) GetEntryPreview(w http.ResponseWriter, r *http.Request) {
	dbID := r.PathValue("database_id")
	idStr := r.PathValue("id")

	// 1. Validate Input
	if dbID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Missing required path parameter: database_id")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid ID format.")
		return
	}

	// 2. Read the preview file from storage
	ioReader, err := h.Storage.ReadPreview(r.Context(), dbID, id)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Preview not found")
		return
	}
	defer ioReader.Close()

	// 3. Content Negotiation: Check if the client specifically requested JSON
	acceptHeader := r.Header.Get("Accept")
	if strings.Contains(acceptHeader, "application/json") {
		// Read the binary data into memory
		previewBytes, err := io.ReadAll(ioReader)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to read preview data")
			return
		}

		// Convert to Base64 and format as a Data URI
		base64Data := base64.StdEncoding.EncodeToString(previewBytes)
		dataURI := "data:image/webp;base64," + base64Data

		// Construct and return the JSON response
		jsonResp := FileJSONResponse{
			Filename: fmt.Sprintf("%d_preview.webp", id),
			MimeType: "image/webp",
			Data:     dataURI,
		}

		utils.RespondWithJSON(w, http.StatusOK, jsonResp)
		return
	}

	// 5. Default Response: Stream the raw binary image
	w.Header().Set("Content-Type", "image/webp")
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, ioReader); err != nil {
		h.Logger.Error("Failed to stream preview to client", "entry", id, "error", err)
	}
}

// @Summary Update entry metadata
// @Description Updates an entry's mutable metadata, including custom fields, the 'timestamp' and the 'filename'.
// @Tags entry
// @Accept json
// @Produce json
// @Param   database_id   path   string                true  "Database ID"
// @Param   id       path   int64                 true  "Entry ID"
// @Param   updates  body   PostPatchEntryRequest  true  "JSON object with fields to update"
// @Success 200 {object} EntryResponse "The full, updated entry metadata object"
// @Failure 400 {object} utils.ErrorResponse "Invalid request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden"
// @Failure 404 {object} utils.ErrorResponse "Database or entry not found"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /database/{database_id}/entry/{id} [patch]
func (h *EntryHandler) PatchEntry(w http.ResponseWriter, r *http.Request) {
	dbID := r.PathValue("database_id")
	idStr := r.PathValue("id")

	// 1. Validate Path Parameters
	if dbID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Missing required path parameter: database_id")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid ID format.")
		return
	}

	user := utils.GetUserFromContext(r.Context())
	if user == nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "User not found")
		return
	}

	// 2. Decode the PATCH Request Body
	var req = PostPatchEntryRequest{
		FileName:     "",
		Timestamp:    math.MinInt64,
		CustomFields: nil,
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	defer r.Body.Close()

	// 3. Fetch the Existing Entry and Database
	db, err := h.Repo.GetDatabase(r.Context(), dbID)
	if err != nil {
		if errors.Is(err, customerrors.ErrRepoUnavailable) {
			utils.RespondWithError(w, http.StatusInternalServerError, "Connection to repository failed.")
			h.Logger.Error("Failed to connect to repository", "error", err)
			return
		} else if errors.Is(err, customerrors.ErrDatabaseNotExisting) {
			utils.RespondWithError(w, http.StatusNotFound, fmt.Sprintf("Database with ID %s does not exist.", dbID))
			return
		} else {
			utils.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error fetching database: %v", err))
			h.Logger.Error("Failed to fetch database for update", "database_id", dbID, "error", err)
			return
		}
	}

	existingEntry, err := h.Repo.GetEntry(r.Context(), dbID, id)
	if err != nil {
		if errors.Is(err, customerrors.ErrNotFound) {
			utils.RespondWithError(w, http.StatusNotFound, "Database or entry not found.")
		} else {
			h.Logger.Error("Failed to fetch entry for update", "entry", id, "error", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve entry.")
		}
		return
	}

	// 4. Apply Updates Safely (Ignoring Go zero-values)

	// Only update if the string is not empty
	if req.FileName != "" {
		existingEntry.FileName = req.FileName
	}

	// Only update the timestamp if it was provided
	if req.Timestamp != math.MinInt64 {
		existingEntry.Timestamp = time.UnixMilli(req.Timestamp)
	}

	// Merge Custom Fields after validation
	if req.CustomFields != nil {
		err = validateCustomFields(req.CustomFields, db.CustomFields)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Error during custom field validation: "+err.Error())
			return
		}

		if existingEntry.CustomFields == nil {
			existingEntry.CustomFields = make(map[string]any)
		}
		for key, value := range req.CustomFields {
			existingEntry.CustomFields[key] = value
		}
	}

	// 5. Save the Updated Entry back to the Database
	updatedEntry, err := h.Repo.UpdateEntry(r.Context(), dbID, existingEntry)
	if err != nil {
		h.Logger.Error("Failed to update entry metadata", "entry", id, "error", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to apply updates to database.")
		return
	}

	// 6. Audit Logging
	h.Auditor.Log(r.Context(), "entry.update", user.Username, fmt.Sprintf("%s:%d", dbID, id), nil)

	// 7. Map to API Response Model and Return
	responseObject := mapToEntryResponse(dbID, updatedEntry)
	utils.RespondWithJSON(w, http.StatusOK, responseObject)
}

// @Summary Bulk delete entries
// @Description Deletes multiple entries in a single atomic transaction. Updates database statistics only once.
// @Tags database
// @Accept  json
// @Produce json
// @Param   database_id  path   string  true  "Database ID"
// @Param   body    body   BulkDeleteRequest true "JSON object containing a list of Entry IDs to delete"
// @Success 200 {object} BulkDeleteResponse "Summary of the deletion operation"
// @Failure 400 {object} utils.ErrorResponse "Invalid request, missing id, or empty IDs list"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden (Requires CanDelete role)"
// @Failure 404 {object} utils.ErrorResponse "Database not found"
// @Failure 500 {object} utils.ErrorResponse "Transaction failed"
// @Security BasicAuth
// @Router /database/{database_id}/entries/delete [post]
func (h *EntryHandler) DeleteEntries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dbID := r.PathValue("database_id")
	user := utils.GetUserFromContext(r.Context())
	if user == nil {
		h.Logger.Error("User not found in context")
		utils.RespondWithError(w, http.StatusInternalServerError, "user not found in context")
		return
	}

	var req BulkDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request or empty IDs list")
		return
	}

	// 2. Delete the files and entries
	deletedMeta, err := shared.DeleteMultipleSafe(ctx, h.Repo, h.Storage, dbID, req.IDs)

	// 3. Calculate disk space freed
	var spaceFreed uint64 = 0
	var deletedCount = len(deletedMeta)
	for _, e := range deletedMeta {
		spaceFreed += e.Filesize + e.PreviewSize
	}

	// Safely extract the error message if one exists
	var errorMsg string
	if err != nil {
		errorMsg = err.Error()
	}

	// 4. Respond
	resp := BulkDeleteResponse{
		DatabaseID:      dbID,
		DeletedCount:    deletedCount,
		SpaceFreedBytes: spaceFreed,
		Message:         fmt.Sprintf("Successfully deleted %d entries.", deletedCount),
		Errors:          errorMsg, // Safe to use now!
	}

	// check for internal status or user errors
	status := http.StatusOK
	if err != nil {
		if errors.Is(err, customerrors.ErrRepoUnavailable) || errors.Is(err, customerrors.ErrStorageUnavailable) {
			status = http.StatusInternalServerError
		} else if errors.Is(err, customerrors.ErrDatabaseNotExisting) {
			status = http.StatusNotFound
		} else {
			// Optional: Fallback status for any other unexpected errors
			status = http.StatusInternalServerError
		}
	}

	h.Auditor.Log(r.Context(), "entries.delete", user.Username, dbID, map[string]any{"count": deletedCount})
	utils.RespondWithJSON(w, status, resp)
}

// @Summary Get entries from a database (basic)
// @Description Retrieves a paginated list of entries from a specific database. Only supports time-based filters.
// @Tags database
// @Produce json
// @Param   database_id  path   string  true   "Database ID"
// @Param   limit   query  int     false  "Number of entries to return (default 30)"
// @Param   offset  query  int     false  "Offset for pagination (default 0)"
// @Param   order   query  string  false  "Sort order ('asc' or 'desc', default 'desc')"
// @Param   tstart  query  int64   false  "Start timestamp (Unix milliseconds)"
// @Param   tend    query  int64   false  "End timestamp (Unix milliseconds)"
// @Success 200 {array} EntryResponse "Returns an array of entry metadata objects"
// @Failure 400 {object} utils.ErrorResponse "Missing id param or invalid parameter formats"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden (Requires CanView role)"
// @Failure 404 {object} utils.ErrorResponse "Database not found"
// @Failure 500 {object} utils.ErrorResponse "Failed to retrieve entries"
// @Security BasicAuth
// @Router /database/{database_id}/entries [get]
func (h *EntryHandler) QueryEntries(w http.ResponseWriter, r *http.Request) {
	dbID := r.PathValue("database_id")

	user := utils.GetUserFromContext(r.Context())
	if user == nil {
		h.Logger.Error("User not found in context")
		utils.RespondWithError(w, http.StatusInternalServerError, "user not found in context")
		return
	}

	limit := parseQueryInt(r, "limit", 30)
	offset := parseQueryInt(r, "offset", 0)

	order := r.URL.Query().Get("order")
	if order != "asc" {
		order = "desc" // Default to desc
	}

	var tStart, tEnd time.Time
	tStartQuery := parseQueryInt64(r, "tstart", math.MinInt64)
	if tStartQuery != math.MinInt64 {
		tStart = time.UnixMilli(tStartQuery)
	}
	tEndQuery := parseQueryInt64(r, "tend", math.MaxInt64)
	if tEndQuery != math.MaxInt64 {
		tEnd = time.UnixMilli(tEndQuery)
	}

	entries, err := h.Repo.GetEntries(r.Context(), dbID, limit, offset, order, tStart, tEnd)
	if err != nil {
		h.Logger.Error("Failed to query entries", "error", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve entries")
		return
	}

	// Map DB models to API responses
	results := make([]EntryResponse, 0, len(entries))
	for _, entry := range entries {
		results = append(results, mapToEntryResponse(dbID, entry))
	}

	h.Auditor.Log(r.Context(), "entries.query", user.Username, dbID, nil)
	utils.RespondWithJSON(w, http.StatusOK, results)
}

// @Summary Search for entries in a database (complex)
// @Description Retrieves a list of entry metadata matching the complex, nested filter criteria provided in the request body.
// @Tags database
// @Accept  json
// @Produce json
// @Param   database_id  path   string        true  "Database ID"
// @Param   search  body   repository.SearchRequest  true  "JSON body defining filter, sort, and pagination logic"
// @Success 200 {array} EntryResponse "Returns an array of matching results (even if empty)"
// @Failure 400 {object} utils.ErrorResponse "Missing id, invalid JSON, missing limit, or invalid filter/sort"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden (Requires CanView role)"
// @Failure 404 {object} utils.ErrorResponse "Database not found"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Security BasicAuth
// @Router /database/{database_id}/entries/search [post]
func (h *EntryHandler) SearchEntries(w http.ResponseWriter, r *http.Request) {
	dbID := r.PathValue("database_id")

	user := utils.GetUserFromContext(r.Context())
	if user == nil {
		h.Logger.Error("User not found in context")
		utils.RespondWithError(w, http.StatusInternalServerError, "user not found in context")
		return
	}

	var searchPayload SearchRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&searchPayload); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	// Fetch database to get custom fields for query validation
	db, err := h.Repo.GetDatabase(r.Context(), dbID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Database not found")
		return
	}

	searchReq := searchPayload.toModel()
	entries, err := h.Repo.SearchEntries(r.Context(), dbID, searchReq, db.CustomFields)
	if err != nil {
		h.Logger.Error("Search failed", "error", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Map DB models to API responses
	results := make([]EntryResponse, 0, len(entries))
	for _, entry := range entries {
		results = append(results, mapToEntryResponse(dbID, entry))
	}

	h.Auditor.Log(r.Context(), "entries.search", user.Username, dbID, nil)
	utils.RespondWithJSON(w, http.StatusOK, results)
}

// @Summary Export entries as ZIP
// @Description Streams a ZIP archive containing the files and metadata (CSV) for the specified entries using io.Pipe.
// @Tags database
// @Accept  json
// @Produce application/zip
// @Param   database_id  path   string        true  "Database ID"
// @Param   body    body   ExportRequest  true  "List of Entry IDs to export"
// @Success 200 {file} file "ZIP Archive containing files and entries.csv"
// @Failure 400 {object} utils.ErrorResponse "Missing id query parameter or empty IDs list"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden (Requires CanView role)"
// @Failure 404 {object} utils.ErrorResponse "Database not found"
// @Failure 500 {object} utils.ErrorResponse "ZIP streaming failed"
// @Security BasicAuth
// @Router /database/{database_id}/entries/export [post]
func (h *EntryHandler) ExportEntries(w http.ResponseWriter, r *http.Request) {
	dbID := r.PathValue("database_id")

	user := utils.GetUserFromContext(r.Context())
	if user == nil {
		h.Logger.Error("User not found in context")
		utils.RespondWithError(w, http.StatusInternalServerError, "user not found in context")
		return
	}

	var req ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request or empty IDs list")
		return
	}

	// Verify database existence and fetch custom fields
	db, err := h.Repo.GetDatabase(r.Context(), dbID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Database not found")
		return
	}

	// Set headers for ZIP download
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s_export.zip\"", db.Name))

	// Use io.Pipe to stream generation directly to the HTTP response
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		zipWriter := zip.NewWriter(pw)
		defer zipWriter.Close()

		// 1. Create CSV file inside ZIP
		csvFile, err := zipWriter.Create("entries.csv")
		if err != nil {
			h.Logger.Error("Failed to create CSV in zip", "error", err)
			pw.CloseWithError(err)
			return
		}

		csvWriter := csv.NewWriter(csvFile)

		// --- Build dynamic CSV Header ---
		header := []string{"id", "filename", "timestamp", "filesize", "previewsize", "mime_type", "status"}
		for _, cf := range db.CustomFields {
			header = append(header, cf.Name)
		}
		_ = csvWriter.Write(header)

		// Keep track of valid entries so we don't have to query the DB twice
		var validEntries []repo.Entry

		// Pass 1: Fetch metadata and write all CSV rows
		for _, id := range req.IDs {
			// Fetch metadata
			entry, err := h.Repo.GetEntry(r.Context(), dbID, id)
			if err != nil {
				h.Logger.Warn("Skipping entry in export (not found)", "id", id)
				continue
			}

			validEntries = append(validEntries, entry)

			// --- Build dynamic CSV Row ---
			row := []string{
				strconv.FormatInt(entry.ID, 10),
				entry.FileName,
				entry.Timestamp.Format(time.RFC3339),
				strconv.FormatUint(entry.Size, 10),
				strconv.FormatUint(entry.PreviewSize, 10),
				entry.MimeType,
				strconv.Itoa(int(entry.Status)),
			}

			// Append custom field values safely
			for _, cf := range db.CustomFields {
				val, exists := entry.CustomFields[cf.Name]
				if !exists || val == nil {
					row = append(row, "") // Empty column if no value
				} else {
					row = append(row, fmt.Sprintf("%v", val))
				}
			}

			_ = csvWriter.Write(row)
		}

		// Flush the CSV buffer to the zip file BEFORE creating new zip entries
		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			h.Logger.Error("Failed to flush CSV", "error", err)
		}

		// Pass 2: Stream the files into the ZIP
		for _, entry := range validEntries {
			// Fetch file stream from storage
			fileStream, err := h.Storage.Read(r.Context(), dbID, entry.ID, 0, -1)
			if err != nil {
				h.Logger.Warn("Failed to read file from storage for export", "id", entry.ID, "error", err)
				continue
			}

			// Create file inside ZIP (this would close the previous file, but we already flushed the CSV!)
			zipEntryPath := fmt.Sprintf("files/%d_%s", entry.ID, entry.FileName)
			zipFile, err := zipWriter.Create(zipEntryPath)
			if err != nil {
				fileStream.Close()
				continue
			}

			// Stream content into ZIP
			_, _ = io.Copy(zipFile, fileStream)
			fileStream.Close()
		}
	}()

	h.Auditor.Log(r.Context(), "entries.export", user.Username, dbID, map[string]any{"count": len(req.IDs)})

	// Stream the pipe reader directly to the response writer
	if _, err := io.Copy(w, pr); err != nil {
		h.Logger.Error("Failed to stream ZIP to client", "error", err)
	}
}
