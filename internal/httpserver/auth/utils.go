package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
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

		// Extract User ID
		// Note: JWT stores numbers as float64 by default
		userIDFloat, ok := claims["sub"].(float64)
		if !ok {
			return repository.User{}, errors.New("invalid subject in token")
		}

		// Fetch fresh user data from DB to ensure they still exist / weren't banned
		return am.Repo.GetUserByID(context.Background(), int64(userIDFloat))
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

// hasPerm checks if the user has the requested permission for the given database ID.
func (am *AuthMiddleware) hasPerm(user *repository.User, dbID string, perm string) bool {
	if user == nil {
		return false
	}

	perms, err := am.Repo.GetUserPermissions(context.Background(), user.ID, dbID)
	if err != nil {
		return false
	}

	if strings.Contains(perms.Roles, perm) {
		return true
	}

	return false
}
