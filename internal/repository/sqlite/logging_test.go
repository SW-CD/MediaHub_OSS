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

func TestGetLogs_TimeFilterEpochAndHistorical(t *testing.T) {
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

	t1950 := time.Date(1950, 1, 1, 12, 0, 0, 0, time.UTC)
	t1970 := time.Unix(0, 0).UTC()
	t2025 := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	// Insert audit logs directly with distinct timestamps
	_, err = r.DB.ExecContext(ctx,
		`INSERT INTO audit_logs (timestamp, action, actor, resource, details) VALUES (?, ?, ?, ?, ?)`,
		t1950.UnixMilli(), "CREATE", "admin", "db1", `{"msg":"historical 1950 log"}`,
	)
	if err != nil {
		t.Fatalf("failed to insert 1950 log: %v", err)
	}

	_, err = r.DB.ExecContext(ctx,
		`INSERT INTO audit_logs (timestamp, action, actor, resource, details) VALUES (?, ?, ?, ?, ?)`,
		t1970.UnixMilli(), "CREATE", "admin", "db1", `{"msg":"epoch 1970 log"}`,
	)
	if err != nil {
		t.Fatalf("failed to insert 1970 log: %v", err)
	}

	_, err = r.DB.ExecContext(ctx,
		`INSERT INTO audit_logs (timestamp, action, actor, resource, details) VALUES (?, ?, ?, ?, ?)`,
		t2025.UnixMilli(), "CREATE", "admin", "db1", `{"msg":"future 2025 log"}`,
	)
	if err != nil {
		t.Fatalf("failed to insert 2025 log: %v", err)
	}

	// 1. Filter with TStart = 1940 and TEnd = 1970 -> Should return 1950 and 1970 logs
	t1940 := time.Date(1940, 1, 1, 0, 0, 0, 0, time.UTC)
	logs, err := r.GetLogs(ctx, repo.QueryOptions{
		TStart: t1940,
		TEnd:   t1970,
	})
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs for range [1940, 1970], got %d", len(logs))
	}

	// 2. Filter with exact epoch timestamp TStart = 1970, TEnd = 1970 -> Should return only 1970 log
	logsEpoch, err := r.GetLogs(ctx, repo.QueryOptions{
		TStart: t1970,
		TEnd:   t1970,
	})
	if err != nil {
		t.Fatalf("GetLogs for epoch failed: %v", err)
	}
	if len(logsEpoch) != 1 || logsEpoch[0].Details["msg"] != "epoch 1970 log" {
		t.Fatalf("expected only epoch 1970 log, got %v", logsEpoch)
	}
}
