// filepath: internal/api/handlers/search_handler_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"mediahub/internal/config"
	"mediahub/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupSearchTestAPI(t *testing.T) (*httptest.Server, *MockDatabaseService, *MockEntryService, func()) {
	t.Helper()

	mockDBService := new(MockDatabaseService)
	mockEntryService := new(MockEntryService)
	mockAuditor := new(MockAuditor)
	dummyCfg := &config.Config{}

	infoSvc := new(MockInfoService)
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
		mockAuditor, // <-- Inject
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

func TestSearchEntriesAPI(t *testing.T) {
	server, mockDB, mockEntrySvc, cleanup := setupSearchTestAPI(t)
	defer cleanup()

	dbPayload := &models.Database{
		Name:        "SearchAPITestDB",
		ContentType: "image",
		CustomFields: []models.CustomField{
			{Name: "score", Type: "REAL"},
		},
	}
	mockDB.On("GetDatabase", "SearchAPITestDB").Return(dbPayload, nil)

	searchReq1 := models.SearchRequest{
		Filter: &models.SearchFilter{
			Field:    "score",
			Operator: ">",
			Value:    0.9,
		},
		Pagination: &models.SearchPagination{Limit: func(i int) *int { return &i }(10)},
	}

	// Fix Mock call: use specific matcher for CustomFields slice
	mockEntrySvc.On("SearchEntries",
		"SearchAPITestDB",
		&searchReq1,
		mock.Anything, // Custom fields slice
	).Return([]models.Entry{{"id": 2, "score": 0.95}}, nil).Once()

	searchBody, _ := json.Marshal(searchReq1)
	resp, err := http.Post(server.URL+"/database/entries/search?name=SearchAPITestDB", "application/json", bytes.NewReader(searchBody))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
