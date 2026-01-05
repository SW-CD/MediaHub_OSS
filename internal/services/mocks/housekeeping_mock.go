// filepath: internal/services/mocks/housekeeping_mock.go
package mocks

import (
	"mediahub/internal/models"
	"mediahub/internal/services"

	"github.com/stretchr/testify/mock"
)

// MockHousekeepingService is a mock implementation of services.HousekeepingService
type MockHousekeepingService struct {
	mock.Mock
}

var _ services.HousekeepingService = (*MockHousekeepingService)(nil)

func (m *MockHousekeepingService) Start() {
	m.Called()
}

func (m *MockHousekeepingService) Stop() {
	m.Called()
}

func (m *MockHousekeepingService) TriggerHousekeeping(dbName string) (*models.HousekeepingReport, error) {
	args := m.Called(dbName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.HousekeepingReport), args.Error(1)
}
