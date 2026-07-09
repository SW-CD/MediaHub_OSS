package userhandler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mediahub_oss/internal/httpserver/utils"
	repo "mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared"
	"mediahub_oss/internal/shared/customerrors"
	"net/http"
	"time"
)

// APIKeyResponse is the standard metadata response.
type APIKeyResponse struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	KeyHint     string `json:"key_hint"`
	ScopeView   bool   `json:"scope_view"`
	ScopeCreate bool   `json:"scope_create"`
	ScopeEdit   bool   `json:"scope_edit"`
	ScopeDelete bool   `json:"scope_delete"`
	ScopeAdmin  bool   `json:"scope_admin"`
	CreatedAt   int64  `json:"created_at"`
	ExpiresAt   *int64 `json:"expires_at"`   // nullable
	LastUsedAt  *int64 `json:"last_used_at"`  // nullable
}

// APIKeyCreatedResponse includes the generated plaintext token.
type APIKeyCreatedResponse struct {
	APIKeyResponse
	Token string `json:"token"`
}

// APIKeyWithUserResponse is returned by the global admin keys endpoint.
type APIKeyWithUserResponse struct {
	APIKeyResponse
	User UserSubResponse `json:"user"`
}

type UserSubResponse struct {
	ID               string `json:"id"`
	Username         string `json:"username"`
	IsAdmin          bool   `json:"is_admin"`
	IsServiceAccount bool   `json:"is_service_account"`
}

type CreateAPIKeyPayload struct {
	Name        string `json:"name"`
	ExpiresAt   *int64 `json:"expires_at"`
	ScopeView   bool   `json:"scope_view"`
	ScopeCreate bool   `json:"scope_create"`
	ScopeEdit   bool   `json:"scope_edit"`
	ScopeDelete bool   `json:"scope_delete"`
	ScopeAdmin  bool   `json:"scope_admin"`
}

type UpdateAPIKeyPayload struct {
	Name        *string `json:"name"`
	ExpiresAt   *int64  `json:"expires_at"`
	ScopeView   *bool   `json:"scope_view"`
	ScopeCreate *bool   `json:"scope_create"`
	ScopeEdit   *bool   `json:"scope_edit"`
	ScopeDelete *bool   `json:"scope_delete"`
	ScopeAdmin  *bool   `json:"scope_admin"`
}

func mapToAPIKeyResponse(key repo.APIKey) APIKeyResponse {
	var expiresAt *int64
	if !key.ExpiresAt.IsZero() {
		val := key.ExpiresAt.UnixMilli()
		expiresAt = &val
	}

	var lastUsedAt *int64
	if !key.LastUsedAt.IsZero() {
		val := key.LastUsedAt.UnixMilli()
		lastUsedAt = &val
	}

	return APIKeyResponse{
		ID:          string(key.ID),
		UserID:      string(key.UserID),
		Name:        key.Name,
		KeyHint:     key.KeyHint,
		ScopeView:   key.ScopeView,
		ScopeCreate: key.ScopeCreate,
		ScopeEdit:   key.ScopeEdit,
		ScopeDelete: key.ScopeDelete,
		ScopeAdmin:  key.ScopeAdmin,
		CreatedAt:   key.CreatedAt.UnixMilli(),
		ExpiresAt:   expiresAt,
		LastUsedAt:  lastUsedAt,
	}
}

// GetAllAPIKeys godoc
// @Summary      Retrieve all API keys
// @Description  Retrieves all active API keys in the system with associated user details. Requires global admin role.
// @Tags         User
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   APIKeyWithUserResponse
// @Failure      401  {object}  utils.ErrorResponse "Authentication failed"
// @Failure      403  {object}  utils.ErrorResponse "Forbidden"
// @Router       /users/keys [get]
func (h *UserHandler) GetAllAPIKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	keys, err := h.Repo.GetAllAPIKeys(ctx)
	if err != nil {
		h.Logger.Error("Failed to retrieve all API keys", "error", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	users, err := h.Repo.GetUsers(ctx, nil)
	if err != nil {
		h.Logger.Error("Failed to retrieve users", "error", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	userMap := make(map[repo.ULID]repo.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	resp := make([]APIKeyWithUserResponse, 0, len(keys))
	for _, key := range keys {
		u, ok := userMap[key.UserID]
		if !ok {
			u, err = h.Repo.GetUserByID(ctx, key.UserID)
			if err != nil {
				continue
			}
		}
		resp = append(resp, APIKeyWithUserResponse{
			APIKeyResponse: mapToAPIKeyResponse(key),
			User: UserSubResponse{
				ID:               string(u.ID),
				Username:         u.Username,
				IsAdmin:          u.IsAdmin,
				IsServiceAccount: u.IsServiceAccount,
			},
		})
	}

	utils.RespondWithJSON(w, http.StatusOK, resp)
}

// CreateAPIKey godoc
// @Summary      Create a new API key
// @Description  Generates a new API key for the specified user and returns the plaintext token. Plentext token is returned only once. Requires admin or self ownership.
// @Tags         User
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        user_ulid path string true "User ULID"
// @Param        payload body CreateAPIKeyPayload true "API Key details"
// @Success      201  {object}  APIKeyCreatedResponse
// @Failure      400  {object}  utils.ErrorResponse "Bad Request"
// @Failure      401  {object}  utils.ErrorResponse "Unauthorized"
// @Failure      403  {object}  utils.ErrorResponse "Forbidden"
// @Failure      404  {object}  utils.ErrorResponse "User not found"
// @Router       /user/{user_ulid}/keys [post]
func (h *UserHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userIDStr := r.PathValue("user_ulid")
	if userIDStr == "" {
		userIDStr = r.PathValue("user_id")
	}

	user, err := h.Repo.GetUserByID(ctx, repo.ULID(userIDStr))
	if err != nil {
		if errors.Is(err, customerrors.ErrNotFound) {
			utils.RespondWithError(w, http.StatusNotFound, "User not found")
		} else {
			h.Logger.Error("Failed to retrieve user", "error", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	var payload CreateAPIKeyPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if payload.Name == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Name is required")
		return
	}

	// Generate 16 bytes of randomness for the secret part (32 hex characters)
	secretBytes := make([]byte, 16)
	if _, err := rand.Read(secretBytes); err != nil {
		h.Logger.Error("Failed to generate secure random bytes", "error", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	secret := hex.EncodeToString(secretBytes)
	token := "srv_" + secret
	keyHint := "srv_..." + secret[len(secret)-4:]

	hashBytes := sha256.Sum256([]byte(secret))
	keyHash := hex.EncodeToString(hashBytes[:])

	var expiresAt time.Time
	if payload.ExpiresAt != nil {
		expiresAt = time.UnixMilli(*payload.ExpiresAt)
	}

	keyModel := repo.APIKey{
		ID:          repo.ULID(shared.GenerateULID()),
		UserID:      user.ID,
		Name:        payload.Name,
		KeyHash:     keyHash,
		KeyHint:     keyHint,
		ScopeView:   payload.ScopeView,
		ScopeCreate: payload.ScopeCreate,
		ScopeEdit:   payload.ScopeEdit,
		ScopeDelete: payload.ScopeDelete,
		ScopeAdmin:  payload.ScopeAdmin,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
	}

	createdKey, err := h.Repo.CreateAPIKey(ctx, keyModel)
	if err != nil {
		h.Logger.Error("Failed to create API key in repository", "error", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	response := APIKeyCreatedResponse{
		APIKeyResponse: mapToAPIKeyResponse(createdKey),
		Token:          token,
	}

	utils.RespondWithJSON(w, http.StatusCreated, response)
}

// GetAPIKeys godoc
// @Summary      Retrieve API keys for a user
// @Description  Retrieves the complete list of keys associated with the specified user. Hint only, actual key is hashed. Requires admin or self ownership.
// @Tags         User
// @Produce      json
// @Security     BearerAuth
// @Param        user_ulid path string true "User ULID"
// @Success      200  {array}   APIKeyResponse
// @Failure      401  {object}  utils.ErrorResponse "Unauthorized"
// @Failure      403  {object}  utils.ErrorResponse "Forbidden"
// @Failure      404  {object}  utils.ErrorResponse "User not found"
// @Router       /user/{user_ulid}/keys [get]
func (h *UserHandler) GetAPIKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userIDStr := r.PathValue("user_ulid")
	if userIDStr == "" {
		userIDStr = r.PathValue("user_id")
	}

	if _, err := h.Repo.GetUserByID(ctx, repo.ULID(userIDStr)); err != nil {
		if errors.Is(err, customerrors.ErrNotFound) {
			utils.RespondWithError(w, http.StatusNotFound, "User not found")
		} else {
			h.Logger.Error("Failed to retrieve user", "error", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	keys, err := h.Repo.GetAPIKeysByUserID(ctx, repo.ULID(userIDStr))
	if err != nil {
		h.Logger.Error("Failed to retrieve API keys for user", "error", err, "user_id", userIDStr)
		utils.RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	resp := make([]APIKeyResponse, len(keys))
	for i, key := range keys {
		resp[i] = mapToAPIKeyResponse(key)
	}

	utils.RespondWithJSON(w, http.StatusOK, resp)
}

// GetAPIKey godoc
// @Summary      Retrieve a specific API key details
// @Description  Retrieves the metadata of a specific API key. Requires admin or self ownership.
// @Tags         User
// @Produce      json
// @Security     BearerAuth
// @Param        user_ulid path string true "User ULID"
// @Param        key_ulid path string true "Key ULID"
// @Success      200  {object}  APIKeyResponse
// @Failure      401  {object}  utils.ErrorResponse "Unauthorized"
// @Failure      403  {object}  utils.ErrorResponse "Forbidden"
// @Failure      404  {object}  utils.ErrorResponse "Key not found"
// @Router       /user/{user_ulid}/keys/{key_ulid} [get]
func (h *UserHandler) GetAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userIDStr := r.PathValue("user_ulid")
	if userIDStr == "" {
		userIDStr = r.PathValue("user_id")
	}
	keyIDStr := r.PathValue("key_ulid")
	if keyIDStr == "" {
		keyIDStr = r.PathValue("key_id")
	}

	key, err := h.Repo.GetAPIKeyByID(ctx, repo.ULID(keyIDStr))
	if err != nil {
		if errors.Is(err, customerrors.ErrNotFound) {
			utils.RespondWithError(w, http.StatusNotFound, "API Key not found")
		} else {
			h.Logger.Error("Failed to retrieve API key", "error", err, "key_id", keyIDStr)
			utils.RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	if string(key.UserID) != userIDStr {
		utils.RespondWithError(w, http.StatusNotFound, "API Key not found for this user")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, mapToAPIKeyResponse(key))
}

// UpdateAPIKey godoc
// @Summary      Update an API key
// @Description  Updates an existing API key's name, expiry timestamp, or scopes. Requires admin or self ownership.
// @Tags         User
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        user_ulid path string true "User ULID"
// @Param        key_ulid path string true "Key ULID"
// @Param        payload body UpdateAPIKeyPayload true "Update payload"
// @Success      200  {object}  APIKeyResponse
// @Failure      400  {object}  utils.ErrorResponse "Bad Request"
// @Failure      401  {object}  utils.ErrorResponse "Unauthorized"
// @Failure      403  {object}  utils.ErrorResponse "Forbidden"
// @Failure      404  {object}  utils.ErrorResponse "Key not found"
// @Router       /user/{user_ulid}/keys/{key_ulid} [patch]
func (h *UserHandler) UpdateAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userIDStr := r.PathValue("user_ulid")
	if userIDStr == "" {
		userIDStr = r.PathValue("user_id")
	}
	keyIDStr := r.PathValue("key_ulid")
	if keyIDStr == "" {
		keyIDStr = r.PathValue("key_id")
	}

	key, err := h.Repo.GetAPIKeyByID(ctx, repo.ULID(keyIDStr))
	if err != nil {
		if errors.Is(err, customerrors.ErrNotFound) {
			utils.RespondWithError(w, http.StatusNotFound, "API Key not found")
		} else {
			h.Logger.Error("Failed to retrieve API key", "error", err, "key_id", keyIDStr)
			utils.RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	if string(key.UserID) != userIDStr {
		utils.RespondWithError(w, http.StatusNotFound, "API Key not found for this user")
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var payload UpdateAPIKeyPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if payload.Name != nil {
		key.Name = *payload.Name
	}
	if payload.ScopeView != nil {
		key.ScopeView = *payload.ScopeView
	}
	if payload.ScopeCreate != nil {
		key.ScopeCreate = *payload.ScopeCreate
	}
	if payload.ScopeEdit != nil {
		key.ScopeEdit = *payload.ScopeEdit
	}
	if payload.ScopeDelete != nil {
		key.ScopeDelete = *payload.ScopeDelete
	}
	if payload.ScopeAdmin != nil {
		key.ScopeAdmin = *payload.ScopeAdmin
	}

	// Detect if expires_at was explicitly sent (even if null)
	var rawMap map[string]any
	if err := json.Unmarshal(bodyBytes, &rawMap); err == nil {
		if _, hasExpiresAt := rawMap["expires_at"]; hasExpiresAt {
			if payload.ExpiresAt == nil {
				key.ExpiresAt = time.Time{}
			} else {
				key.ExpiresAt = time.UnixMilli(*payload.ExpiresAt)
			}
		}
	}

	updatedKey, err := h.Repo.UpdateAPIKey(ctx, key)
	if err != nil {
		h.Logger.Error("Failed to update API key", "error", err, "key_id", keyIDStr)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update API Key")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, mapToAPIKeyResponse(updatedKey))
}

// DeleteAPIKey godoc
// @Summary      Delete/revoke an API key
// @Description  Deletes/revokes an existing API key. Requires admin or self ownership.
// @Tags         User
// @Produce      json
// @Security     BearerAuth
// @Param        user_ulid path string true "User ULID"
// @Param        key_ulid path string true "Key ULID"
// @Success      200  {object}  utils.MessageResponse
// @Failure      401  {object}  utils.ErrorResponse "Unauthorized"
// @Failure      403  {object}  utils.ErrorResponse "Forbidden"
// @Failure      404  {object}  utils.ErrorResponse "Key not found"
// @Router       /user/{user_ulid}/keys/{key_ulid} [delete]
func (h *UserHandler) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userIDStr := r.PathValue("user_ulid")
	if userIDStr == "" {
		userIDStr = r.PathValue("user_id")
	}
	keyIDStr := r.PathValue("key_ulid")
	if keyIDStr == "" {
		keyIDStr = r.PathValue("key_id")
	}

	key, err := h.Repo.GetAPIKeyByID(ctx, repo.ULID(keyIDStr))
	if err != nil {
		if errors.Is(err, customerrors.ErrNotFound) {
			utils.RespondWithError(w, http.StatusNotFound, "API Key not found")
		} else {
			h.Logger.Error("Failed to retrieve API key", "error", err, "key_id", keyIDStr)
			utils.RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	if string(key.UserID) != userIDStr {
		utils.RespondWithError(w, http.StatusNotFound, "API Key not found for this user")
		return
	}

	err = h.Repo.DeleteAPIKey(ctx, repo.ULID(keyIDStr))
	if err != nil {
		h.Logger.Error("Failed to delete API key", "error", err, "key_id", keyIDStr)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete API Key")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, utils.MessageResponse{
		Message: fmt.Sprintf("API key '%s' (ID: %s) was successfully deleted.", key.Name, key.ID),
	})
}
