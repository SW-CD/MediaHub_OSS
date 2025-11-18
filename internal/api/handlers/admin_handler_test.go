// filepath: internal/api/handlers/admin_handler_test.go
package handlers

import (
	"encoding/json"
	"errors"
	"mediahub/internal/models"
	"mediahub/internal/repository" // <-- Import services
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// setupAdminTest creates a handler with a mocked UserService
func setupAdminTest(t *testing.T) (*httptest.ResponseRecorder, *http.Request, *MockUserService, *MockInfoService) {
	mockUserSvc := new(MockUserService)
	mockInfoSvc := new(MockInfoService)
	rr := httptest.NewRecorder()
	return rr, httptest.NewRequest("GET", "/", nil), mockUserSvc, mockInfoSvc
}

func TestGetUsers(t *testing.T) {
	rr, req, mockUserSvc, mockInfoSvc := setupAdminTest(t)

	// Mock service response
	mockUsers := []models.User{
		{ID: 1, Username: "admin", PasswordHash: "hash1"},
		{ID: 2, Username: "viewer", PasswordHash: "hash2"},
	}
	mockUserSvc.On("GetUsers").Return(mockUsers, nil)

	// --- REFACTOR: Mock InfoService for NewHandlers ---
	mockInfoSvc.On("GetInfo").Return(models.Info{
		Version:     "test",
		UptimeSince: time.Now(), // <-- FIX: Was StartTime
	})
	// --- END REFACTOR ---

	// Create handler and serve
	// --- REFACTOR: Pass mockUserSvc as services.UserService interface ---
	h := NewHandlers(mockInfoSvc, mockUserSvc, nil, nil, nil, nil, nil) // <-- Added nil for TokenService
	// --- END REFACTOR ---
	h.GetUsers(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)
	mockUserSvc.AssertExpectations(t)

	var users []models.User
	err := json.Unmarshal(rr.Body.Bytes(), &users)
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "admin", users[0].Username)
	assert.Equal(t, "", users[0].PasswordHash, "Password hash was not sanitized")
	assert.Equal(t, "viewer", users[1].Username)
	assert.Equal(t, "", users[1].PasswordHash, "Password hash was not sanitized")
}

func TestCreateUser(t *testing.T) {
	rr, _, mockUserSvc, mockInfoSvc := setupAdminTest(t)

	// Mock service response
	createArgs := repository.UserCreateArgs{
		Username: "newuser",
		Password: "password",
		CanView:  true,
	}
	mockReturnUser := &models.User{
		ID:       3,
		Username: "newuser",
		CanView:  true,
	}
	mockUserSvc.On("CreateUser", createArgs).Return(mockReturnUser, nil)

	// --- REFACTOR: Mock InfoService for NewHandlers ---
	mockInfoSvc.On("GetInfo").Return(models.Info{})
	// --- END REFACTOR ---

	// Create request
	body := `{"username":"newuser", "password":"password", "can_view":true}`
	req, _ := http.NewRequest("POST", "/user", strings.NewReader(body))

	// Create handler and serve
	// --- REFACTOR: Pass mockUserSvc as services.UserService interface ---
	h := NewHandlers(mockInfoSvc, mockUserSvc, nil, nil, nil, nil, nil) // <-- Added nil for TokenService
	// --- END REFACTOR ---
	h.CreateUser(rr, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, rr.Code)
	mockUserSvc.AssertExpectations(t)

	var user models.User
	err := json.Unmarshal(rr.Body.Bytes(), &user)
	assert.NoError(t, err)
	assert.Equal(t, "newuser", user.Username)
	assert.Equal(t, int64(3), user.ID)
}

func TestCreateUser_Conflict(t *testing.T) {
	rr, _, mockUserSvc, mockInfoSvc := setupAdminTest(t)

	// Mock service response
	mockUserSvc.On("CreateUser", mock.Anything).Return(nil, repository.ErrUserExists)

	// --- REFACTOR: Mock InfoService for NewHandlers ---
	mockInfoSvc.On("GetInfo").Return(models.Info{})
	// --- END REFACTOR ---

	// Create request
	body := `{"username":"existinguser", "password":"password"}`
	req, _ := http.NewRequest("POST", "/user", strings.NewReader(body))

	// Create handler and serve
	// --- REFACTOR: Pass mockUserSvc as services.UserService interface ---
	h := NewHandlers(mockInfoSvc, mockUserSvc, nil, nil, nil, nil, nil) // <-- Added nil for TokenService
	// --- END REFACTOR ---
	h.CreateUser(rr, req)

	// Assertions
	assert.Equal(t, http.StatusConflict, rr.Code)
	mockUserSvc.AssertExpectations(t)

	var errResp ErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Equal(t, "Username already exists", errResp.Error)
}

func TestUpdateUser(t *testing.T) {
	rr, _, mockUserSvc, mockInfoSvc := setupAdminTest(t)
	userID := 1

	// Mock service response
	originalUser := &models.User{
		ID:       int64(userID),
		Username: "testuser",
		CanView:  true,
	}
	updatedUser := &models.User{
		ID:       int64(userID),
		Username: "testuser",
		CanView:  false, // Changed
		CanEdit:  true,  // Changed
	}
	newPassword := "newpass"

	// 1. Mock GetUserByID
	mockUserSvc.On("GetUserByID", userID).Return(originalUser, nil).Once()

	// 2. Mock UpdateUser
	// We need to match the arguments the handler will build
	expectedUpdateModel := *originalUser
	expectedUpdateModel.CanView = false
	expectedUpdateModel.CanEdit = true

	mockUserSvc.On("UpdateUser", userID, expectedUpdateModel, &newPassword).Return(updatedUser, nil).Once()

	// --- REFACTOR: Mock InfoService for NewHandlers ---
	mockInfoSvc.On("GetInfo").Return(models.Info{})
	// --- END REFACTOR ---

	// Create request
	body := `{"can_view":false, "can_edit":true, "password":"newpass"}`
	req, _ := http.NewRequest("PATCH", "/user?id=1", strings.NewReader(body))

	// Create handler and serve
	// --- REFACTOR: Pass mockUserSvc as services.UserService interface ---
	h := NewHandlers(mockInfoSvc, mockUserSvc, nil, nil, nil, nil, nil) // <-- Added nil for TokenService
	// --- END REFACTOR ---
	h.UpdateUser(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)
	mockUserSvc.AssertExpectations(t)

	var user models.User
	err := json.Unmarshal(rr.Body.Bytes(), &user)
	assert.NoError(t, err)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, false, user.CanView)
	assert.Equal(t, true, user.CanEdit)
}

func TestUpdateUser_LastAdmin(t *testing.T) {
	rr, _, mockUserSvc, mockInfoSvc := setupAdminTest(t)
	adminID := 1

	// Mock service response
	adminUser := &models.User{
		ID:       int64(adminID),
		Username: "admin",
		IsAdmin:  true,
	}

	// 1. Mock GetUserByID
	mockUserSvc.On("GetUserByID", adminID).Return(adminUser, nil).Once()

	// 2. Mock UpdateUser
	expectedUpdateModel := *adminUser
	expectedUpdateModel.IsAdmin = false // Attempting to remove admin

	mockUserSvc.On("UpdateUser", adminID, expectedUpdateModel, (*string)(nil)).Return(nil, errors.New("cannot remove the last admin's admin role")).Once()

	// --- REFACTOR: Mock InfoService for NewHandlers ---
	mockInfoSvc.On("GetInfo").Return(models.Info{})
	// --- END REFACTOR ---

	// Create request
	body := `{"is_admin":false}` // Attempt to remove admin
	req, _ := http.NewRequest("PATCH", "/user?id=1", strings.NewReader(body))

	// Create handler and serve
	// --- REFACTOR: Pass mockUserSvc as services.UserService interface ---
	h := NewHandlers(mockInfoSvc, mockUserSvc, nil, nil, nil, nil, nil) // <-- Added nil for TokenService
	// --- END REFACTOR ---
	h.UpdateUser(rr, req)

	// Assertions
	assert.Equal(t, http.StatusConflict, rr.Code)
	mockUserSvc.AssertExpectations(t)

	var errResp ErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Equal(t, "cannot remove the last admin's admin role", errResp.Error)
}

func TestDeleteUser(t *testing.T) {
	rr, _, mockUserSvc, mockInfoSvc := setupAdminTest(t)
	userID := 2

	// Mock service response
	mockUserSvc.On("DeleteUser", userID).Return(nil).Once()

	// --- REFACTOR: Mock InfoService for NewHandlers ---
	mockInfoSvc.On("GetInfo").Return(models.Info{})
	// --- END REFACTOR ---

	// Create request
	req, _ := http.NewRequest("DELETE", "/user?id=2", nil)

	// Create handler and serve
	// --- Pass mockUserSvc as services.UserService interface ---
	h := NewHandlers(mockInfoSvc, mockUserSvc, nil, nil, nil, nil, nil) // <-- Added nil for TokenService
	h.DeleteUser(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)
	mockUserSvc.AssertExpectations(t)

	var resp MessageResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "User deleted successfully.", resp.Message)
}

func TestDeleteUser_LastAdmin(t *testing.T) {
	rr, _, mockUserSvc, mockInfoSvc := setupAdminTest(t)
	adminID := 1

	// Mock service response
	mockUserSvc.On("DeleteUser", adminID).Return(errors.New("cannot delete the last admin user")).Once()

	// --- REFACTOR: Mock InfoService for NewHandlers ---
	mockInfoSvc.On("GetInfo").Return(models.Info{})
	// --- END REFACTOR ---

	// Create request
	req, _ := http.NewRequest("DELETE", "/user?id=1", nil)

	// Create handler and serve
	// --- REFACTOR: Pass mockUserSvc as services.UserService interface ---
	h := NewHandlers(mockInfoSvc, mockUserSvc, nil, nil, nil, nil, nil) // <-- Added nil for TokenService
	// --- END REFACTOR ---
	h.DeleteUser(rr, req)

	// Assertions
	assert.Equal(t, http.StatusConflict, rr.Code)
	mockUserSvc.AssertExpectations(t)

	var errResp ErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Equal(t, "cannot delete the last admin user", errResp.Error)
}
