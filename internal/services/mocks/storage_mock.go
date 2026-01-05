// filepath: internal/services/mocks/storage_mock.go
package mocks

import (
	"github.com/stretchr/testify/mock"
)

// MockStorageService mocks the file storage operations
type MockStorageService struct {
	mock.Mock
}

func (m *MockStorageService) DeleteEntryFile(dbName string, timestamp, entryID int64) error {
	args := m.Called(dbName, timestamp, entryID)
	return args.Error(0)
}

func (m *MockStorageService) DeletePreviewFile(dbName string, timestamp, entryID int64) error {
	args := m.Called(dbName, timestamp, entryID)
	return args.Error(0)
}
