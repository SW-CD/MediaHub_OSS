// filepath: internal/housekeeping/service_test.go
package housekeeping

import (
	"mediahub/internal/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDB is a mock implementation of the DBTX interface for testing.
type MockDB struct {
	mock.Mock
}

func (m *MockDB) GetDatabase(name string) (*models.Database, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Database), args.Error(1)
}

func (m *MockDB) GetDatabaseStats(dbName string) (*models.Stats, error) {
	args := m.Called(dbName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Stats), args.Error(1)
}

func (m *MockDB) GetEntriesOlderThan(dbName string, timestamp int64, customFields []models.CustomField) ([]models.Entry, error) {
	args := m.Called(dbName, timestamp, customFields)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Entry), args.Error(1)
}

func (m *MockDB) GetOldestEntries(dbName string, limit, offset int, customFields []models.CustomField) ([]models.Entry, error) {
	args := m.Called(dbName, limit, offset, customFields)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Entry), args.Error(1)
}

func (m *MockDB) DeleteEntry(dbName string, id int64) error {
	args := m.Called(dbName, id)
	return args.Error(0)
}

func (m *MockDB) UpdateDatabaseLastHkRun(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockDB) GetDatabases() ([]models.Database, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Database), args.Error(1)
}

type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) DeleteEntryFile(dbName string, timestamp, entryID int64) error {
	args := m.Called(dbName, timestamp, entryID)
	return args.Error(0)
}

func (m *MockStorage) DeletePreviewFile(dbName string, timestamp, entryID int64) error {
	args := m.Called(dbName, timestamp, entryID)
	return args.Error(0)
}

// setupTest creates a new service with mock dependencies for testing.
func setupTest() (*Service, *MockDB, *MockStorage) {
	mockDB := new(MockDB)
	mockStorage := new(MockStorage)
	deps := Dependencies{
		DB:      mockDB,
		Storage: mockStorage,
	}
	service := NewService(deps)
	return service, mockDB, mockStorage
}

func TestScheduleNextRun(t *testing.T) {
	t.Run("No databases", func(t *testing.T) {
		service, mockDB, _ := setupTest()
		mockDB.On("GetDatabases").Return([]models.Database{}, nil).Once()
		duration := service.scheduleNextRun()
		assert.Equal(t, DefaultCheckInterval, duration)
		mockDB.AssertExpectations(t)
	})

	t.Run("One database, next run in future", func(t *testing.T) {
		service, mockDB, _ := setupTest()
		dbs := []models.Database{
			{Name: "DB1", Housekeeping: models.Housekeeping{Interval: "1h"}, LastHkRun: time.Now().Add(-30 * time.Minute)},
		}
		mockDB.On("GetDatabases").Return(dbs, nil).Once()
		duration := service.scheduleNextRun()
		assert.True(t, duration > 29*time.Minute && duration < 31*time.Minute)
		mockDB.AssertExpectations(t)
	})

	t.Run("One database, next run in past", func(t *testing.T) {
		service, mockDB, _ := setupTest()
		dbs := []models.Database{
			{Name: "DB1", Housekeeping: models.Housekeeping{Interval: "1h"}, LastHkRun: time.Now().Add(-90 * time.Minute)},
		}
		mockDB.On("GetDatabases").Return(dbs, nil).Once()
		duration := service.scheduleNextRun()
		assert.Equal(t, MinCheckInterval, duration)
		mockDB.AssertExpectations(t)
	})

	t.Run("Multiple databases, soonest is chosen", func(t *testing.T) {
		service, mockDB, _ := setupTest()
		dbs := []models.Database{
			{Name: "DB1", Housekeeping: models.Housekeeping{Interval: "1h"}, LastHkRun: time.Now().Add(-30 * time.Minute)},
			{Name: "DB2", Housekeeping: models.Housekeeping{Interval: "30m"}, LastHkRun: time.Now().Add(-15 * time.Minute)},
		}
		mockDB.On("GetDatabases").Return(dbs, nil).Once()
		duration := service.scheduleNextRun()
		assert.True(t, duration > 14*time.Minute && duration < 16*time.Minute)
		mockDB.AssertExpectations(t)
	})

	t.Run("Database with invalid interval", func(t *testing.T) {
		service, mockDB, _ := setupTest()
		dbs := []models.Database{
			{Name: "DB1", Housekeeping: models.Housekeeping{Interval: "invalid"}},
		}
		mockDB.On("GetDatabases").Return(dbs, nil).Once()
		duration := service.scheduleNextRun()
		assert.Equal(t, DefaultCheckInterval, duration)
		mockDB.AssertExpectations(t)
	})

	t.Run("Database with zero interval", func(t *testing.T) {
		service, mockDB, _ := setupTest()
		dbs := []models.Database{
			{Name: "DB1", Housekeeping: models.Housekeeping{Interval: "0"}},
		}
		mockDB.On("GetDatabases").Return(dbs, nil).Once()
		duration := service.scheduleNextRun()
		assert.Equal(t, DefaultCheckInterval, duration)
		mockDB.AssertExpectations(t)
	})
}

func TestRunForDatabase_CleanupByAge(t *testing.T) {
	_, mockDB, mockStorage := setupTest()
	deps := Dependencies{DB: mockDB, Storage: mockStorage}

	db := &models.Database{
		Name: "TestDB_Age",
		Housekeeping: models.Housekeeping{
			MaxAge:    "30d",
			DiskSpace: "1G", // Set high so it doesn't trigger
		},
	}

	oldEntry := models.Entry{
		"id":        int64(1),
		"timestamp": time.Now().Add(-31 * 24 * time.Hour).Unix(),
		"filesize":  int64(1024),
	}

	mockDB.On("GetDatabase", "TestDB_Age").Return(db, nil)
	mockDB.On("GetEntriesOlderThan", "TestDB_Age", mock.AnythingOfType("int64"), mock.Anything).Return([]models.Entry{oldEntry}, nil)
	mockDB.On("GetDatabaseStats", "TestDB_Age").Return(&models.Stats{TotalDiskSpaceBytes: 2048}, nil)
	mockStorage.On("DeleteEntryFile", "TestDB_Age", oldEntry["timestamp"].(int64), oldEntry["id"].(int64)).Return(nil)
	mockStorage.On("DeletePreviewFile", "TestDB_Age", oldEntry["timestamp"].(int64), oldEntry["id"].(int64)).Return(nil)
	mockDB.On("DeleteEntry", "TestDB_Age", int64(1)).Return(nil)

	report, err := RunForDatabase(deps, "TestDB_Age")

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, 1, report.EntriesDeleted)
	assert.Equal(t, int64(1024), report.SpaceFreedBytes)
	mockDB.AssertCalled(t, "DeleteEntry", "TestDB_Age", int64(1))
}

func TestRunForDatabase_CleanupByDiskSpace(t *testing.T) {
	_, mockDB, mockStorage := setupTest()
	deps := Dependencies{DB: mockDB, Storage: mockStorage}

	db := &models.Database{
		Name: "TestDB_Space",
		Housekeeping: models.Housekeeping{
			MaxAge:    "365d",
			DiskSpace: "1K", // 1024 bytes
		},
	}

	oldestEntry := models.Entry{
		"id":        int64(10),
		"timestamp": time.Now().Add(-10 * 24 * time.Hour).Unix(),
		"filesize":  int64(512),
	}
	secondOldestEntry := models.Entry{
		"id":        int64(11),
		"timestamp": time.Now().Add(-9 * 24 * time.Hour).Unix(),
		"filesize":  int64(600),
	}

	mockDB.On("GetDatabase", "TestDB_Space").Return(db, nil)
	mockDB.On("GetEntriesOlderThan", "TestDB_Space", mock.AnythingOfType("int64"), mock.Anything).Return([]models.Entry{}, nil)
	mockDB.On("GetDatabaseStats", "TestDB_Space").Return(&models.Stats{TotalDiskSpaceBytes: 1112}, nil).Once()
	mockDB.On("GetOldestEntries", "TestDB_Space", 100, 0, mock.Anything).Return([]models.Entry{oldestEntry, secondOldestEntry}, nil).Once()
	mockStorage.On("DeleteEntryFile", "TestDB_Space", oldestEntry["timestamp"].(int64), oldestEntry["id"].(int64)).Return(nil)
	mockStorage.On("DeletePreviewFile", "TestDB_Space", oldestEntry["timestamp"].(int64), oldestEntry["id"].(int64)).Return(nil)
	mockDB.On("DeleteEntry", "TestDB_Space", oldestEntry["id"].(int64)).Return(nil)

	report, err := RunForDatabase(deps, "TestDB_Space")

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, 1, report.EntriesDeleted)
	assert.Equal(t, int64(512), report.SpaceFreedBytes)
	mockDB.AssertCalled(t, "DeleteEntry", "TestDB_Space", int64(10))
	mockDB.AssertNotCalled(t, "DeleteEntry", "TestDB_Space", int64(11))
}
