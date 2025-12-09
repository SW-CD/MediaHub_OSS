// filepath: internal/api/handlers/user_handler_test.go
package handlers

import (
	"context"
	"encoding/json"
	"mediahub/internal/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetUserMe(t *testing.T) {
	mockInfoSvc := new(MockInfoService)
	mockInfoSvc.On("GetInfo").Return(models.Info{
		Version:     "test",
		UptimeSince: time.Now(),
	})
	mockAuditor := new(MockAuditor)

	// Updated constructor with MockAuditor
	h := NewHandlers(
		mockInfoSvc, // info
		nil,         // user (not needed for GET)
		nil,         // token
		nil,         // database
		nil,         // entry
		nil,         // housekeeping
		mockAuditor, // auditor
		nil,         // cfg
	)

	mockUser := &models.User{
		ID:       1,
		Username: "testuser",
		CanView:  true,
	}

	req, err := http.NewRequest("GET", "/api/me", nil)
	assert.NoError(t, err)

	ctx := context.WithValue(req.Context(), "user", mockUser)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.GetUserMe(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)

	var returnedUser models.User
	err = json.Unmarshal(rr.Body.Bytes(), &returnedUser)
	assert.NoError(t, err)

	assert.Equal(t, "testuser", returnedUser.Username)
	assert.True(t, returnedUser.CanView)
	assert.Equal(t, "", returnedUser.PasswordHash) // Ensure sensitive fields are cleared
}

func TestGetUserMe_NoUserInContext(t *testing.T) {
	mockInfoSvc := new(MockInfoService)
	mockInfoSvc.On("GetInfo").Return(models.Info{})
	mockAuditor := new(MockAuditor)

	h := NewHandlers(mockInfoSvc, nil, nil, nil, nil, nil, mockAuditor, nil)

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

func TestUpdateUserMe(t *testing.T) {
	mockUserSvc := new(MockUserService)
	mockInfoSvc := new(MockInfoService)
	mockInfoSvc.On("GetInfo").Return(models.Info{})
	mockAuditor := new(MockAuditor)

	h := NewHandlers(mockInfoSvc, mockUserSvc, nil, nil, nil, nil, mockAuditor, nil)

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
	mockUserSvc.AssertExpectations(t)

	var resp MessageResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Password updated successfully.", resp.Message)
}
