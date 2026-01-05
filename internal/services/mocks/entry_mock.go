// filepath: internal/services/mocks/entry_mock.go
package mocks

import (
	"context"
	"io"
	"mediahub/internal/models"
	"mediahub/internal/services"
	"mime/multipart"

	"github.com/stretchr/testify/mock"
)

// MockEntryService is a mock implementation of services.EntryService
type MockEntryService struct {
	mock.Mock
}

var _ services.EntryService = (*MockEntryService)(nil)

func (m *MockEntryService) CreateEntry(dbName string, metadataStr string, file multipart.File, header *multipart.FileHeader) (interface{}, int, error) {
	args := m.Called(dbName, metadataStr, file, header)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0), args.Int(1), args.Error(2)
}

func (m *MockEntryService) DeleteEntry(dbName string, id int64) error {
	args := m.Called(dbName, id)
	return args.Error(0)
}

func (m *MockEntryService) DeleteEntries(dbName string, ids []int64) (int, int64, error) {
	args := m.Called(dbName, ids)
	return args.Int(0), args.Get(1).(int64), args.Error(2)
}

func (m *MockEntryService) ExportEntries(ctx context.Context, dbName string, ids []int64, w io.Writer) error {
	args := m.Called(ctx, dbName, ids, w)
	return args.Error(0)
}

func (m *MockEntryService) UpdateEntry(dbName string, id int64, updates models.Entry) (models.Entry, error) {
	args := m.Called(dbName, id, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(models.Entry), args.Error(1)
}

func (m *MockEntryService) GetEntryFile(dbName string, id int64) (string, string, string, error) {
	args := m.Called(dbName, id)
	return args.String(0), args.String(1), args.String(2), args.Error(3)
}

func (m *MockEntryService) GetEntryPreview(dbName string, id int64) (string, error) {
	args := m.Called(dbName, id)
	return args.String(0), args.Error(1)
}

func (m *MockEntryService) GetEntry(dbName string, id int64, customFields []models.CustomField) (models.Entry, error) {
	args := m.Called(dbName, id, customFields)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(models.Entry), args.Error(1)
}

func (m *MockEntryService) GetEntries(dbName string, limit, offset int, order string, tstart, tend int64, customFields []models.CustomField) ([]models.Entry, error) {
	args := m.Called(dbName, limit, offset, order, tstart, tend, customFields)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Entry), args.Error(1)
}

func (m *MockEntryService) SearchEntries(dbName string, req *models.SearchRequest, customFields []models.CustomField) ([]models.Entry, error) {
	args := m.Called(dbName, req, customFields)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Entry), args.Error(1)
}
