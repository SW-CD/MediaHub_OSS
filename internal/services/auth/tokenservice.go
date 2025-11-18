// filepath: internal/services/auth/token_service.go
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"mediahub/internal/config"
	"mediahub/internal/models"
	"mediahub/internal/repository"
	"mediahub/internal/services"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// accessClaims defines the custom claims for our short-lived access token.
type accessClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// refreshClaims defines the claims for our long-lived, stateful refresh token.
type refreshClaims struct {
	jwt.RegisteredClaims
}

// Compile-time check to ensure tokenService implements the TokenService interface.
var _ TokenService = (*tokenService)(nil)

// tokenService implements the TokenService interface.
type tokenService struct {
	cfg     *config.Config
	userSvc services.UserService
	repo    *repository.Repository
}

// NewTokenService creates a new instance of the tokenService.
func NewTokenService(cfg *config.Config, userSvc services.UserService, repo *repository.Repository) TokenService {
	return &tokenService{cfg: cfg, userSvc: userSvc, repo: repo}
}

// hashToken securely hashes a token string (using SHA-256) for database storage.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// GenerateTokens creates, signs, and *stores* a new token pair.
func (s *tokenService) GenerateTokens(user *models.User) (string, string, error) {
	// 1. Create Access Token (short-lived, stateless)
	accessExpiry := time.Now().Add(time.Minute * time.Duration(s.cfg.JWT.AccessDurationMin))
	accessClaims := &accessClaims{
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			Issuer:    "mediahub",
			Subject:   fmt.Sprintf("%d", user.ID), // Store user ID in 'sub' claim
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	signedAccessToken, err := accessToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	// 2. Create Refresh Token (long-lived, stateful)
	refreshExpiry := time.Now().Add(time.Hour * time.Duration(s.cfg.JWT.RefreshDurationHours))
	// Use a random ID (jti) for the token to ensure uniqueness
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate token id: %w", err)
	}
	jti := hex.EncodeToString(jtiBytes)

	refreshClaims := &refreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			Issuer:    "mediahub",
			Subject:   fmt.Sprintf("%d", user.ID),
			ID:        jti, // Unique identifier for this token
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	signedRefreshToken, err := refreshToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	// 3. Store the hash of the refresh token in the database
	tokenHash := hashToken(signedRefreshToken)
	if err := s.repo.StoreRefreshToken(user.ID, tokenHash, refreshExpiry); err != nil {
		return "", "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return signedAccessToken, signedRefreshToken, nil
}

// ValidateAccessToken checks an access token (stateless).
// It verifies the signature and expiry, then returns the associated user.
func (s *tokenService) ValidateAccessToken(tokenString string) (*models.User, error) {
	claims := &accessClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})

	if err != nil {
		return nil, err // Handles expired tokens as well
	}
	if !token.Valid {
		return nil, errors.New("invalid access token")
	}

	// Token is valid, fetch the user from the claims
	user, err := s.userSvc.GetUserByUsername(claims.Username)
	if err != nil {
		return nil, errors.New("user not found for token")
	}
	return user, nil
}

// ValidateRefreshToken checks a refresh token (stateful).
// It verifies the signature AND checks the database to ensure it hasn't been revoked.
func (s *tokenService) ValidateRefreshToken(tokenString string) (*models.User, error) {
	// 1. Check signature and basic claims (stateless check)
	claims := &refreshClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err // Handles expired tokens
	}
	if !token.Valid {
		return nil, errors.New("invalid refresh token signature or claims")
	}

	// 2. Hash the token and check the database "Allow List" (stateful check)
	tokenHash := hashToken(tokenString)
	userID, err := s.repo.ValidateRefreshToken(tokenHash)
	if err != nil {
		// This means the token is not in the DB (it was logged out, expired, or never existed)
		return nil, fmt.Errorf("token not found in database (revoked or expired): %w", err)
	}

	// 3. Fetch the user associated with the valid token
	user, err := s.userSvc.GetUserByID(int(userID))
	if err != nil {
		return nil, errors.New("user not found for valid token")
	}
	return user, nil
}

// Logout invalidates a refresh token by deleting its hash from the database.
func (s *tokenService) Logout(refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	return s.repo.DeleteRefreshToken(tokenHash)
}
