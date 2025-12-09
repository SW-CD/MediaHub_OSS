// filepath: internal/services/auth/token_service_test.go
package auth_test

import (
	"fmt"
	"mediahub/internal/config"
	"mediahub/internal/db/migrations" // Import
	"mediahub/internal/models"
	"mediahub/internal/repository"
	"mediahub/internal/services"
	"mediahub/internal/services/auth"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pressly/goose/v3" // Import
	"github.com/stretchr/testify/assert"
)

// setupServiceTest creates a temporary database, repository, user service, and token service.
func setupServiceTest(t *testing.T) (*repository.Repository, auth.TokenService, *models.User, func()) {
	t.Helper()
	const dbPath = "test_token_service.db"
	const storageRoot = "test_token_storage"

	os.Remove(dbPath)
	os.RemoveAll(storageRoot)
	os.MkdirAll(storageRoot, 0755)

	testCfg := &config.Config{
		Database: config.DatabaseConfig{
			Path:        dbPath,
			StorageRoot: storageRoot,
		},
		JWT: config.JWTConfig{
			AccessDurationMin:    5,
			RefreshDurationHours: 24,
			Secret:               "super-secret-key-for-testing",
		},
		JWTSecret: "super-secret-key-for-testing",
	}

	repo, err := repository.NewRepository(testCfg)
	if err != nil {
		t.Fatalf("Failed to create repo: %v", err)
	}

	// --- FIX: Apply Migrations ---
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("Failed to set dialect: %v", err)
	}
	if err := goose.Up(repo.DB, "."); err != nil {
		t.Fatalf("Failed to migrate test DB: %v", err)
	}

	userSvc := services.NewUserService(repo)
	tokenSvc := auth.NewTokenService(testCfg, userSvc, repo)

	userArgs := repository.UserCreateArgs{
		Username: "tokenuser",
		Password: "password123",
		CanView:  true,
	}
	user, err := userSvc.CreateUser(userArgs)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	cleanup := func() {
		repo.Close()
		os.Remove(dbPath)
		os.RemoveAll(storageRoot)
	}

	return repo, tokenSvc, user, cleanup
}

// ... Rest of the file remains exactly the same ...
// (TestGenerateTokens, TestValidateAccessToken, etc.)
// Re-include them to ensure file completeness if copying directly.

func TestGenerateTokens(t *testing.T) {
	repo, tokenSvc, user, cleanup := setupServiceTest(t)
	defer cleanup()

	accessToken, refreshToken, err := tokenSvc.GenerateTokens(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)

	parsedAccess, _ := jwt.Parse(accessToken, nil)
	accessClaims, ok := parsedAccess.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, "tokenuser", accessClaims["username"])
	assert.Equal(t, fmt.Sprintf("%d", user.ID), accessClaims["sub"])

	var count int
	err = repo.DB.QueryRow("SELECT COUNT(*) FROM refresh_tokens WHERE user_id = ?", user.ID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Refresh token hash should be stored in database")
}

func TestValidateAccessToken(t *testing.T) {
	_, tokenSvc, user, cleanup := setupServiceTest(t)
	defer cleanup()

	accessToken, _, err := tokenSvc.GenerateTokens(user)
	assert.NoError(t, err)

	validatedUser, err := tokenSvc.ValidateAccessToken(accessToken)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, validatedUser.ID)
	assert.Equal(t, user.Username, validatedUser.Username)

	tamperedToken := accessToken + "a"
	_, err = tokenSvc.ValidateAccessToken(tamperedToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestValidateAccessToken_Expired(t *testing.T) {
	_, tokenSvc, user, cleanup := setupServiceTest(t)
	defer cleanup()

	secret := []byte("super-secret-key-for-testing")
	claims := jwt.MapClaims{
		"username": user.Username,
		"sub":      fmt.Sprintf("%d", user.ID),
		"exp":      time.Now().Add(-1 * time.Minute).Unix(),
		"iss":      "mediahub",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredTokenString, _ := token.SignedString(secret)

	_, err := tokenSvc.ValidateAccessToken(expiredTokenString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestValidateRefreshToken_Stateful(t *testing.T) {
	repo, tokenSvc, user, cleanup := setupServiceTest(t)
	defer cleanup()

	_, refreshToken, err := tokenSvc.GenerateTokens(user)
	assert.NoError(t, err)

	validatedUser, err := tokenSvc.ValidateRefreshToken(refreshToken)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, validatedUser.ID)

	_, err = repo.DB.Exec("DELETE FROM refresh_tokens WHERE user_id = ?", user.ID)
	assert.NoError(t, err)

	_, err = tokenSvc.ValidateRefreshToken(refreshToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token not found in database")
}

func TestLogout(t *testing.T) {
	repo, tokenSvc, user, cleanup := setupServiceTest(t)
	defer cleanup()

	_, refreshToken, err := tokenSvc.GenerateTokens(user)
	assert.NoError(t, err)

	var count int
	repo.DB.QueryRow("SELECT COUNT(*) FROM refresh_tokens").Scan(&count)
	assert.Equal(t, 1, count)

	err = tokenSvc.Logout(refreshToken)
	assert.NoError(t, err)

	repo.DB.QueryRow("SELECT COUNT(*) FROM refresh_tokens").Scan(&count)
	assert.Equal(t, 0, count)

	_, err = tokenSvc.ValidateRefreshToken(refreshToken)
	assert.Error(t, err)
}

func TestUserDeletionCascadesTokens(t *testing.T) {
	repo, tokenSvc, user, cleanup := setupServiceTest(t)
	defer cleanup()

	_, _, err := tokenSvc.GenerateTokens(user)
	assert.NoError(t, err)

	var count int
	repo.DB.QueryRow("SELECT COUNT(*) FROM refresh_tokens").Scan(&count)
	assert.Equal(t, 1, count)

	err = repo.DeleteUser(int(user.ID))
	assert.NoError(t, err)

	repo.DB.QueryRow("SELECT COUNT(*) FROM refresh_tokens").Scan(&count)
	assert.Equal(t, 0, count, "Refresh token should be deleted when user is deleted")
}
