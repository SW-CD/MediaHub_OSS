// filepath: internal/services/auth/interfaces.go
package auth

import "mediahub/internal/models"

// TokenService defines the contract for JWT operations.
type TokenService interface {
	GenerateTokens(user *models.User) (accessToken string, refreshToken string, err error)
	ValidateAccessToken(tokenString string) (*models.User, error)
	ValidateRefreshToken(tokenString string) (*models.User, error)
	Logout(refreshToken string) error
}
