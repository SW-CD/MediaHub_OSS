package recovery

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	repo "mediahub_oss/internal/repository"
	"mediahub_oss/internal/repository/migrations"
	_ "mediahub_oss/internal/repository/migrations/sqlite"
	"mediahub_oss/internal/repository/sqlite"
	"mediahub_oss/internal/storage/localstorage"

	"github.com/pressly/goose/v3"
)

func TestEntryStatusCorrection_ZeroStatsScan(t *testing.T) {
	ctx := context.Background()

	// Create temp dir for storage
	tempDir := t.TempDir()
	storageRoot := filepath.Join(tempDir, "storage")
	_ = os.MkdirAll(storageRoot, 0755)

	// In-memory sqlite repo
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
		Name:        "test_db",
		ContentType: "file",
	}
	createdDB, err := r.CreateDatabase(ctx, dbModel)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	// Create an entry in 'processing' status
	createdEntry, err := r.CreateEntry(ctx, createdDB, repo.Entry{
		FileName: "test.txt",
		MimeType: "text/plain",
		Size:     12,
		Status:   repo.EntryStatusProcessing,
	})
	if err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	// Create the physical file in storage so it gets marked 'ready'
	localStorage := &localstorage.LocalStorage{RootPath: storageRoot}
	_, err = localStorage.Write(ctx, createdDB.ID.String(), createdEntry.ID, bytes.NewReader([]byte("hello world!")))
	if err != nil {
		t.Fatalf("failed to write storage file: %v", err)
	}

	// Deliberately set stats.EntryCount to 0 in database metadata (simulating corrupted / zero stats)
	createdDB.Stats.EntryCount = 0
	_, err = r.UpdateDatabase(ctx, createdDB)
	if err != nil {
		t.Fatalf("failed to update db stats: %v", err)
	}

	// Verify stats.EntryCount is indeed 0
	stats, err := r.GetDatabaseStats(ctx, createdDB.ID)
	if err != nil || stats.EntryCount != 0 {
		t.Fatalf("expected stats.EntryCount == 0, got %d, err: %v", stats.EntryCount, err)
	}

	service := &RecoveryService{
		repo:    r,
		storage: localStorage,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		dryRun:  false,
	}

	// Run EntryStatusCorrection - should scan despite stats.EntryCount == 0 and correct status to 'ready'
	if err := service.EntryStatusCorrection(ctx); err != nil {
		t.Fatalf("EntryStatusCorrection failed: %v", err)
	}

	// Verify entry status is now 'ready'
	updatedEntry, err := r.GetEntry(ctx, createdDB.ID, createdEntry.ID)
	if err != nil {
		t.Fatalf("failed to fetch entry: %v", err)
	}
	if updatedEntry.Status != repo.EntryStatusReady {
		t.Errorf("expected entry status 'ready', got %q", updatedEntry.Status)
	}
}
