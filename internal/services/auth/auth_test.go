// filepath: internal/services/auth/auth_test.go
package auth_test

import (
	"encoding/json"
	"errors"
	"mediahub/internal/config"
	"mediahub/internal/db/migrations" // Import
	"mediahub/internal/models"
	"mediahub/internal/repository"
	"mediahub/internal/services"
	"mediahub/internal/services/auth"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	"github.com/pressly/goose/v3" // Import
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// ... Mocks (Same as before) ...
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

	// --- FIX: Apply Migrations ---
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("Failed to set dialect: %v", err)
	}
	if err := goose.Up(repo.DB, "."); err != nil {
		t.Fatalf("Failed to migrate middleware DB: %v", err)
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

	middleware := auth.NewMiddleware(userSvc, mockTokenSvc)

	cleanup := func() {
		repo.Close()
		os.Remove(dbPath)
		os.RemoveAll(storageRoot)
	}

	return repo, mockTokenSvc, middleware, cleanup
}

// ... Tests (Same as before) ...
func TestAuthMiddleware_Hybrid(t *testing.T) {
	_, mockTokenSvc, middleware, cleanup := setupMiddlewareTest(t)
	defer cleanup()

	r := mux.NewRouter()
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value("user").(*models.User)
		w.Header().Set("X-User", user.Username)
		w.WriteHeader(http.StatusOK)
	})

	r.Handle("/protected", middleware.AuthMiddleware(middleware.RoleMiddleware("CanView")(protectedHandler)))

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("Case 1: Valid Bearer Token (JWT)", func(t *testing.T) {
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
		mockTokenSvc.On("ValidateAccessToken", "invalid.jwt").Return(nil, errors.New("invalid token")).Once()

		req, _ := http.NewRequest("GET", ts.URL+"/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid.jwt")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Case 3: Expired Bearer Token", func(t *testing.T) {
		mockTokenSvc.On("ValidateAccessToken", "expired.jwt").Return(nil, errors.New("token is expired")).Once()

		req, _ := http.NewRequest("GET", ts.URL+"/protected", nil)
		req.Header.Set("Authorization", "Bearer expired.jwt")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		assert.Equal(t, "Token expired", body["error"])
	})

	t.Run("Case 4: Valid Basic Auth (Fallback)", func(t *testing.T) {
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
		noRoleUser := &models.User{ID: 100, Username: "norole", CanView: false}
		mockTokenSvc.On("ValidateAccessToken", "valid.norole.token").Return(noRoleUser, nil).Once()

		req, _ := http.NewRequest("GET", ts.URL+"/protected", nil)
		req.Header.Set("Authorization", "Bearer valid.norole.token")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	mockTokenSvc.AssertExpectations(t)
}
