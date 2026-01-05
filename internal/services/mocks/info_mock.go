// filepath: internal/services/mocks/info_mock.go
package mocks

import (
	"mediahub/internal/models"
	"mediahub/internal/services"

	"github.com/stretchr/testify/mock"
)

// MockInfoService is a mock implementation of services.InfoService
type MockInfoService struct {
	mock.Mock
}

var _ services.InfoService = (*MockInfoService)(nil)

func (m *MockInfoService) GetInfo() models.Info {
	args := m.Called()
	return args.Get(0).(models.Info)
}
