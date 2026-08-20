package sqlite_test

import (
	"context"
	"testing"
	"time"

	repo "mediahub_oss/internal/repository"
	"mediahub_oss/internal/repository/migrations"
	_ "mediahub_oss/internal/repository/migrations/sqlite"
	"mediahub_oss/internal/repository/sqlite"

	"github.com/pressly/goose/v3"
)

func TestCreateEntry_ZeroTimestamp(t *testing.T) {
	ctx := context.Background()

	// 1. Initialize SQLite repository in memory
	r, err := sqlite.NewRepository(":memory:")
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer r.Close()

	// 2. Run migrations
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("failed to set goose dialect: %v", err)
	}
	goose.SetBaseFS(migrations.EmbedFS)
	if err := goose.Up(r.DB, "sqlite"); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// 3. Create a test database
	dbModel := repo.Database{
		Name:        "test_images",
		ContentType: "image",
	}
	createdDB, err := r.CreateDatabase(ctx, dbModel)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	before := time.Now().Add(-1 * time.Second)

	// 4. Create an entry with zero Timestamp
	entry := repo.Entry{
		FileName: "test.jpg",
		MimeType: "image/jpeg",
		Size:     1024,
		Status:   repo.EntryStatusReady,
		MediaFields: map[string]any{
			"width":  800,
			"height": 600,
		},
		// Timestamp is left zero time.Time{}
	}

	createdEntry, err := r.CreateEntry(ctx, createdDB, entry)
	if err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	after := time.Now().Add(1 * time.Second)

	// Check returned entry timestamp
	if createdEntry.Timestamp.IsZero() {
		t.Errorf("expected createdEntry.Timestamp to be populated, got zero time")
	}
	if createdEntry.Timestamp.Before(before) || createdEntry.Timestamp.After(after) {
		t.Errorf("expected createdEntry.Timestamp to be around now, got %v (unix milli: %d)", createdEntry.Timestamp, createdEntry.Timestamp.UnixMilli())
	}
	if createdEntry.Timestamp.UnixMilli() < 0 {
		t.Errorf("timestamp was written as negative epoch (year 0001): %d", createdEntry.Timestamp.UnixMilli())
	}

	// Fetch entry from DB and verify persisted timestamp
	fetchedEntry, err := r.GetEntry(ctx, createdDB.ID, createdEntry.ID)
	if err != nil {
		t.Fatalf("failed to get entry: %v", err)
	}
	if fetchedEntry.Timestamp.UnixMilli() < 0 {
		t.Errorf("persisted timestamp was negative epoch (year 0001): %d", fetchedEntry.Timestamp.UnixMilli())
	}
	if fetchedEntry.Timestamp.Before(before) || fetchedEntry.Timestamp.After(after) {
		t.Errorf("expected fetchedEntry.Timestamp to be around now, got %v (unix milli: %d)", fetchedEntry.Timestamp, fetchedEntry.Timestamp.UnixMilli())
	}
}

func TestGetEntries_TimeFilterEpochAndHistorical(t *testing.T) {
	ctx := context.Background()

	r, err := sqlite.NewRepository(":memory:")
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer r.Close()

	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("failed to set goose dialect: %v", err)
	}
	goose.SetBaseFS(migrations.EmbedFS)
	if err := goose.Up(r.DB, "sqlite"); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	dbModel := repo.Database{
		Name:        "historical_archive",
		ContentType: "image",
	}
	createdDB, err := r.CreateDatabase(ctx, dbModel)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	t1950 := time.Date(1950, 1, 1, 12, 0, 0, 0, time.UTC)
	t1970 := time.Unix(0, 0).UTC() // Epoch = 0
	t2025 := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	// Create 3 entries with specific timestamps
	_, err = r.CreateEntry(ctx, createdDB, repo.Entry{
		FileName:    "photo_1950.jpg",
		MimeType:    "image/jpeg",
		Size:        1000,
		Status:      repo.EntryStatusReady,
		Timestamp:   t1950,
		MediaFields: map[string]any{"width": 800, "height": 600},
	})
	if err != nil {
		t.Fatalf("failed to create 1950 entry: %v", err)
	}

	_, err = r.CreateEntry(ctx, createdDB, repo.Entry{
		FileName:    "photo_1970.jpg",
		MimeType:    "image/jpeg",
		Size:        2000,
		Status:      repo.EntryStatusReady,
		Timestamp:   t1970,
		MediaFields: map[string]any{"width": 800, "height": 600},
	})
	if err != nil {
		t.Fatalf("failed to create 1970 entry: %v", err)
	}

	_, err = r.CreateEntry(ctx, createdDB, repo.Entry{
		FileName:    "photo_2025.jpg",
		MimeType:    "image/jpeg",
		Size:        3000,
		Status:      repo.EntryStatusReady,
		Timestamp:   t2025,
		MediaFields: map[string]any{"width": 800, "height": 600},
	})
	if err != nil {
		t.Fatalf("failed to create 2025 entry: %v", err)
	}

	// 1. Filter with TStart = 1940 and TEnd = 1970 (Epoch 0) -> Should return 1950 and 1970 entries
	t1940 := time.Date(1940, 1, 1, 0, 0, 0, 0, time.UTC)
	results, err := r.GetEntries(ctx, createdDB.ID, repo.QueryOptions{
		TStart: t1940,
		TEnd:   t1970,
	})
	if err != nil {
		t.Fatalf("GetEntries failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 entries for range [1940, 1970], got %d", len(results))
	}

	// 2. Filter with exact epoch timestamp TStart = 1970, TEnd = 1970 -> Should return only 1970 entry
	resultsEpoch, err := r.GetEntries(ctx, createdDB.ID, repo.QueryOptions{
		TStart: t1970,
		TEnd:   t1970,
	})
	if err != nil {
		t.Fatalf("GetEntries for epoch failed: %v", err)
	}
	if len(resultsEpoch) != 1 || resultsEpoch[0].FileName != "photo_1970.jpg" {
		t.Fatalf("expected only photo_1970.jpg, got %v", resultsEpoch)
	}
}

func TestDeleteDatabase_InvalidatesCustomFieldsCache(t *testing.T) {
	ctx := context.Background()

	r, err := sqlite.NewRepository(":memory:")
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer r.Close()

	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("failed to set goose dialect: %v", err)
	}
	goose.SetBaseFS(migrations.EmbedFS)
	if err := goose.Up(r.DB, "sqlite"); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	dbModel := repo.Database{
		Name:        "test_cf_db",
		ContentType: "image",
	}
	createdDB, err := r.CreateDatabase(ctx, dbModel)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	// Add custom field
	_, err = r.AddCustomField(ctx, createdDB.ID, repo.CustomFieldDef{
		Name: "photographer",
		Type: "string",
	})
	if err != nil {
		t.Fatalf("failed to add custom field: %v", err)
	}

	// Fetch custom fields to populate cache
	fields, err := r.GetCustomFields(ctx, createdDB.ID)
	if err != nil {
		t.Fatalf("failed to get custom fields: %v", err)
	}
	if len(fields) != 1 {
		t.Fatalf("expected 1 custom field, got %d", len(fields))
	}

	// Verify cache has the key
	cacheKey := "cf:" + createdDB.ID.String()
	if _, found := r.Cache.Get(cacheKey); !found {
		t.Fatalf("expected cache key %s to be populated", cacheKey)
	}

	// Delete database
	if err := r.DeleteDatabase(ctx, createdDB.ID); err != nil {
		t.Fatalf("failed to delete database: %v", err)
	}

	// Verify cache key was evicted
	if _, found := r.Cache.Get(cacheKey); found {
		t.Fatalf("expected cache key %s to be deleted after DeleteDatabase", cacheKey)
	}
}
