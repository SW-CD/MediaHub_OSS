package tokenhandler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"mediahub_oss/internal/httpserver/utils"
	"mediahub_oss/internal/logging/audit"
	"mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared/customerrors"
)

type TokenHandler struct {
	Logger          *slog.Logger
	Auditor         audit.AuditLogger
	Repo            repository.Repository
	JWTSecret       []byte
	AccessDuration  time.Duration
	RefreshDuration time.Duration
}

// TokenResponse defines the JSON payload for successful token generation.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// TokenRequest defines the JSON payload for refreshing or logging out.
type TokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// @Summary Get a token pair
// @Description Obtains an internal JWT Access/Refresh token pair.
// @Description Supports two authentication methods:
// @Description 1. Local Authentication: Send standard Basic Auth headers.
// @Description 2. OIDC Token Exchange (commercial version only): Send a JSON body containing a valid external JWT (`idp_token`).
// @Description Providing both methods in a single request will result in a 400 Bad Request.
// @Tags token
// @Accept json
// @Produce json
// @Param body body OidcTokenRequest false "OIDC Identity Provider Token (required if not using Basic Auth)"
// @Success 200 {object} TokenResponse "Returns access and refresh tokens"
// @Failure 400 {object} utils.ErrorResponse "Ambiguous authentication request"
// @Failure 401 {object} utils.ErrorResponse "Invalid credentials, invalid OIDC token, or missing authentication"
// @Failure 500 {object} utils.ErrorResponse "Internal server error or OIDC not available"
// @Security BasicAuth
// @Router /api/token [post]
func (h *TokenHandler) GetToken(w http.ResponseWriter, r *http.Request) {

	username, password, hasBasicAuth := r.BasicAuth()
	oidcReq, hasOIDCAuth := checkOIDC(r)
	var user repository.User
	var err error

	// Requires either basic auth or OIDC auth
	if hasBasicAuth && hasOIDCAuth {
		h.Logger.Warn("Login attempt failed: ambiguous request (both Basic Auth and OIDC provided)")
		utils.RespondWithError(w, http.StatusBadRequest, "Ambiguous authentication request")
		return
	} else if !(hasBasicAuth || hasOIDCAuth) {
		utils.RespondWithError(w, http.StatusUnauthorized, "Missing authentication credentials")
		return
	}

	if hasBasicAuth {
		user, err = h.handleBasicAuth(r, username, password)
		if errors.Is(err, customerrors.ErrNotFound) {
			h.Logger.Warn("Login attempt failed: user not found", "username", username)
			utils.RespondWithError(w, http.StatusUnauthorized, "Invalid username or password")
			return
		} else if errors.Is(err, customerrors.ErrPermissionDenied) {
			h.Logger.Warn("Login attempt failed: invalid password", "username", username)
			utils.RespondWithError(w, http.StatusUnauthorized, "Invalid username or password")
			return
		} else if err != nil {
			h.Logger.Error("Failed to handle Basic Auth", "error", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to handle Basic Auth")
			return
		}
	}

	if hasOIDCAuth {
		user, err = h.handleOIDCValidationAndProvisioning(r.Context(), oidcReq.IdpToken)
		if errors.Is(err, customerrors.ErrNotImplemented) {
			h.Logger.Error("OIDC not available", "error", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "OIDC not available")
			return
		} else if err != nil {
			h.Logger.Error("Failed to handle OIDC", "error", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to handle OIDC")
			return
		}
	}

	// Generate and return tokens
	accessToken, refreshToken, err := h.generateTokens(r, user.ID)
	if err != nil {
		h.Logger.Error("Failed to generate tokens", "error", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	h.Auditor.Log(r.Context(), "auth.login", user.Username, "token", nil)

	utils.RespondWithJSON(w, http.StatusOK, TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// @Summary Refresh token pair
// @Description Uses a valid Refresh Token to obtain a new Access/Refresh token pair.
// @Tags token
// @Accept json
// @Produce json
// @Param body body TokenRequest true "Refresh token JSON body"
// @Success 200 {object} TokenResponse "Returns new access and refresh tokens"
// @Failure 400 {object} utils.ErrorResponse "Invalid request body"
// @Failure 401 {object} utils.ErrorResponse "Invalid or expired refresh token"
// @Router /api/token/refresh [post]
func (h *TokenHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	// Hash the provided refresh token to look it up in the database
	tokenHash := hashToken(req.RefreshToken)

	// Validate the token in the DB and get the associated UserID
	userID, err := h.Repo.ValidateRefreshToken(r.Context(), tokenHash)
	if errors.Is(err, customerrors.ErrNotFound) || errors.Is(err, customerrors.ErrPermissionDenied) {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	} else if err != nil {
		h.Logger.Error("Failed to verify refresh token", "error", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to verify refresh token")
		return
	}

	// Delete the old token (Token Rotation)
	_ = h.Repo.DeleteRefreshToken(r.Context(), tokenHash)

	// Generate a fresh pair of tokens
	accessToken, newRefreshToken, err := h.generateTokens(r, userID)
	if err != nil {
		h.Logger.Error("Failed to generate new tokens during refresh", "error", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	h.Auditor.Log(r.Context(), "auth.refresh", fmt.Sprintf("user_id:%d", userID), "token", nil)

	utils.RespondWithJSON(w, http.StatusOK, TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	})
}

// @Summary Logout
// @Description Revokes a Refresh Token, effectively logging the user out.
// @Tags token
// @Accept json
// @Produce json
// @Param body body TokenRequest true "Refresh token JSON body"
// @Success 200 {object} utils.MessageResponse "Logged out successfully message"
// @Failure 400 {object} utils.ErrorResponse "Invalid request body"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /api/logout [post]
func (h *TokenHandler) Logout(w http.ResponseWriter, r *http.Request) {

	user := utils.GetUserFromContext(r.Context())
	if user == nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "User not found")
		return
	}

	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	tokenHash := hashToken(req.RefreshToken)

	// Delete the token from the database
	err := h.Repo.DeleteRefreshToken(r.Context(), tokenHash)
	if err != nil {
		h.Logger.Warn("Logout attempted with invalid or already deleted token", "error", err)
		// We still return 200 OK to the client to prevent token enumeration
	}

	h.Auditor.Log(r.Context(), "auth.logout", user.Username, "token", nil)

	utils.RespondWithJSON(w, http.StatusOK, utils.MessageResponse{
		Message: "Logged out successfully.",
	})
}
