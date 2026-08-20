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
