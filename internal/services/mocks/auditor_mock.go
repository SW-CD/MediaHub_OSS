// filepath: internal/services/mocks/auditor_mock.go
package mocks

import (
	"context"
	"mediahub/internal/services"

	"github.com/stretchr/testify/mock"
)

type MockAuditor struct {
	mock.Mock
}

var _ services.Auditor = (*MockAuditor)(nil)

func (m *MockAuditor) Log(ctx context.Context, action string, actor string, resource string, details map[string]interface{}) {
	m.Called(ctx, action, actor, resource, details)
}
