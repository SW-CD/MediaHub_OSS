package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"mediahub_oss/internal/httpserver/utils"
	"mediahub_oss/internal/repository"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// validateJWT parses the token string, validates the signature, and retrieves the user.
func (am *AuthMiddleware) validateJWT(tokenString string) (repository.User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return repository.User{}, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return am.JWTSecret, nil
	})

	if err != nil {
		return repository.User{}, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Check expiration
		if exp, ok := claims["exp"].(float64); ok {
			if time.Unix(int64(exp), 0).Before(time.Now()) {
				return repository.User{}, errors.New("token expired")
			}
		}

		// Extract User ID (ULID string)
		userIDStr, ok := claims["sub"].(string)
		if !ok {
			return repository.User{}, errors.New("invalid subject in token")
		}

		// Fetch fresh user data from DB to ensure they still exist / weren't banned
		return am.Repo.GetUserByID(context.Background(), repository.ULID(userIDStr))
	}

	return repository.User{}, errors.New("invalid token claims")
}

// validateBasicAuth decodes base64 credentials and verifies the password hash.
func (am *AuthMiddleware) validateBasicAuth(encodedValue string) (repository.User, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedValue)
	if err != nil {
		return repository.User{}, errors.New("invalid base64")
	}

	pair := strings.SplitN(string(decodedBytes), ":", 2)
	if len(pair) != 2 {
		return repository.User{}, errors.New("invalid basic auth format")
	}

	username, password := pair[0], pair[1]

	user, err := am.Repo.GetUserByUsername(context.Background(), username)
	if err != nil {
		return repository.User{}, errors.New("user not found")
	}

	// Verify Password using bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return repository.User{}, errors.New("invalid password")
	}

	return user, nil
}

// validateAPIKey hashes the secret part of the token, queries the API key joined with the owner, and checks for expiry.
func (am *AuthMiddleware) validateAPIKey(token string) (repository.User, repository.APIKey, error) {
	if len(token) <= 4 {
		return repository.User{}, repository.APIKey{}, errors.New("invalid token length")
	}

	secret := token[4:]
	hashBytes := sha256.Sum256([]byte(secret))
	keyHash := hex.EncodeToString(hashBytes[:])

	key, user, err := am.Repo.GetAPIKeyWithOwnerByHash(context.Background(), keyHash)
	if err != nil {
		return repository.User{}, repository.APIKey{}, err
	}

	// Validate expiry
	if !key.ExpiresAt.IsZero() && time.Now().After(key.ExpiresAt) {
		return repository.User{}, repository.APIKey{}, errors.New("token expired")
	}

	return user, key, nil
}

// hasPerm checks if the user has the requested permission for the given database ID using cached permissions in the context.
func (am *AuthMiddleware) hasPerm(ctx context.Context, dbID string, perm string) bool {
	permsMap := utils.GetPermissionsFromContext(ctx)
	if permsMap == nil {
		return false
	}

	p, ok := permsMap[repository.ULID(dbID)]
	if !ok {
		return false
	}

	if strings.Contains(p.Roles, perm) {
		return true
	}

	return false
}

// HasPermission validates if the authenticated user has access to a specific database action,
// taking into account API key restriction scopes (intersection logic).
func (am *AuthMiddleware) HasPermission(ctx context.Context, perm string, dbID string) bool {
	user := utils.GetUserFromContext(ctx)
	if user == nil {
		return false
	}

	// 1. API Key Scope checks (Intersection logic)
	apiKey := utils.GetAPIKeyFromContext(ctx)
	if apiKey != nil {
		var scopeAllowed bool
		switch perm {
		case "CanView":
			scopeAllowed = apiKey.ScopeView
		case "CanCreate":
			scopeAllowed = apiKey.ScopeCreate
		case "CanEdit":
			scopeAllowed = apiKey.ScopeEdit
		case "CanDelete":
			scopeAllowed = apiKey.ScopeDelete
		default:
			scopeAllowed = false
		}
		if !scopeAllowed {
			return false
		}
	}

	// 2. Global Admins bypass all database permission checks
	if utils.IsAdminFromContext(ctx) {
		return true
	}

	// 3. Check database-specific permissions from context
	return am.hasPerm(ctx, dbID, perm)
}
