// filepath: internal/api/handlers/token_handler_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"mediahub/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// setupTokenHandlerTestAPI creates a new test server for token handlers.
func setupTokenHandlerTestAPI(t *testing.T) (*httptest.Server, *MockTokenService, *MockUserService, func()) {
	t.Helper()

	mockTokenSvc := new(MockTokenService)
	mockUserSvc := new(MockUserService)
	mockInfoSvc := new(MockInfoService)

	// Mock InfoService for NewHandlers (required dependency)
	mockInfoSvc.On("GetInfo").Return(models.Info{
		Version:     "test",
		UptimeSince: time.Now(),
	})

	// Use the AuthMiddleware with the mocked services
	// Note: For unit testing specific handlers, we often don't need the full middleware stack
	// unless we are testing the routing protection itself.
	// Here we pass the mockTokenSvc to the handlers struct.
	h := NewHandlers(
		mockInfoSvc,
		mockUserSvc,
		mockTokenSvc, // TokenService is 3rd argument
		nil,          // database
		nil,          // entry
		nil,          // housekeeping
		nil,          // cfg
	)

	r := mux.NewRouter()

	// Public Routes
	r.HandleFunc("/api/token", h.GetToken).Methods("POST")
	r.HandleFunc("/api/token/refresh", h.RefreshToken).Methods("POST")

	// Protected Routes (Mocking auth middleware context for simplicity in unit tests)
	// In a real integration test, we would mount the actual middleware.
	// Here we will manually set context in the test if needed, or rely on the handler logic.
	r.HandleFunc("/logout", h.Logout).Methods("POST")

	server := httptest.NewServer(r)

	cleanup := func() {
		server.Close()
	}

	return server, mockTokenSvc, mockUserSvc, cleanup
}

// TestGetToken_InvalidUser tests the scenario where the username is not found.
// Note: We skip success testing here because mocking bcrypt inside the handler requires
// real hash generation setup, which is better suited for integration tests.
func TestGetToken_InvalidUser(t *testing.T) {
	server, _, mockUser, cleanup := setupTokenHandlerTestAPI(t)
	defer cleanup()

	mockUser.On("GetUserByUsername", "unknown").Return(nil, errors.New("user not found"))

	req, _ := http.NewRequest("POST", server.URL+"/api/token", nil)
	req.SetBasicAuth("unknown", "pass")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestRefreshToken_Success(t *testing.T) {
	server, mockToken, _, cleanup := setupTokenHandlerTestAPI(t)
	defer cleanup()

	validRefreshToken := "valid.refresh.token"
	mockUserObj := &models.User{ID: 1, Username: "user1"}

	// 1. Mock ValidateRefreshToken
	mockToken.On("ValidateRefreshToken", validRefreshToken).Return(mockUserObj, nil).Once()

	// 2. Mock Logout (Token Rotation)
	mockToken.On("Logout", validRefreshToken).Return(nil).Once()

	// 3. Mock GenerateTokens
	mockToken.On("GenerateTokens", mockUserObj).Return("new_access", "new_refresh", nil).Once()

	// Request Body
	reqBody := map[string]string{"refresh_token": validRefreshToken}
	jsonBody, _ := json.Marshal(reqBody)

	resp, err := http.Post(server.URL+"/api/token/refresh", "application/json", bytes.NewReader(jsonBody))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var tokenResp tokenResponse
	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	assert.NoError(t, err)
	assert.Equal(t, "new_access", tokenResp.AccessToken)
	assert.Equal(t, "new_refresh", tokenResp.RefreshToken)

	mockToken.AssertExpectations(t)
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	server, mockToken, _, cleanup := setupTokenHandlerTestAPI(t)
	defer cleanup()

	invalidToken := "invalid.token"

	// Mock ValidateRefreshToken to fail
	mockToken.On("ValidateRefreshToken", invalidToken).Return(nil, errors.New("invalid token")).Once()

	reqBody := map[string]string{"refresh_token": invalidToken}
	jsonBody, _ := json.Marshal(reqBody)

	resp, err := http.Post(server.URL+"/api/token/refresh", "application/json", bytes.NewReader(jsonBody))
	assert.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	mockToken.AssertExpectations(t)
}

func TestLogout_Success(t *testing.T) {
	server, mockToken, _, cleanup := setupTokenHandlerTestAPI(t)
	defer cleanup()

	refreshToken := "token.to.revoke"

	// Mock Logout
	mockToken.On("Logout", refreshToken).Return(nil).Once()

	reqBody := map[string]string{"refresh_token": refreshToken}
	jsonBody, _ := json.Marshal(reqBody)

	resp, err := http.Post(server.URL+"/logout", "application/json", bytes.NewReader(jsonBody))
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockToken.AssertExpectations(t)
}

func TestLogout_InternalError(t *testing.T) {
	server, mockToken, _, cleanup := setupTokenHandlerTestAPI(t)
	defer cleanup()

	refreshToken := "token.db.error"

	// Mock Logout to fail
	mockToken.On("Logout", refreshToken).Return(errors.New("db error")).Once()

	reqBody := map[string]string{"refresh_token": refreshToken}
	jsonBody, _ := json.Marshal(reqBody)

	resp, err := http.Post(server.URL+"/logout", "application/json", bytes.NewReader(jsonBody))
	assert.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	mockToken.AssertExpectations(t)
}
