// filepath: internal/api/handlers/entry_handler_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mediahub/internal/models"
	"mediahub/internal/services"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- REFACTOR: MockEntryService is now defined in main_test.go ---

// setupEntryHandlerTestAPI creates a new test server for entry handlers.
func setupEntryHandlerTestAPI(t *testing.T) (*httptest.Server, *MockEntryService, *MockDatabaseService, func()) {
	t.Helper()

	mockEntrySvc := new(MockEntryService)
	mockDbSvc := new(MockDatabaseService)

	// --- REFACTOR: Mock InfoService ---
	infoSvc := new(MockInfoService) // Use mock
	infoSvc.On("GetInfo").Return(models.Info{
		Version:     "test",
		UptimeSince: time.Now(),
	})

	h := NewHandlers(
		infoSvc,      // info
		nil,          // user
		mockDbSvc,    // database (for GetEntryMeta)
		mockEntrySvc, // entry
		nil,          // housekeeping
		nil,          // cfg
	)
	// --- END REFACTOR ---

	r := mux.NewRouter()
	r.HandleFunc("/entry", h.UploadEntry).Methods("POST")
	r.HandleFunc("/entry/meta", h.GetEntryMeta).Methods("GET")
	r.HandleFunc("/entry/file", h.GetEntry).Methods("GET") // <-- ADDED FOR FILENAME TEST

	server := httptest.NewServer(r)

	cleanup := func() {
		server.Close()
	}

	return server, mockEntrySvc, mockDbSvc, cleanup
}

func TestUploadEntry_Success(t *testing.T) {
	server, mockEntrySvc, _, cleanup := setupEntryHandlerTestAPI(t)
	defer cleanup()

	// --- Prepare multipart form ---
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Metadata part
	metadata := map[string]interface{}{
		"description": "Test entry from API",
		"timestamp":   12345,
	}
	metadataBytes, _ := json.Marshal(metadata)
	metadataStr := string(metadataBytes)
	part, _ := writer.CreateFormField("metadata")
	part.Write(metadataBytes)

	// Entry part
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="dummy.png"`)
	h.Set("Content-Type", "image/png")
	entryPart, _ := writer.CreatePart(h)
	entryPart.Write([]byte("dummy png data")) // Just write dummy data
	writer.Close()

	// --- Mock the service call ---
	returnedEntry := models.Entry{
		"id":          float64(1), // JSON unmarshals numbers to float64
		"description": "Test entry from API",
		"mime_type":   "image/png",
		"filename":    "dummy.png",
		"width":       float64(300),
		"height":      float64(200),
		"status":      "ready", // <-- ADDED
	}
	// ---
	// FIX: Update mock to return (body, status, error)
	// ---
	mockEntrySvc.On(
		"CreateEntry",
		"EntryAPITestDB",
		metadataStr,
		mock.Anything, // mock.AnythingOfType("multipart.File")
		mock.Anything, // mock.AnythingOfType("*multipart.FileHeader")
	).Return(returnedEntry, http.StatusCreated, nil).Once() // <-- RETURN (body, 201, nil)
	// --- END FIX ---

	// --- Upload Entry ---
	req, _ := http.NewRequest("POST", server.URL+"/entry?database_name=EntryAPITestDB", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)

	// --- Assertions ---
	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, string(bodyBytes))
	}
	defer resp.Body.Close()

	var createdEntry models.Entry
	err = json.NewDecoder(resp.Body).Decode(&createdEntry)
	assert.NoError(t, err)
	assert.Equal(t, "Test entry from API", createdEntry["description"])
	assert.Equal(t, float64(300), createdEntry["width"])
	assert.Equal(t, "dummy.png", createdEntry["filename"])

	// Verify the mock was called
	mockEntrySvc.AssertExpectations(t)
}

func TestUploadEntry_DBNotFound(t *testing.T) {
	server, mockEntrySvc, _, cleanup := setupEntryHandlerTestAPI(t)
	defer cleanup()

	// --- Prepare multipart form (minimal) ---
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("metadata", `{"timestamp": 12345}`)
	part, _ := writer.CreateFormFile("file", "dummy.png")
	part.Write([]byte("dummy data"))
	writer.Close()

	// ---
	// FIX: Update mock to return (body, status, error)
	// ---
	mockEntrySvc.On(
		"CreateEntry",
		"NotFoundDB",
		mock.AnythingOfType("string"),
		mock.Anything, // mock.AnythingOfType("multipart.File")
		mock.Anything, // mock.AnythingOfType("*multipart.FileHeader")
	).Return(nil, http.StatusNotFound, services.ErrNotFound).Once() // <-- RETURN (nil, 404, err)
	// --- END FIX ---

	// --- Upload Entry ---
	req, _ := http.NewRequest("POST", server.URL+"/entry?database_name=NotFoundDB", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)

	// --- Assertions ---
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var errResp ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	assert.NoError(t, err)
	assert.Equal(t, "Database not found.", errResp.Error)

	mockEntrySvc.AssertExpectations(t)
}

func TestUploadEntry_UnsupportedMedia(t *testing.T) {
	server, mockEntrySvc, _, cleanup := setupEntryHandlerTestAPI(t)
	defer cleanup()

	// --- Prepare multipart form ---
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("metadata", `{"timestamp": 12345}`)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="dummy.txt"`)
	h.Set("Content-Type", "text/plain") // This is the important part
	part, _ := writer.CreatePart(h)
	part.Write([]byte("dummy data"))
	writer.Close()

	// ---
	// FIX: Update mock to return (body, status, error)
	// ---
	mockEntrySvc.On(
		"CreateEntry",
		"ImageDB",
		mock.AnythingOfType("string"),
		mock.Anything, // mock.AnythingOfType("multipart.File")
		mock.Anything, // mock.AnythingOfType("*multipart.FileHeader")
	).Return(nil, http.StatusUnsupportedMediaType, services.ErrUnsupported).Once() // <-- RETURN (nil, 415, err)
	// --- END FIX ---

	// --- Upload Entry ---
	req, _ := http.NewRequest("POST", server.URL+"/entry?database_name=ImageDB", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)

	// --- Assertions ---
	assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
	mockEntrySvc.AssertExpectations(t)
}

// --- REFACTOR: TestGetEntryMeta now uses mockEntrySvc.GetEntry ---
func TestGetEntryMeta(t *testing.T) {
	server, mockEntrySvc, mockDbSvc, cleanup := setupEntryHandlerTestAPI(t)
	defer cleanup()

	dbName := "TestDB"
	entryID := int64(1)

	// 1. Mock GetDatabase (This is still correct, handler needs it for custom fields)
	mockDB := &models.Database{
		Name:         dbName,
		ContentType:  "image",
		CustomFields: models.CustomFields{{Name: "location", Type: "TEXT"}},
	}
	mockDbSvc.On("GetDatabase", dbName).Return(mockDB, nil).Once()

	// 2. Mock GetEntry (from EntryService)
	mockEntry := models.Entry{
		"id":       entryID,
		"location": "Test Location",
		"status":   "ready", // <-- ADDED
	}
	// --- THIS IS THE FIX ---
	// We explicitly cast mockDB.CustomFields to the slice type
	// []models.CustomField to match the interface signature.
	mockEntrySvc.On("GetEntry", dbName, entryID, []models.CustomField(mockDB.CustomFields)).Return(mockEntry, nil).Once()
	// --- END FIX ---

	// 3. Make the request
	req, _ := http.NewRequest("GET", server.URL+fmt.Sprintf("/entry/meta?database_name=%s&id=%d", dbName, entryID), nil)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err, "HTTP request failed") // <-- Added error message

	// 4. Assertions
	// Check for panic first
	if resp == nil {
		t.Fatal("Response was nil, indicating a panic occurred in the server")
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var entryResp models.Entry
	err = json.NewDecoder(resp.Body).Decode(&entryResp)
	assert.NoError(t, err, "Failed to decode JSON response")
	assert.Equal(t, "Test Location", entryResp["location"])

	// Assert that *both* mocks were called as expected
	mockDbSvc.AssertExpectations(t)
	mockEntrySvc.AssertExpectations(t)
}

// --- ADDED: Test for GetEntry file download headers ---
func TestGetEntry_WithFilename(t *testing.T) {
	server, mockEntrySvc, _, cleanup := setupEntryHandlerTestAPI(t)
	defer cleanup()

	// We need a real, temporary file for http.ServeFile to work
	tmpfile, err := os.CreateTemp("", "test-*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name()) // Clean up the temp file
	tmpfile.Write([]byte("dummy content"))
	tmpfile.Close()

	// 1. Mock GetEntryFile to return the path to our temp file
	mockEntrySvc.On("GetEntryFile", "TestDB_File", int64(2)).Return(
		tmpfile.Name(),    // path
		"text/plain",      // mime_type
		"my_download.txt", // filename
		nil,               // error
	).Once()

	// 2. Make the request to the test server
	req, _ := http.NewRequest("GET", server.URL+fmt.Sprintf("/entry/file?database_name=%s&id=%d", "TestDB_File", 2), nil)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err, "HTTP request failed")

	// 3. Assertions
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))
	assert.Equal(t, "attachment; filename=\"my_download.txt\"", resp.Header.Get("Content-Disposition"))

	// Check body content
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "dummy content", string(bodyBytes))

	mockEntrySvc.AssertExpectations(t)
}
