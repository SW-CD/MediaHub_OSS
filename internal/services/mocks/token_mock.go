// filepath: internal/services/mocks/token_mock.go
package mocks

import (
	"mediahub/internal/models"
	"mediahub/internal/services/auth"

	"github.com/stretchr/testify/mock"
)

// MockTokenService is a mock implementation of auth.TokenService
type MockTokenService struct {
	mock.Mock
}

var _ auth.TokenService = (*MockTokenService)(nil)

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
