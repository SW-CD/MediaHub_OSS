// filepath: internal/services/auth/auth_test.go
package auth_test

import (
	"mediahub/internal/config"
	"mediahub/internal/repository" // <-- IMPORT REPOSITORY
	"mediahub/internal/services"   // <-- IMPORT SERVICES
	"mediahub/internal/services/auth"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux" // <-- FIX: Corrected import path
	"golang.org/x/crypto/bcrypt"
)

// setupTestDB creates a temporary database for testing.
func setupTestDB(t *testing.T) (*repository.Repository, func()) { // <-- RETURN REPOSITORY
	const dbPath = "test_auth.db"
	const storageRoot = "test_auth_storage"
	os.Remove(dbPath)
	os.RemoveAll(storageRoot)
	os.MkdirAll(storageRoot, 0755)

	dummyCfg := &config.Config{
		Database: config.DatabaseConfig{
			Path:        dbPath,
			StorageRoot: storageRoot,
		},
	}
	// --- REFACTOR: Use Repository ---
	repo, err := repository.NewRepository(dummyCfg)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create a test user with a known password ("password")
	password := "password"
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Insert user
	_, err = repo.DB.Exec(`
		INSERT INTO users (username, password_hash, can_view, can_create, can_edit, can_delete)
		VALUES (?, ?, 1, 0, 0, 0)
	`, "testuser", string(passwordHash))
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	cleanup := func() {
		repo.Close()
		os.Remove(dbPath)
		os.RemoveAll(storageRoot)
	}

	return repo, cleanup
	// --- END REFACTOR ---
}

// TestAuthMiddleware tests the authentication and authorization middleware.
func TestAuthMiddleware(t *testing.T) {
	repo, cleanup := setupTestDB(t) // Get repo
	defer cleanup()

	// --- REFACTOR: Use UserService ---
	userService := services.NewUserService(repo)
	authMiddleware := auth.NewMiddleware(userService) // Pass service
	// --- END REFACTOR ---

	// Create a router and a test handler
	r := mux.NewRouter()
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Apply middleware
	r.Handle("/protected", authMiddleware.AuthMiddleware(authMiddleware.RoleMiddleware("CanView")(protectedHandler)))
	r.Handle("/forbidden", authMiddleware.AuthMiddleware(authMiddleware.RoleMiddleware("CanDelete")(protectedHandler)))

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test cases
	tests := []struct {
		name           string
		path           string
		username       string
		password       string
		expectedStatus int
	}{
		{"No Auth", "/protected", "", "", http.StatusUnauthorized},
		{"Bad Password", "/protected", "testuser", "wrongpassword", http.StatusUnauthorized},
		{"Correct Auth, Sufficient Role", "/protected", "testuser", "password", http.StatusOK},
		{"Correct Auth, Insufficient Role", "/forbidden", "testuser", "password", http.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", ts.URL+tc.path, nil)
			if tc.username != "" && tc.password != "" {
				req.SetBasicAuth(tc.username, tc.password)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
			}
		})
	}
}
