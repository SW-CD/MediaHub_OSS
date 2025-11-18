// filepath: internal/services/auth/token_service_test.go
package auth_test

import (
	"fmt"
	"mediahub/internal/config"
	"mediahub/internal/models"
	"mediahub/internal/repository"
	"mediahub/internal/services"
	"mediahub/internal/services/auth"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

// setupServiceTest creates a temporary database, repository, user service, and token service.
// It also creates a default test user.
func setupServiceTest(t *testing.T) (*repository.Repository, auth.TokenService, *models.User, func()) {
	t.Helper()
	const dbPath = "test_token_service.db"
	const storageRoot = "test_token_storage"

	// Clean up previous runs
	os.Remove(dbPath)
	os.RemoveAll(storageRoot)
	os.MkdirAll(storageRoot, 0755)

	// 1. Setup Config
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
		JWTSecret: "super-secret-key-for-testing", // Runtime secret
	}

	// 2. Setup Repository
	repo, err := repository.NewRepository(testCfg)
	if err != nil {
		t.Fatalf("Failed to create repo: %v", err)
	}

	// 3. Setup Services
	userSvc := services.NewUserService(repo)
	tokenSvc := auth.NewTokenService(testCfg, userSvc, repo)

	// 4. Create a Test User
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

func TestGenerateTokens(t *testing.T) {
	repo, tokenSvc, user, cleanup := setupServiceTest(t)
	defer cleanup()

	// Call GenerateTokens
	accessToken, refreshToken, err := tokenSvc.GenerateTokens(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)

	// Verify Access Token contents
	// We can't verify the signature easily without exporting the secret or using the validator,
	// but we can parse the claims (unverified) to check structure.
	parsedAccess, _ := jwt.Parse(accessToken, nil)
	accessClaims, ok := parsedAccess.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, "tokenuser", accessClaims["username"])
	assert.Equal(t, fmt.Sprintf("%d", user.ID), accessClaims["sub"])

	// Verify Refresh Token persistence in Database
	// We query the refresh_tokens table directly to ensure the hash is stored.
	var count int
	err = repo.DB.QueryRow("SELECT COUNT(*) FROM refresh_tokens WHERE user_id = ?", user.ID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Refresh token hash should be stored in database")
}

func TestValidateAccessToken(t *testing.T) {
	_, tokenSvc, user, cleanup := setupServiceTest(t)
	defer cleanup()

	// 1. Generate valid tokens
	accessToken, _, err := tokenSvc.GenerateTokens(user)
	assert.NoError(t, err)

	// 2. Validate successfully
	validatedUser, err := tokenSvc.ValidateAccessToken(accessToken)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, validatedUser.ID)
	assert.Equal(t, user.Username, validatedUser.Username)

	// 3. Test Invalid Token (Tampered)
	// Append a character to invalidate signature
	tamperedToken := accessToken + "a"
	_, err = tokenSvc.ValidateAccessToken(tamperedToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid") // Likely "signature is invalid"
}

func TestValidateAccessToken_Expired(t *testing.T) {
	_, tokenSvc, user, cleanup := setupServiceTest(t)
	defer cleanup()

	// Manually create an expired token using the same secret
	secret := []byte("super-secret-key-for-testing")
	claims := jwt.MapClaims{
		"username": user.Username,
		"sub":      fmt.Sprintf("%d", user.ID),
		"exp":      time.Now().Add(-1 * time.Minute).Unix(), // Expired 1 min ago
		"iss":      "mediahub",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredTokenString, _ := token.SignedString(secret)

	// Attempt validate
	_, err := tokenSvc.ValidateAccessToken(expiredTokenString)
	assert.Error(t, err)
	// JWT library usually returns "token is expired"
	assert.Contains(t, err.Error(), "expired")
}

func TestValidateRefreshToken_Stateful(t *testing.T) {
	repo, tokenSvc, user, cleanup := setupServiceTest(t)
	defer cleanup()

	// 1. Generate tokens
	_, refreshToken, err := tokenSvc.GenerateTokens(user)
	assert.NoError(t, err)

	// 2. Validate successfully
	validatedUser, err := tokenSvc.ValidateRefreshToken(refreshToken)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, validatedUser.ID)

	// 3. Simulate manual revocation (DB deletion)
	// Delete the token from the DB directly
	_, err = repo.DB.Exec("DELETE FROM refresh_tokens WHERE user_id = ?", user.ID)
	assert.NoError(t, err)

	// 4. Validate again -> Should fail (Stateful check)
	_, err = tokenSvc.ValidateRefreshToken(refreshToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token not found in database")
}

func TestLogout(t *testing.T) {
	repo, tokenSvc, user, cleanup := setupServiceTest(t)
	defer cleanup()

	// 1. Generate tokens
	_, refreshToken, err := tokenSvc.GenerateTokens(user)
	assert.NoError(t, err)

	// Verify it exists in DB
	var count int
	repo.DB.QueryRow("SELECT COUNT(*) FROM refresh_tokens").Scan(&count)
	assert.Equal(t, 1, count)

	// 2. Call Logout
	err = tokenSvc.Logout(refreshToken)
	assert.NoError(t, err)

	// 3. Verify it is gone from DB
	repo.DB.QueryRow("SELECT COUNT(*) FROM refresh_tokens").Scan(&count)
	assert.Equal(t, 0, count)

	// 4. Validate should fail
	_, err = tokenSvc.ValidateRefreshToken(refreshToken)
	assert.Error(t, err)
}

func TestUserDeletionCascadesTokens(t *testing.T) {
	repo, tokenSvc, user, cleanup := setupServiceTest(t)
	defer cleanup()

	// 1. Generate tokens
	_, _, err := tokenSvc.GenerateTokens(user)
	assert.NoError(t, err)

	// Verify token exists
	var count int
	repo.DB.QueryRow("SELECT COUNT(*) FROM refresh_tokens").Scan(&count)
	assert.Equal(t, 1, count)

	// 2. Delete User (Using Repository directly or Service)
	// This tests the ON DELETE CASCADE SQL definition
	err = repo.DeleteUser(int(user.ID))
	assert.NoError(t, err)

	// 3. Verify token is gone automatically
	repo.DB.QueryRow("SELECT COUNT(*) FROM refresh_tokens").Scan(&count)
	assert.Equal(t, 0, count, "Refresh token should be deleted when user is deleted")
}
