// filepath: internal/api/handlers/main_test.go
package handlers

import (
	"mediahub/internal/config"
	"mediahub/internal/models"
	"mediahub/internal/services/mocks"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/mock"
)

func setupDBHandlerTestAPI(t *testing.T) (*httptest.Server, *mocks.MockDatabaseService, *mocks.MockInfoService, *mocks.MockAuditor, func()) {
	t.Helper()

	mockDBService := new(mocks.MockDatabaseService)
	mockInfoService := new(mocks.MockInfoService)
	mockAuditor := new(mocks.MockAuditor)
	mockEntryService := new(mocks.MockEntryService)

	dummyCfg := &config.Config{}

	mockInfoService.On("GetInfo").Return(models.Info{
		Version:     "test",
		UptimeSince: time.Now(),
	})

	h := NewHandlers(
		mockInfoService,
		nil,
		nil,
		mockDBService,
		mockEntryService,
		nil,
		mockAuditor,
		dummyCfg,
	)

	r := mux.NewRouter()
	r.HandleFunc("/database", h.CreateDatabase).Methods("POST")
	r.HandleFunc("/database", h.UpdateDatabase).Methods("PUT")
	r.HandleFunc("/databases", h.GetDatabases).Methods("GET")
	r.HandleFunc("/database", h.GetDatabase).Methods("GET")
	r.HandleFunc("/database/entries/delete", h.DeleteEntries).Methods("POST")

	server := httptest.NewServer(r)
	cleanup := func() {
		server.Close()
	}

	return server, mockDBService, mockInfoService, mockAuditor, cleanup
}

func setupDBHandlerTestAPI_Full(t *testing.T) (*httptest.Server, *mocks.MockDatabaseService, *mocks.MockEntryService, *mocks.MockInfoService, *mocks.MockAuditor, func()) {
	t.Helper()

	mockDBService := new(mocks.MockDatabaseService)
	mockEntryService := new(mocks.MockEntryService)
	mockInfoService := new(mocks.MockInfoService)
	mockAuditor := new(mocks.MockAuditor)
	dummyCfg := &config.Config{}

	mockInfoService.On("GetInfo").Return(models.Info{
		Version:     "test",
		UptimeSince: time.Now(),
	})

	h := NewHandlers(
		mockInfoService,
		nil,
		nil,
		mockDBService,
		mockEntryService,
		nil,
		mockAuditor,
		dummyCfg,
	)

	r := mux.NewRouter()
	r.HandleFunc("/database", h.CreateDatabase).Methods("POST")
	r.HandleFunc("/database", h.UpdateDatabase).Methods("PUT")
	r.HandleFunc("/databases", h.GetDatabases).Methods("GET")
	r.HandleFunc("/database", h.GetDatabase).Methods("GET")
	r.HandleFunc("/database/entries/delete", h.DeleteEntries).Methods("POST")

	server := httptest.NewServer(r)
	cleanup := func() {
		server.Close()
	}

	return server, mockDBService, mockEntryService, mockInfoService, mockAuditor, cleanup
}

func setupEntryHandlerTestAPI(t *testing.T) (*httptest.Server, *mocks.MockEntryService, *mocks.MockDatabaseService, *mocks.MockAuditor, func()) {
	t.Helper()

	mockEntrySvc := new(mocks.MockEntryService)
	mockDbSvc := new(mocks.MockDatabaseService)
	mockAuditor := new(mocks.MockAuditor)

	infoSvc := new(mocks.MockInfoService)
	infoSvc.On("GetInfo").Return(models.Info{
		Version:     "test",
		UptimeSince: time.Now(),
	})

	dummyCfg := &config.Config{
		MaxSyncUploadSizeBytes: 8 << 20,
	}

	h := NewHandlers(
		infoSvc,
		nil,
		nil,
		mockDbSvc,
		mockEntrySvc,
		nil,
		mockAuditor,
		dummyCfg,
	)

	r := mux.NewRouter()
	r.HandleFunc("/entry", h.UploadEntry).Methods("POST")
	r.HandleFunc("/entry/meta", h.GetEntryMeta).Methods("GET")
	r.HandleFunc("/entry/file", h.GetEntry).Methods("GET")
	r.HandleFunc("/entry/preview", h.GetEntryPreview).Methods("GET")
	r.HandleFunc("/database/entries/export", h.ExportEntries).Methods("POST")

	server := httptest.NewServer(r)

	cleanup := func() {
		server.Close()
	}

	return server, mockEntrySvc, mockDbSvc, mockAuditor, cleanup
}

func setupAdminTest(t *testing.T) (*httptest.ResponseRecorder, *http.Request, *mocks.MockUserService, *mocks.MockInfoService, *mocks.MockAuditor) {
	mockUserSvc := new(mocks.MockUserService)
	mockInfoSvc := new(mocks.MockInfoService)
	mockAuditor := new(mocks.MockAuditor)
	rr := httptest.NewRecorder()
	return rr, httptest.NewRequest("GET", "/", nil), mockUserSvc, mockInfoSvc, mockAuditor
}

func setupTokenHandlerTestAPI(t *testing.T) (*httptest.Server, *mocks.MockTokenService, *mocks.MockUserService, func()) {
	t.Helper()

	mockTokenSvc := new(mocks.MockTokenService)
	mockUserSvc := new(mocks.MockUserService)
	mockInfoSvc := new(mocks.MockInfoService)
	mockAuditor := new(mocks.MockAuditor)

	mockInfoSvc.On("GetInfo").Return(models.Info{
		Version:     "test",
		UptimeSince: time.Now(),
	})

	h := NewHandlers(
		mockInfoSvc,
		mockUserSvc,
		mockTokenSvc,
		nil,
		nil,
		nil,
		mockAuditor,
		nil,
	)

	r := mux.NewRouter()
	r.HandleFunc("/api/token", h.GetToken).Methods("POST")
	r.HandleFunc("/api/token/refresh", h.RefreshToken).Methods("POST")
	r.HandleFunc("/logout", h.Logout).Methods("POST")

	server := httptest.NewServer(r)
	cleanup := func() {
		server.Close()
	}

	return server, mockTokenSvc, mockUserSvc, cleanup
}

func setupSearchTestAPI(t *testing.T) (*httptest.Server, *mocks.MockDatabaseService, *mocks.MockEntryService, func()) {
	t.Helper()

	mockDBService := new(mocks.MockDatabaseService)
	mockEntryService := new(mocks.MockEntryService)
	mockAuditor := new(mocks.MockAuditor)
	dummyCfg := &config.Config{}

	infoSvc := new(mocks.MockInfoService)
	infoSvc.On("GetInfo").Return(models.Info{
		Version:     "test",
		UptimeSince: time.Now(),
	})
	h := NewHandlers(
		infoSvc,
		nil,
		nil,
		mockDBService,
		mockEntryService,
		nil,
		mockAuditor,
		dummyCfg,
	)

	r := mux.NewRouter()
	r.HandleFunc("/database/entries/search", h.SearchEntries).Methods("POST")

	server := httptest.NewServer(r)
	cleanup := func() {
		server.Close()
	}

	return server, mockDBService, mockEntryService, cleanup
}

func AuditLogMatcher(action string, dbName string) interface{} {
	return mock.MatchedBy(func(args []interface{}) bool {
		if len(args) < 4 {
			return false
		}
		if args[1] != action {
			return false
		}
		if args[3] != dbName {
			return false
		}
		return true
	})
}
