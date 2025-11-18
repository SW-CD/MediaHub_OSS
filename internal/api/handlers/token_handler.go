// filepath: internal/api/handlers/token_handler.go
package handlers

import (
	"encoding/json"
	"mediahub/internal/logging"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

// tokenRequest is the JSON body for refresh and logout endpoints.
type tokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// tokenResponse is the JSON body returned on successful token generation.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// @Summary Get JWT tokens
// @Description Authenticate using Basic Auth to receive an access and refresh token.
// @Tags Auth
// @Produce  json
// @Success 200 {object} tokenResponse
// @Failure 401 {object} ErrorResponse "Authentication failed"
// @Failure 500 {object} ErrorResponse "Token generation failed"
// @Security BasicAuth
// @Router /token [post]
func (h *Handlers) GetToken(w http.ResponseWriter, r *http.Request) {
	username, password, ok := r.BasicAuth()
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Authentication failed: Missing Basic Auth")
		return
	}

	// Validate user
	user, err := h.User.GetUserByUsername(username)
	if err != nil {
		// Avoid revealing if user exists
		respondWithError(w, http.StatusUnauthorized, "Authentication failed")
		return
	}

	// Compare password
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication failed")
		return
	}

	// Generate tokens (stateful)
	accessToken, refreshToken, err := h.Token.GenerateTokens(user)
	if err != nil {
		logging.Log.Errorf("Token generation failed for %s: %v", username, err)
		respondWithError(w, http.StatusInternalServerError, "Could not generate tokens")
		return
	}

	respondWithJSON(w, http.StatusOK, tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// @Summary Refresh JWT access token
// @Description Provide a valid refresh token to receive a new access token. The old refresh token is revoked.
// @Tags Auth
// @Accept   json
// @Produce  json
// @Param   token  body  tokenRequest  true  "Refresh Token"
// @Success 200 {object} tokenResponse
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Invalid or expired token"
// @Failure 500 {object} ErrorResponse "Token generation failed"
// @Router /token/refresh [post]
func (h *Handlers) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate the refresh token (checks signature and DB)
	user, err := h.Token.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	// Token Rotation: Invalidate the old refresh token immediately
	// This is a security best practice.
	if err := h.Token.Logout(req.RefreshToken); err != nil {
		logging.Log.Warnf("Failed to invalidate old refresh token during refresh for user %s: %v", user.Username, err)
		// We can continue, but it's a potential security risk
	}

	// Issue new token pair
	accessToken, refreshToken, err := h.Token.GenerateTokens(user)
	if err != nil {
		logging.Log.Errorf("Token refresh failed for %s: %v", user.Username, err)
		respondWithError(w, http.StatusInternalServerError, "Could not generate tokens")
		return
	}

	respondWithJSON(w, http.StatusOK, tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// @Summary Logout
// @Description Invalidates a refresh token. This endpoint is protected by an Access Token.
// @Tags Auth
// @Accept   json
// @Produce  json
// @Param   token  body  tokenRequest  true  "Refresh Token to invalidate"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Authentication required (invalid access token)"
// @Failure 500 {object} ErrorResponse "Could not process token"
// @Security BearerAuth
// @Router /logout [post]
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.Token.Logout(req.RefreshToken); err != nil {
		logging.Log.Errorf("Logout failed: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to logout")
		return
	}

	respondWithJSON(w, http.StatusOK, MessageResponse{Message: "Logged out successfully."})
}
