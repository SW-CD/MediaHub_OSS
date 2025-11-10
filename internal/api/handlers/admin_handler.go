// filepath: internal/api/handlers/admin_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"mediahub/internal/logging"
	"mediahub/internal/repository" // <-- IMPORT REPOSITORY
	"mediahub/internal/services"
	"net/http"
	"strconv"
)

// UserUpdateRequest is a DTO for updating a user's roles or password.
type UserUpdateRequest struct {
	CanView   *bool   `json:"can_view,omitempty"`
	CanCreate *bool   `json:"can_create,omitempty"`
	CanEdit   *bool   `json:"can_edit,omitempty"`
	CanDelete *bool   `json:"can_delete,omitempty"`
	IsAdmin   *bool   `json:"is_admin,omitempty"`
	Password  *string `json:"password,omitempty"`
}

// UserCreateRequest is a DTO for creating a new user.
type UserCreateRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password,omitempty"`
	CanView   bool   `json:"can_view"`
	CanCreate bool   `json:"can_create"`
	CanEdit   bool   `json:"can_edit"`
	CanDelete bool   `json:"can_delete"`
	IsAdmin   bool   `json:"is_admin"`
}

// @Summary Get all users
// @Description Retrieves a list of all users in the system.
// @Tags Admin
// @Produce json
// @Success 200 {array} models.User
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users [get]
// @Security BasicAuth
func (h *Handlers) GetUsers(w http.ResponseWriter, r *http.Request) {
	logging.Log.Debug("GetUsers: Handler started.")
	// --- Call UserService ---
	users, err := h.User.GetUsers()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get users")
		return
	}

	// Sanitize password hashes for the response.
	for i := range users {
		users[i].PasswordHash = ""
	}
	respondWithJSON(w, http.StatusOK, users)
}

// @Summary Create a new user
// @Description Creates a new user account.
// @Tags Admin
// @Accept json
// @Produce json
// @Param user body UserCreateRequest true "User creation request"
// @Success 201 {object} models.User
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /user [post]
// @Security BasicAuth
func (h *Handlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req UserCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// --- Map DTO to Service Args ---
	userArgs := repository.UserCreateArgs{
		Username:  req.Username,
		Password:  req.Password,
		CanView:   req.CanView,
		CanCreate: req.CanCreate,
		CanEdit:   req.CanEdit,
		CanDelete: req.CanDelete,
		IsAdmin:   req.IsAdmin,
	}

	logging.Log.Debugf("CreateUser: Handler: Calling UserService for '%s'", req.Username)
	createdUser, err := h.User.CreateUser(userArgs)
	if err != nil {
		if errors.Is(err, repository.ErrUserExists) {
			respondWithError(w, http.StatusConflict, "Username already exists")
		} else if errors.Is(err, services.ErrValidation) {
			respondWithError(w, http.StatusBadRequest, err.Error())
		} else {
			respondWithError(w, http.StatusInternalServerError, "Failed to create user")
		}
		return
	}

	createdUser.PasswordHash = "" // Sanitize password hash for the response
	respondWithJSON(w, http.StatusCreated, createdUser)
}

// @Summary Update a user's roles or password
// @Description Updates an existing user's roles or password.
// @Tags Admin
// @Accept json
// @Produce json
// @Param id query int true "User ID"
// @Param user body UserUpdateRequest true "User update request"
// @Success 200 {object} models.User
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /user [patch]
// @Security BasicAuth
func (h *Handlers) UpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req UserUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	logging.Log.Debugf("UpdateUser: Handler started for user ID %d", id)

	// --- Call UserService ---
	// We must get the original user first to apply defaults
	originalUser, err := h.User.GetUserByID(id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Create a new model object for the update
	userToUpdate := *originalUser // Start with original values

	// Apply updates from the request
	if req.CanView != nil {
		userToUpdate.CanView = *req.CanView
	}
	if req.CanCreate != nil {
		userToUpdate.CanCreate = *req.CanCreate
	}
	if req.CanEdit != nil {
		userToUpdate.CanEdit = *req.CanEdit
	}
	if req.CanDelete != nil {
		userToUpdate.CanDelete = *req.CanDelete
	}
	if req.IsAdmin != nil {
		userToUpdate.IsAdmin = *req.IsAdmin
	}

	// Pass the updates to the service
	// The service will handle the "last admin" logic
	finalUser, err := h.User.UpdateUser(id, userToUpdate, req.Password)
	if err != nil {
		if err.Error() == "user not found" {
			respondWithError(w, http.StatusNotFound, err.Error())
		} else if err.Error() == "cannot remove the last admin's admin role" {
			respondWithError(w, http.StatusConflict, err.Error())
		} else {
			respondWithError(w, http.StatusInternalServerError, "Failed to update user")
		}
		return
	}

	// Create a safe copy for the response to avoid leaking the hash.
	safeUser := *finalUser
	safeUser.PasswordHash = ""

	respondWithJSON(w, http.StatusOK, safeUser)
}

// @Summary Delete a user
// @Description Deletes a user account.
// @Tags Admin
// @Produce json
// @Param id query int true "User ID"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /user [delete]
// @Security BasicAuth
func (h *Handlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	logging.Log.Debugf("DeleteUser: Handler: Calling UserService for ID %d", id)

	// --- Call UserService ---
	// The service now handles the "last admin" check
	if err := h.User.DeleteUser(id); err != nil {
		if err.Error() == "user not found" {
			respondWithError(w, http.StatusNotFound, err.Error())
		} else if err.Error() == "cannot delete the last admin user" {
			respondWithError(w, http.StatusConflict, err.Error())
		} else {
			respondWithError(w, http.StatusInternalServerError, "Failed to delete user")
		}
		return
	}

	respondWithJSON(w, http.StatusOK, MessageResponse{Message: "User deleted successfully."})
}
