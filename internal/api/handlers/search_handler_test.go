// filepath: internal/api/handlers/search_handler_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"mediahub/internal/models"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSearchEntriesAPI(t *testing.T) {
	// Use shared setup
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

	// Use specific matcher for the request pointer, generic for custom fields
	mockEntrySvc.On("SearchEntries",
		"SearchAPITestDB",
		mock.MatchedBy(func(req *models.SearchRequest) bool {
			return req.Filter.Field == "score" && req.Filter.Value == 0.9
		}),
		mock.Anything, // Custom fields slice
	).Return([]models.Entry{{"id": int64(2), "score": 0.95}}, nil).Once()

	searchBody, _ := json.Marshal(searchReq1)
	resp, err := http.Post(server.URL+"/database/entries/search?name=SearchAPITestDB", "application/json", bytes.NewReader(searchBody))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var results []models.Entry
	json.NewDecoder(resp.Body).Decode(&results)
	assert.Len(t, results, 1)
	// JSON numbers come back as float64 usually
	assert.Equal(t, 0.95, results[0]["score"])
}
