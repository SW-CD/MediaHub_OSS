// filepath: internal/services/mocks/database_mock.go
package mocks

import (
	"mediahub/internal/models"
	"mediahub/internal/services"

	"github.com/stretchr/testify/mock"
)

// MockDatabaseService is a mock implementation of services.DatabaseService
type MockDatabaseService struct {
	mock.Mock
}

var _ services.DatabaseService = (*MockDatabaseService)(nil)

func (m *MockDatabaseService) GetDatabase(name string) (*models.Database, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Database), args.Error(1)
}

func (m *MockDatabaseService) GetDatabases() ([]models.Database, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Database), args.Error(1)
}

func (m *MockDatabaseService) CreateDatabase(payload models.DatabaseCreatePayload) (*models.Database, error) {
	args := m.Called(payload)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Database), args.Error(1)
}

func (m *MockDatabaseService) UpdateDatabase(name string, updates models.DatabaseUpdatePayload) (*models.Database, error) {
	args := m.Called(name, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Database), args.Error(1)
}

func (m *MockDatabaseService) DeleteDatabase(name string) error {
	args := m.Called(name)
	return args.Error(0)
}
