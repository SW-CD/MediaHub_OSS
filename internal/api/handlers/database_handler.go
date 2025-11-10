// filepath: internal/api/handlers/database_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"mediahub/internal/logging"
	"mediahub/internal/models"
	"mediahub/internal/services"
	"net/http"
	"strings"
)

// @Summary Create a new database
// @Description Creates a new database with custom fields and a dedicated entry table.
// @Tags database
// @Accept  json
// @Produce  json
// @Param   database  body  models.DatabaseCreatePayload  true  "Database Metadata"
// @Success 201 {object} models.Database
// @Failure 400 {object} ErrorResponse "Invalid request payload or missing name"
// @Failure 409 {object} ErrorResponse "Database name already in use"
// @Failure 500 {object} ErrorResponse "Failed to create database or storage folder"
// @Security BasicAuth
// @Router /database [post]
func (h *Handlers) CreateDatabase(w http.ResponseWriter, r *http.Request) {
	// --- Use payload from models package ---
	var payload models.DatabaseCreatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		logging.Log.Warnf("Failed to decode request body: %v", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if payload.Name == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required field: name")
		return
	}
	if payload.ContentType == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required field: content_type")
		return
	}

	// --- Call service with the payload ---
	createdDB, err := h.Database.CreateDatabase(payload)
	if err != nil {
		if errors.Is(err, services.ErrDependencies) {
			// e.g., FFmpeg check failed
			respondWithError(w, http.StatusBadRequest, err.Error())
		} else if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			// This error comes from the repository
			respondWithError(w, http.StatusConflict, "Database name already in use.")
		} else if strings.Contains(err.Error(), "invalid database name") {
			respondWithError(w, http.StatusBadRequest, err.Error())
		} else {
			logging.Log.Errorf("Failed to create database: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to create database.")
		}
		return
	}

	logging.Log.Infof("Database created successfully: %s", createdDB.Name)
	respondWithJSON(w, http.StatusCreated, createdDB)
}

// @Summary Get database details
// @Description Retrieves details, custom fields, and statistics for a specific database.
// @Tags database
// @Produce  json
// @Param   name  query  string  true  "Database Name"
// @Success 200 {object} models.Database
// @Failure 400 {object} ErrorResponse "Missing name parameter"
// @Failure 404 {object} ErrorResponse "Database not found"
// @Failure 500 {object} ErrorResponse "Failed to calculate stats"
// @Security BasicAuth
// @Router /database [get]
func (h *Handlers) GetDatabase(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required query parameter: name")
		return
	}

	// --- Call DatabaseService ---
	db, err := h.Database.GetDatabase(name)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Database not found.")
		return
	}

	respondWithJSON(w, http.StatusOK, db)
}

// @Summary List all databases
// @Description Retrieves a list of all available databases and their statistics.
// @Tags database
// @Produce  json
// @Success 200 {array} models.Database "Returns an empty array if no databases exist"
// @Failure 500 {object} ErrorResponse "Failed to retrieve databases"
// @Security BasicAuth
// @Router /databases [get]
func (h *Handlers) GetDatabases(w http.ResponseWriter, r *http.Request) {
	// --- Call DatabaseService ---
	dbs, err := h.Database.GetDatabases()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve databases.")
		return
	}

	// Ensure an empty array `[]` is returned instead of `null` if no databases exist
	if dbs == nil {
		dbs = []models.Database{}
	}

	respondWithJSON(w, http.StatusOK, dbs)
}

// @Summary Update database housekeeping rules
// @Description Updates the housekeeping rules for a specific database.
// @Tags database
// @Accept  json
// @Produce  json
// @Param   name  query  string  true  "Database Name"
// @Param   housekeeping  body  models.DatabaseUpdatePayload  true  "Housekeeping Rules or Config flag"
// @Success 200 {object} models.Database
// @Failure 400 {object} ErrorResponse "Invalid request payload or missing name"
// @Failure 404 {object} ErrorResponse "Database not found"
// @Failure 500 {object} ErrorResponse "Failed to update database"
// @Security BasicAuth
// @Router /database [put]
func (h *Handlers) UpdateDatabase(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required query parameter: name")
		return
	}

	// --- Use payload from models package ---
	var updates models.DatabaseUpdatePayload
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// --- Call service with the payload ---
	updatedDB, err := h.Database.UpdateDatabase(name, updates)
	if err != nil {
		if errors.Is(err, services.ErrDependencies) {
			// e.g., FFmpeg check failed
			respondWithError(w, http.StatusBadRequest, err.Error())
		} else if err.Error() == "database not found" {
			respondWithError(w, http.StatusNotFound, err.Error())
		} else {
			respondWithError(w, http.StatusInternalServerError, "Failed to update database.")
		}
		return
	}

	// Return the full updated database object
	respondWithJSON(w, http.StatusOK, updatedDB)
}

// @Summary Delete a database
// @Description Deletes a database, its entry table, and all of its associated entries and metadata.
// @Tags database
// @Produce  json
// @Param   name  query  string  true  "Database Name"
// @Success 200 {object} MessageResponse "Success message"
// @Failure 400 {object} ErrorResponse "Missing name parameter or invalid name"
// @Failure 404 {object} ErrorResponse "Database not found"
// @Failure 500 {object} ErrorResponse "Failed to delete database record or folder"
// @Security BasicAuth
// @Router /database [delete]
func (h *Handlers) DeleteDatabase(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required query parameter: name")
		return
	}

	// --- Call DatabaseService ---
	if err := h.Database.DeleteDatabase(name); err != nil {
		// Differentiate between "not found" and other errors
		if strings.Contains(err.Error(), "not found") { // Improve error checking if possible
			respondWithError(w, http.StatusNotFound, "Database not found.")
		} else if strings.Contains(err.Error(), "invalid database name") {
			respondWithError(w, http.StatusBadRequest, "Invalid database name.")
		} else {
			respondWithError(w, http.StatusInternalServerError, "Failed to delete database record.")
		}
		return
	}

	logging.Log.Infof("Database deleted successfully: %s", name)
	respondWithJSON(w, http.StatusOK, MessageResponse{
		Message: "Database '" + name + "' and all its contents were successfully deleted.",
	})
}
