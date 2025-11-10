// filepath: internal/api/handlers/user_handler_test.go
package handlers

import (
	"context"
	"encoding/json"
	"mediahub/internal/models" // <-- Import services
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- REFACTOR: MockUserService is now defined in main_test.go ---

func TestGetUserMe(t *testing.T) {
	// --- REFACTOR: Mock InfoService for NewHandlers ---
	mockInfoSvc := new(MockInfoService)
	mockInfoSvc.On("GetInfo").Return(models.Info{
		Version:     "test",
		UptimeSince: time.Now(), // <-- FIX: Was StartTime
	})
	// This test doesn't require the full API server, just a handler.
	h := NewHandlers(mockInfoSvc, nil, nil, nil, nil, nil) // No service needed for this handler
	// --- END REFACTOR ---

	// Create a mock user
	mockUser := &models.User{
		ID:       1,
		Username: "testuser",
		CanView:  true,
	}

	// Create a request with the mock user in the context
	req, err := http.NewRequest("GET", "/api/me", nil)
	assert.NoError(t, err)
	ctx := context.WithValue(req.Context(), "user", mockUser)
	req = req.WithContext(ctx)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	h.GetUserMe(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the response body
	var returnedUser models.User
	err = json.Unmarshal(rr.Body.Bytes(), &returnedUser)
	assert.NoError(t, err)

	assert.Equal(t, "testuser", returnedUser.Username)
	assert.True(t, returnedUser.CanView)
	assert.False(t, returnedUser.CanCreate)        // Ensure other fields are default
	assert.Equal(t, "", returnedUser.PasswordHash) // Ensure sensitive fields are cleared
}

func TestGetUserMe_NoUserInContext(t *testing.T) {
	// --- REFACTOR: Mock InfoService for NewHandlers ---
	mockInfoSvc := new(MockInfoService)
	mockInfoSvc.On("GetInfo").Return(models.Info{})
	h := NewHandlers(mockInfoSvc, nil, nil, nil, nil, nil) // No service needed
	// --- END REFACTOR ---

	req, err := http.NewRequest("GET", "/api/me", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	h.GetUserMe(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var responseBody map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &responseBody)
	assert.NoError(t, err)
	assert.Equal(t, "No user found in context", responseBody["error"])
}

// --- NEW: Test for UpdateUserMe ---
func TestUpdateUserMe(t *testing.T) {
	mockUserSvc := new(MockUserService)
	// --- REFACTOR: Mock InfoService for NewHandlers ---
	mockInfoSvc := new(MockInfoService)
	mockInfoSvc.On("GetInfo").Return(models.Info{})
	h := NewHandlers(mockInfoSvc, mockUserSvc, nil, nil, nil, nil)
	// --- END REFACTOR ---

	mockUser := &models.User{
		ID:       1,
		Username: "testuser",
	}

	// Mock the service call
	mockUserSvc.On("UpdateUserPassword", "testuser", "newpass").Return(nil)

	// Create request
	body := `{"password":"newpass"}`
	req, err := http.NewRequest("PATCH", "/api/me", strings.NewReader(body))
	assert.NoError(t, err)

	// Add user to context
	ctx := context.WithValue(req.Context(), "user", mockUser)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.UpdateUserMe(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)
	mockUserSvc.AssertExpectations(t) // Verify the service was called

	var resp MessageResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Password updated successfully.", resp.Message)
}
