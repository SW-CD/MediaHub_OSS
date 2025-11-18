// filepath: internal/services/auth/auth_test.go
package auth_test

import (
	"encoding/json"
	"errors"
	"mediahub/internal/config"
	"mediahub/internal/models"
	"mediahub/internal/repository"
	"mediahub/internal/services"
	"mediahub/internal/services/auth"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// --- Mocks ---

type MockTokenService struct {
	mock.Mock
}

func (m *MockTokenService) GenerateTokens(user *models.User) (string, string, error) {
	args := m.Called(user)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockTokenService) ValidateAccessToken(tokenString string) (*models.User, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockTokenService) ValidateRefreshToken(tokenString string) (*models.User, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockTokenService) Logout(refreshToken string) error {
	args := m.Called(refreshToken)
	return args.Error(0)
}

// --- Test Setup ---

// setupMiddlewareTest creates the dependencies needed to test the middleware.
// It uses a real DB for UserService (to test Basic Auth fallback) and a Mock for TokenService.
func setupMiddlewareTest(t *testing.T) (*repository.Repository, *MockTokenService, *auth.Middleware, func()) {
	t.Helper()
	const dbPath = "test_middleware.db"
	const storageRoot = "test_middleware_storage"
	os.Remove(dbPath)
	os.RemoveAll(storageRoot)
	os.MkdirAll(storageRoot, 0755)

	cfg := &config.Config{
		Database: config.DatabaseConfig{Path: dbPath, StorageRoot: storageRoot},
	}
	repo, err := repository.NewRepository(cfg)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a test user for Basic Auth tests
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	_, err = repo.DB.Exec(`
		INSERT INTO users (username, password_hash, can_view, can_create, can_edit, can_delete)
		VALUES (?, ?, 1, 0, 0, 0)
	`, "basicuser", string(passwordHash))
	if err != nil {
		t.Fatalf("Failed to seed user: %v", err)
	}

	userSvc := services.NewUserService(repo)
	mockTokenSvc := new(MockTokenService)

	// Initialize Middleware with Hybrid services
	middleware := auth.NewMiddleware(userSvc, mockTokenSvc)

	cleanup := func() {
		repo.Close()
		os.Remove(dbPath)
		os.RemoveAll(storageRoot)
	}

	return repo, mockTokenSvc, middleware, cleanup
}

// --- Tests ---

func TestAuthMiddleware_Hybrid(t *testing.T) {
	_, mockTokenSvc, middleware, cleanup := setupMiddlewareTest(t)
	defer cleanup()

	// Setup Router with protected endpoint
	r := mux.NewRouter()
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value("user").(*models.User)
		w.Header().Set("X-User", user.Username)
		w.WriteHeader(http.StatusOK)
	})

	// Apply Middleware
	// We wrap the handler with AuthMiddleware -> RoleMiddleware("CanView")
	r.Handle("/protected", middleware.AuthMiddleware(middleware.RoleMiddleware("CanView")(protectedHandler)))

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("Case 1: Valid Bearer Token (JWT)", func(t *testing.T) {
		// Setup Mock
		jwtUser := &models.User{ID: 99, Username: "jwtuser", CanView: true}
		mockTokenSvc.On("ValidateAccessToken", "valid.jwt.token").Return(jwtUser, nil).Once()

		req, _ := http.NewRequest("GET", ts.URL+"/protected", nil)
		req.Header.Set("Authorization", "Bearer valid.jwt.token")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "jwtuser", resp.Header.Get("X-User"))
	})

	t.Run("Case 2: Invalid Bearer Token", func(t *testing.T) {
		// Setup Mock to fail
		mockTokenSvc.On("ValidateAccessToken", "invalid.jwt").Return(nil, errors.New("invalid token")).Once()

		req, _ := http.NewRequest("GET", ts.URL+"/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid.jwt")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Case 3: Expired Bearer Token", func(t *testing.T) {
		// Setup Mock to return specific expired error
		mockTokenSvc.On("ValidateAccessToken", "expired.jwt").Return(nil, errors.New("token is expired")).Once()

		req, _ := http.NewRequest("GET", ts.URL+"/protected", nil)
		req.Header.Set("Authorization", "Bearer expired.jwt")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		// Verify the specific error message for frontend handling
		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		assert.Equal(t, "Token expired", body["error"])
	})

	t.Run("Case 4: Valid Basic Auth (Fallback)", func(t *testing.T) {
		// No TokenService interaction expected
		req, _ := http.NewRequest("GET", ts.URL+"/protected", nil)
		req.SetBasicAuth("basicuser", "password123")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "basicuser", resp.Header.Get("X-User"))
	})

	t.Run("Case 5: Invalid Basic Auth", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/protected", nil)
		req.SetBasicAuth("basicuser", "wrongpassword")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Case 6: Role Authorization Failure", func(t *testing.T) {
		// Create a user WITHOUT "CanView" role
		noRoleUser := &models.User{ID: 100, Username: "norole", CanView: false}
		mockTokenSvc.On("ValidateAccessToken", "valid.norole.token").Return(noRoleUser, nil).Once()

		req, _ := http.NewRequest("GET", ts.URL+"/protected", nil)
		req.Header.Set("Authorization", "Bearer valid.norole.token")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		// Should be Forbidden (403), not Unauthorized (401)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	mockTokenSvc.AssertExpectations(t)
}
