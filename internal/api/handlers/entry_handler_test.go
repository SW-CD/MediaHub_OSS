// filepath: internal/api/handlers/entry_handler_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mediahub/internal/config"
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

// setupEntryHandlerTestAPI creates a new test server for entry handlers.
func setupEntryHandlerTestAPI(t *testing.T) (*httptest.Server, *MockEntryService, *MockDatabaseService, *MockAuditor, func()) {
	t.Helper()

	mockEntrySvc := new(MockEntryService)
	mockDbSvc := new(MockDatabaseService)
	mockAuditor := new(MockAuditor)

	infoSvc := new(MockInfoService)
	infoSvc.On("GetInfo").Return(models.Info{
		Version:     "test",
		UptimeSince: time.Now(),
	})

	dummyCfg := &config.Config{
		MaxSyncUploadSizeBytes: 8 << 20, // 8MB default for tests
	}

	h := NewHandlers(
		infoSvc,
		nil,
		nil,
		mockDbSvc,
		mockEntrySvc,
		nil,
		mockAuditor, // <-- Inject Mock
		dummyCfg,
	)

	r := mux.NewRouter()
	r.HandleFunc("/entry", h.UploadEntry).Methods("POST")
	r.HandleFunc("/entry/meta", h.GetEntryMeta).Methods("GET")
	r.HandleFunc("/entry/file", h.GetEntry).Methods("GET")
	r.HandleFunc("/entry/preview", h.GetEntryPreview).Methods("GET")

	server := httptest.NewServer(r)

	cleanup := func() {
		server.Close()
	}

	return server, mockEntrySvc, mockDbSvc, mockAuditor, cleanup
}

func TestUploadEntry_Success(t *testing.T) {
	server, mockEntrySvc, _, mockAuditor, cleanup := setupEntryHandlerTestAPI(t)
	defer cleanup()

	// --- Prepare multipart form ---
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	metadata := map[string]interface{}{
		"description": "Test entry from API",
		"timestamp":   12345,
	}
	metadataBytes, _ := json.Marshal(metadata)
	metadataStr := string(metadataBytes)
	part, _ := writer.CreateFormField("metadata")
	part.Write(metadataBytes)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="dummy.png"`)
	h.Set("Content-Type", "image/png")
	entryPart, _ := writer.CreatePart(h)
	entryPart.Write([]byte("dummy png data"))
	writer.Close()

	// --- Mock the service call ---
	returnedEntry := models.Entry{
		"id":          int64(1),
		"description": "Test entry from API",
		"mime_type":   "image/png",
		"filename":    "dummy.png",
		"width":       float64(300),
		"height":      float64(200),
		"status":      "ready",
	}

	mockEntrySvc.On(
		"CreateEntry",
		"EntryAPITestDB",
		metadataStr,
		mock.Anything,
		mock.Anything,
	).Return(returnedEntry, http.StatusCreated, nil).Once()

	// --- Expect Audit Log call ---
	mockAuditor.On("Log",
		mock.Anything, // Context
		"entry.upload",
		mock.Anything, // Actor
		"EntryAPITestDB:1",
		mock.MatchedBy(func(details map[string]interface{}) bool {
			return details["filename"] == "dummy.png" && details["mode"] == "sync"
		}),
	).Return().Once()

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

	mockEntrySvc.AssertExpectations(t)
	mockAuditor.AssertExpectations(t)
}

func TestUploadEntry_DBNotFound(t *testing.T) {
	server, mockEntrySvc, _, mockAuditor, cleanup := setupEntryHandlerTestAPI(t)
	defer cleanup()

	// --- Prepare multipart form (minimal) ---
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("metadata", `{"timestamp": 12345}`)
	part, _ := writer.CreateFormFile("file", "dummy.png")
	part.Write([]byte("dummy data"))
	writer.Close()

	mockEntrySvc.On(
		"CreateEntry",
		"NotFoundDB",
		mock.AnythingOfType("string"),
		mock.Anything,
		mock.Anything,
	).Return(nil, http.StatusNotFound, services.ErrNotFound).Once()

	// No audit log expected on failure

	req, _ := http.NewRequest("POST", server.URL+"/entry?database_name=NotFoundDB", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	mockEntrySvc.AssertExpectations(t)
	mockAuditor.AssertNotCalled(t, "Log")
}

func TestUploadEntry_UnsupportedMedia(t *testing.T) {
	server, mockEntrySvc, _, mockAuditor, cleanup := setupEntryHandlerTestAPI(t)
	defer cleanup()

	// --- Prepare multipart form ---
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("metadata", `{"timestamp": 12345}`)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="dummy.txt"`)
	h.Set("Content-Type", "text/plain")
	part, _ := writer.CreatePart(h)
	part.Write([]byte("dummy data"))
	writer.Close()

	mockEntrySvc.On(
		"CreateEntry",
		"ImageDB",
		mock.AnythingOfType("string"),
		mock.Anything,
		mock.Anything,
	).Return(nil, http.StatusUnsupportedMediaType, services.ErrUnsupported).Once()

	// No audit log expected on failure

	req, _ := http.NewRequest("POST", server.URL+"/entry?database_name=ImageDB", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
	mockEntrySvc.AssertExpectations(t)
	mockAuditor.AssertNotCalled(t, "Log")
}

func TestGetEntryMeta(t *testing.T) {
	server, mockEntrySvc, mockDbSvc, _, cleanup := setupEntryHandlerTestAPI(t)
	defer cleanup()

	dbName := "TestDB"
	entryID := int64(1)

	mockDB := &models.Database{
		Name:         dbName,
		ContentType:  "image",
		CustomFields: models.CustomFields{{Name: "location", Type: "TEXT"}},
	}
	mockDbSvc.On("GetDatabase", dbName).Return(mockDB, nil).Once()

	mockEntry := models.Entry{
		"id":       entryID,
		"location": "Test Location",
		"status":   "ready",
	}
	mockEntrySvc.On("GetEntry", dbName, entryID, []models.CustomField(mockDB.CustomFields)).Return(mockEntry, nil).Once()

	req, _ := http.NewRequest("GET", server.URL+fmt.Sprintf("/entry/meta?database_name=%s&id=%d", dbName, entryID), nil)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err, "HTTP request failed")

	if resp == nil {
		t.Fatal("Response was nil")
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var entryResp models.Entry
	err = json.NewDecoder(resp.Body).Decode(&entryResp)
	assert.NoError(t, err, "Failed to decode JSON response")
	assert.Equal(t, "Test Location", entryResp["location"])

	mockDbSvc.AssertExpectations(t)
	mockEntrySvc.AssertExpectations(t)
}

func TestGetEntry_WithFilename(t *testing.T) {
	server, mockEntrySvc, _, _, cleanup := setupEntryHandlerTestAPI(t)
	defer cleanup()

	tmpfile, err := os.CreateTemp("", "test-*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	tmpfile.Write([]byte("dummy content"))
	tmpfile.Close()

	mockEntrySvc.On("GetEntryFile", "TestDB_File", int64(2)).Return(
		tmpfile.Name(),    // path
		"text/plain",      // mime_type
		"my_download.txt", // filename
		nil,               // error
	).Once()

	req, _ := http.NewRequest("GET", server.URL+fmt.Sprintf("/entry/file?database_name=%s&id=%d", "TestDB_File", 2), nil)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err, "HTTP request failed")

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))
	assert.Equal(t, "attachment; filename=\"my_download.txt\"", resp.Header.Get("Content-Disposition"))

	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "dummy content", string(bodyBytes))

	mockEntrySvc.AssertExpectations(t)
}

func TestGetEntry_JSON(t *testing.T) {
	server, mockEntrySvc, _, _, cleanup := setupEntryHandlerTestAPI(t)
	defer cleanup()

	tmpfile, err := os.CreateTemp("", "test-*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	tmpfile.Write([]byte("Hello"))
	tmpfile.Close()

	mockEntrySvc.On("GetEntryFile", "TestDB", int64(1)).Return(
		tmpfile.Name(),
		"text/plain",
		"hello.txt",
		nil,
	).Once()

	req, _ := http.NewRequest("GET", server.URL+"/entry/file?database_name=TestDB&id=1", nil)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var jsonResp models.FileJSONResponse
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	assert.NoError(t, err)

	assert.Equal(t, "hello.txt", jsonResp.Filename)
	assert.Equal(t, "text/plain", jsonResp.MimeType)
	assert.Equal(t, "data:text/plain;base64,SGVsbG8=", jsonResp.Data)
}

func TestGetEntryPreview_JSON(t *testing.T) {
	server, mockEntrySvc, _, _, cleanup := setupEntryHandlerTestAPI(t)
	defer cleanup()

	tmpfile, err := os.CreateTemp("", "preview-*.jpg")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	tmpfile.Write([]byte("FakeImage"))
	tmpfile.Close()

	mockEntrySvc.On("GetEntryPreview", "TestDB", int64(10)).Return(tmpfile.Name(), nil).Once()

	req, _ := http.NewRequest("GET", server.URL+"/entry/preview?database_name=TestDB&id=10", nil)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var jsonResp models.FileJSONResponse
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	assert.NoError(t, err)

	assert.Equal(t, "10.jpg", jsonResp.Filename)
	assert.Equal(t, "image/jpeg", jsonResp.MimeType)
	assert.Equal(t, "data:image/jpeg;base64,RmFrZUltYWdl", jsonResp.Data)
}
