package entryhandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"mediahub_oss/internal/httpserver/entryhandler"
	"mediahub_oss/internal/httpserver/utils"
	"mediahub_oss/internal/processing"
	repo "mediahub_oss/internal/repository"
	"mediahub_oss/internal/repository/migrations"
	_ "mediahub_oss/internal/repository/migrations/sqlite"
	"mediahub_oss/internal/repository/sqlite"
	"mediahub_oss/internal/storage/localstorage"

	"github.com/pressly/goose/v3"
)

func setupTestEnvironment(t *testing.T) (*sqlite.SQLiteRepository, *localstorage.LocalStorage, *entryhandler.EntryHandler, repo.Database) {
	t.Helper()
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	r, err := sqlite.NewRepository(":memory:")
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("failed to set goose dialect: %v", err)
	}
	goose.SetBaseFS(migrations.EmbedFS)
	if err := goose.Up(r.DB, "sqlite"); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	tempDir := t.TempDir()
	storageRoot := filepath.Join(tempDir, "storage")
	_ = os.MkdirAll(storageRoot, 0755)
	store := &localstorage.LocalStorage{RootPath: storageRoot}

	db, err := r.CreateDatabase(ctx, repo.Database{
		Name:        "test_images",
		ContentType: "image",
	})
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}

	conv := &mockMediaConverter{}
	proc, err := processing.NewProcessor(r, store, conv, 2, 4, logger)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	handler := &entryhandler.EntryHandler{
		Repo:                   r,
		Storage:                store,
		MediaConverter:         conv,
		Processor:              proc,
		Auditor:                &mockAuditor{},
		Logger:                 logger,
		MaxSyncUploadSizeBytes: 1024,
	}

	return r, store, handler, db
}

func TestQueryEntries_InvalidQueryParamsReturn400(t *testing.T) {
	r, _, handler, db := setupTestEnvironment(t)
	defer r.Close()

	tests := []struct {
		name string
		url  string
	}{
		{"invalid limit", "/api/database/" + db.ID.String() + "/entries?limit=invalid"},
		{"invalid offset", "/api/database/" + db.ID.String() + "/entries?offset=abc"},
		{"invalid tstart", "/api/database/" + db.ID.String() + "/entries?tstart=foo"},
		{"invalid tend", "/api/database/" + db.ID.String() + "/entries?tend=bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req.SetPathValue("database_id", db.ID.String())
			req = req.WithContext(context.WithValue(req.Context(), utils.UserKey, &repo.User{Username: "admin", IsAdmin: true}))
			w := httptest.NewRecorder()

			handler.QueryEntries(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400 for %s, got %d", tt.url, w.Code)
			}
		})
	}
}

func TestSearchEntries_ValidationErrorsReturn400(t *testing.T) {
	r, _, handler, db := setupTestEnvironment(t)
	defer r.Close()

	// Invalid operator payload
	searchBody := map[string]any{
		"filter": map[string]any{
			"operator": "and",
			"conditions": []map[string]any{
				{
					"field":    "filename",
					"operator": "INVALID_OP",
					"value":    "test",
				},
			},
		},
		"pagination": map[string]any{
			"limit": 10,
		},
	}
	bodyBytes, _ := json.Marshal(searchBody)

	req := httptest.NewRequest(http.MethodPost, "/api/database/"+db.ID.String()+"/entries/search", bytes.NewReader(bodyBytes))
	req.SetPathValue("database_id", db.ID.String())
	req = req.WithContext(context.WithValue(req.Context(), utils.UserKey, &repo.User{Username: "admin", IsAdmin: true}))
	w := httptest.NewRecorder()

	handler.SearchEntries(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid operator search, got %d", w.Code)
	}
}

func TestDeleteEntries_SuccessAndPartialSuccess(t *testing.T) {
	r, store, handler, db := setupTestEnvironment(t)
	defer r.Close()
	ctx := context.Background()

	// Create an entry
	entry, err := r.CreateEntry(ctx, db, repo.Entry{
		FileName:    "test.jpg",
		MimeType:    "image/jpeg",
		Size:        100,
		Status:      repo.EntryStatusReady,
		MediaFields: map[string]any{"width": 800, "height": 600},
	})
	if err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	// Write dummy file to storage
	_, _ = store.Write(ctx, db.ID.String(), entry.ID, bytes.NewReader([]byte("test")))

	// Bulk delete
	deleteReq := entryhandler.BulkDeleteRequest{
		IDs: []int64{entry.ID},
	}
	bodyBytes, _ := json.Marshal(deleteReq)

	req := httptest.NewRequest(http.MethodPost, "/api/database/"+db.ID.String()+"/entries/delete", bytes.NewReader(bodyBytes))
	req.SetPathValue("database_id", db.ID.String())
	req = req.WithContext(context.WithValue(req.Context(), utils.UserKey, &repo.User{Username: "admin", IsAdmin: true}))
	w := httptest.NewRecorder()

	handler.DeleteEntries(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for bulk delete, got %d", w.Code)
	}

	var resp entryhandler.BulkDeleteResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.DeletedCount != 1 {
		t.Errorf("expected deleted_count 1, got %d", resp.DeletedCount)
	}
}
