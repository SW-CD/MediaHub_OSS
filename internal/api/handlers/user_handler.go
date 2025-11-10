// filepath: internal/api/handlers/user_handler.go
package handlers

import (
	"encoding/json"
	"mediahub/internal/logging"
	"mediahub/internal/models"
	"net/http"
)

// PasswordUpdateRequest is a DTO for updating a user's password.
type PasswordUpdateRequest struct {
	Password string `json:"password"`
}

// @Summary Get current user
// @Description Get the currently authenticated user's details.
// @Tags Users
// @Produce json
// @Success 200 {object} models.User
// @Failure 401 {object} ErrorResponse
// @Router /me [get]
// @Security BasicAuth
func (h *Handlers) GetUserMe(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "No user found in context")
		return
	}

	logging.Log.Debugf("GetUserMe: Handler started for user '%s' (ID: %d)", user.Username, user.ID)

	// Create a safe copy of the user to sanitize for the response.
	// This prevents accidentally modifying the user object in the cache.
	safeUser := *user
	safeUser.PasswordHash = "" // Remove the password hash from the response.

	respondWithJSON(w, http.StatusOK, safeUser)
}

// @Summary Update current user's password
// @Description Allows a user to change their own password.
// @Tags Users
// @Accept json
// @Produce json
// @Param password body PasswordUpdateRequest true "Password update request"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /me [patch]
// @Security BasicAuth
func (h *Handlers) UpdateUserMe(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "No user found in context")
		return
	}

	var req PasswordUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Password cannot be empty")
		return
	}

	logging.Log.Debugf("UpdateUserMe: User '%s' updating their password.", user.Username)

	// --- Call UserService ---
	if err := h.User.UpdateUserPassword(user.Username, req.Password); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	respondWithJSON(w, http.StatusOK, MessageResponse{Message: "Password updated successfully."})
}
