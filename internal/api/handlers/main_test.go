// filepath: internal/api/handlers/main_test.go
package handlers

import (
	"mediahub/internal/config"
	"mediahub/internal/models"
	"mediahub/internal/repository"
	"mediahub/internal/services"
	"mime/multipart"

	"github.com/stretchr/testify/mock"
)

// --- MOCK USER SERVICE ---
type MockUserService struct {
	mock.Mock
}

// --- REFACTOR: Compile-time check to ensure struct implements interface ---
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

// --- MOCK DATABASE SERVICE ---
type MockDatabaseService struct {
	mock.Mock
}

// --- REFACTOR: Compile-time check to ensure struct implements interface ---
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

// --- REMOVED: GetEntry, GetEntries, SearchEntries moved to MockEntryService ---

// --- MOCK ENTRY SERVICE ---
type MockEntryService struct {
	mock.Mock
}

// --- REFACTOR: Compile-time check to ensure struct implements interface ---
var _ services.EntryService = (*MockEntryService)(nil)

// ---
// FIX: Updated CreateEntry signature to match interface
// ---
func (m *MockEntryService) CreateEntry(dbName string, metadataStr string, file multipart.File, header *multipart.FileHeader) (interface{}, int, error) {
	args := m.Called(dbName, metadataStr, file, header)
	if args.Get(0) == nil {
		// Return (nil, status, error)
		return nil, args.Int(1), args.Error(2)
	}
	// Return (body, status, error)
	return args.Get(0), args.Int(1), args.Error(2)
}

func (m *MockEntryService) DeleteEntry(dbName string, id int64) error {
	args := m.Called(dbName, id)
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

// --- ADDED: Methods moved from MockDatabaseService ---

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

// --- MOCK HOUSEKEEPING SERVICE ---
type MockHousekeepingService struct {
	mock.Mock
}

// --- REFACTOR: Compile-time check to ensure struct implements interface ---
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

// --- MOCK INFO SERVICE ---
type MockInfoService struct {
	mock.Mock
}

// --- REFACTOR: Compile-time check to ensure struct implements interface ---
var _ services.InfoService = (*MockInfoService)(nil)

func (m *MockInfoService) GetInfo() models.Info {
	args := m.Called()
	return args.Get(0).(models.Info)
}
