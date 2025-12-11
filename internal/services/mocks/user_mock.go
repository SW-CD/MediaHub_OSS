// filepath: internal/services/mocks/user_mock.go
package mocks

import (
	"mediahub/internal/config"
	"mediahub/internal/models"
	"mediahub/internal/repository"
	"mediahub/internal/services"

	"github.com/stretchr/testify/mock"
)

// MockUserService is a mock implementation of services.UserService
type MockUserService struct {
	mock.Mock
}

// Compile-time check to ensure interface compliance
var _ services.UserService = (*MockUserService)(nil)

func (m *MockUserService) GetUserByUsername(username string) (*models.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) GetUserByID(id int) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) GetUsers() ([]models.User, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockUserService) UpdateUserPassword(username, password string) error {
	args := m.Called(username, password)
	return args.Error(0)
}

func (m *MockUserService) CreateUser(cArgs repository.UserCreateArgs) (*models.User, error) {
	args := m.Called(cArgs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) UpdateUser(id int, req models.User, newPassword *string) (*models.User, error) {
	args := m.Called(id, req, newPassword)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) DeleteUser(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserService) InitializeAdminUser(cfg *config.Config) error {
	args := m.Called(cfg)
	return args.Error(0)
}
