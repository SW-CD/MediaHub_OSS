package postgres

import (
	"errors"
	"strings"
	"testing"
	"time"

	"mediahub_oss/internal/media"
	repo "mediahub_oss/internal/repository"

	"github.com/Masterminds/squirrel"
	"github.com/lib/pq"
)

// TestInterfaceImplementation verifies PostgresRepository implements repository.Repository interface.
func TestInterfaceImplementation(t *testing.T) {
	var _ repo.Repository = (*PostgresRepository)(nil)
}

func TestBuildDynamicTableSchema(t *testing.T) {
	r := &PostgresRepository{
		Builder:         squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		AllowedStatuses: repo.GetAllEntryStatuses(),
		MediaFields: map[string][]MediaField{
			"image": {
				{Name: "width", PostgresType: "INTEGER"},
				{Name: "height", PostgresType: "INTEGER"},
			},
		},
	}

	customFields := []repo.CustomFieldDef{
		{ID: 0, Name: "artist", Type: "TEXT", IsIndexed: true},
		{ID: 1, Name: "year", Type: "INTEGER", IsIndexed: false},
		{ID: 2, Name: "rating", Type: "REAL", IsIndexed: true},
		{ID: 3, Name: "is_favorite", Type: "BOOLEAN", IsIndexed: false},
	}

	sql, err := r.BuildDynamicTableSchema("01HGFB9Z5W7ABCDEFGHJKMNPQR", "image", customFields)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(sql, `"entries_01HGFB9Z5W7ABCDEFGHJKMNPQR"`) {
		t.Errorf("expected quoted table name in SQL, got: %s", sql)
	}
	if !strings.Contains(sql, `"cf_0" TEXT`) {
		t.Errorf("expected cf_0 column in SQL, got: %s", sql)
	}
	if !strings.Contains(sql, `"cf_1" INTEGER`) {
		t.Errorf("expected cf_1 column in SQL, got: %s", sql)
	}
	if !strings.Contains(sql, `"cf_2" DOUBLE PRECISION`) {
		t.Errorf("expected cf_2 column in SQL, got: %s", sql)
	}
	if !strings.Contains(sql, `"cf_3" BOOLEAN`) {
		t.Errorf("expected cf_3 column in SQL, got: %s", sql)
	}
	if !strings.Contains(sql, `id SERIAL PRIMARY KEY`) {
		t.Errorf("expected SERIAL id column in SQL, got: %s", sql)
	}
}

func TestPQUniqueViolation(t *testing.T) {
	err1 := &pq.Error{Code: "23505"}
	if !isPQUniqueViolation(err1) {
		t.Errorf("expected isPQUniqueViolation to return true for 23505 error")
	}

	err2 := errors.New("duplicate key value violates unique constraint")
	if !isPQUniqueViolation(err2) {
		t.Errorf("expected isPQUniqueViolation to return true for text match")
	}

	err3 := errors.New("some other error")
	if isPQUniqueViolation(err3) {
		t.Errorf("expected isPQUniqueViolation to return false")
	}
}

func TestValidOperator(t *testing.T) {
	validOps := []string{"=", "!=", ">", ">=", "<", "<=", "LIKE", "ILIKE", "like", "ilike"}
	for _, op := range validOps {
		if !isValidOperator(op) {
			t.Errorf("expected operator %s to be valid", op)
		}
	}

	if isValidOperator("DROP") {
		t.Errorf("expected invalid operator to return false")
	}
}

func TestMapToPostgresType(t *testing.T) {
	tests := map[string]string{
		"INTEGER":  "INTEGER",
		"REAL":     "DOUBLE PRECISION",
		"TEXT":     "TEXT",
		"BOOLEAN":  "BOOLEAN",
		"uint64":   "BIGINT",
		"INT64":    "BIGINT",
		"float64":  "DOUBLE PRECISION",
		"uint8":    "SMALLINT",
		"SMALLINT": "SMALLINT",
	}
	for in, expected := range tests {
		got := mapToPostgresType(in)
		if got != expected {
			t.Errorf("mapToPostgresType(%s) = %s, expected %s", in, got, expected)
		}
	}
}

func TestAllMediaTypesConfigured(t *testing.T) {
	for _, contentType := range media.GetContentTypes() {
		_, err := media.GetMetadataFields(contentType)
		if err != nil {
			t.Errorf("failed to get metadata fields for content type %s: %v", contentType, err)
		}
	}
}

func TestAcquireLockQuery(t *testing.T) {
	r := &PostgresRepository{
		Builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}

	ttlMs := int64(30000)
	query, _, err := r.Builder.Insert("system_locks").
		Columns("lock_name", "locked_at", "locked_by", "expires_at").
		Values(
			"hk_lock",
			squirrel.Expr("(EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT"),
			"worker-1",
			squirrel.Expr("(EXTRACT(EPOCH FROM clock_timestamp()) * 1000 + ?)::BIGINT", ttlMs),
		).
		Suffix(`
            ON CONFLICT (lock_name) DO UPDATE 
            SET locked_at = EXCLUDED.locked_at, 
                locked_by = EXCLUDED.locked_by, 
                expires_at = EXCLUDED.expires_at 
            WHERE system_locks.expires_at < (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)
               OR system_locks.locked_by = EXCLUDED.locked_by
            RETURNING lock_name
        `).
		ToSql()

	if err != nil {
		t.Fatalf("failed to build query: %v", err)
	}

	if !strings.Contains(query, "clock_timestamp()") {
		t.Errorf("expected clock_timestamp() in query, got: %s", query)
	}
	if !strings.Contains(query, "system_locks.locked_by = EXCLUDED.locked_by") {
		t.Errorf("expected owner renewal check in query, got: %s", query)
	}
}

func TestCreateEntryQuery_DynamicTimestamp(t *testing.T) {
	r := &PostgresRepository{
		Builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}

	dbNowExpr := squirrel.Expr("(EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT")

	t.Run("zero timestamp uses clock_timestamp expression", func(t *testing.T) {
		insertData := map[string]any{
			"created_at":       dbNowExpr,
			"updated_at":       dbNowExpr,
			"timestamp":        dbNowExpr,
			"filesize":         1024,
			"preview_filesize": 0,
			"filename":         "test.jpg",
			"status":           repo.EntryStatusReady,
			"mime_type":        "image/jpeg",
		}

		query, args, err := r.Builder.Insert(`"entries_01HGFB9Z5W7ABCDEFGHJKMNPQR"`).
			SetMap(insertData).
			Suffix("RETURNING id, timestamp, created_at, updated_at").
			ToSql()

		if err != nil {
			t.Fatalf("failed to build query: %v", err)
		}

		if !strings.Contains(query, "clock_timestamp()") {
			t.Errorf("expected clock_timestamp() in query, got: %s", query)
		}
		if !strings.Contains(query, "RETURNING id, timestamp, created_at, updated_at") {
			t.Errorf("expected RETURNING clause in query, got: %s", query)
		}
		// Since created_at, updated_at, timestamp are raw expressions, args only contain filesize, preview_filesize, filename, status, mime_type
		if len(args) != 5 {
			t.Errorf("expected 5 args for raw timestamp expressions, got: %d (%v)", len(args), args)
		}
	})

	t.Run("explicit timestamp uses parameter binding", func(t *testing.T) {
		explicitTS := int64(1700000000000)
		insertData := map[string]any{
			"created_at":       dbNowExpr,
			"updated_at":       dbNowExpr,
			"timestamp":        explicitTS,
			"filesize":         1024,
			"preview_filesize": 0,
			"filename":         "test.jpg",
			"status":           repo.EntryStatusReady,
			"mime_type":        "image/jpeg",
		}

		query, args, err := r.Builder.Insert(`"entries_01HGFB9Z5W7ABCDEFGHJKMNPQR"`).
			SetMap(insertData).
			Suffix("RETURNING id, timestamp, created_at, updated_at").
			ToSql()

		if err != nil {
			t.Fatalf("failed to build query: %v", err)
		}

		if !strings.Contains(query, "RETURNING id, timestamp, created_at, updated_at") {
			t.Errorf("expected RETURNING clause in query, got: %s", query)
		}

		if len(args) != 6 {
			t.Errorf("expected 6 args when timestamp is provided, got: %d (%v)", len(args), args)
		}

		foundTS := false
		for _, arg := range args {
			if arg == explicitTS {
				foundTS = true
				break
			}
		}
		if !foundTS {
			t.Errorf("expected explicit timestamp %d in args, got: %v", explicitTS, args)
		}
	})
}

func TestDynamicTimestampQueries(t *testing.T) {
	r := &PostgresRepository{
		Builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}

	t.Run("UpdateEntriesStatus query", func(t *testing.T) {
		query, _, err := r.Builder.Update(`"entries_01HGFB9Z5W7ABCDEFGHJKMNPQR"`).
			Set("status", repo.EntryStatusReady).
			Set("updated_at", squirrel.Expr("(EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT")).
			Where(squirrel.Eq{"id": []int64{1, 2, 3}}).
			ToSql()
		if err != nil {
			t.Fatalf("failed to build query: %v", err)
		}
		if !strings.Contains(query, "clock_timestamp()") {
			t.Errorf("expected clock_timestamp() in UpdateEntriesStatus query, got: %s", query)
		}
	})

	t.Run("ClaimQueuedEntry query", func(t *testing.T) {
		query, _, err := r.Builder.Update(`"entries_01HGFB9Z5W7ABCDEFGHJKMNPQR"`).
			Set("status", repo.EntryStatusProcessing).
			Set("updated_at", squirrel.Expr("(EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT")).
			Where(squirrel.Eq{"id": int64(10), "status": repo.EntryStatusQueued}).
			ToSql()
		if err != nil {
			t.Fatalf("failed to build query: %v", err)
		}
		if !strings.Contains(query, "clock_timestamp()") {
			t.Errorf("expected clock_timestamp() in ClaimQueuedEntry query, got: %s", query)
		}
	})

	t.Run("UpdateAPIKeyLastUsed query", func(t *testing.T) {
		query, args, err := r.Builder.Update("api_keys").
			Set("last_used_at", squirrel.Expr("(EXTRACT(EPOCH FROM clock_timestamp()) * 1000 - ?)::BIGINT", int64(5000))).
			Where(squirrel.Eq{"id": "01HGFB9Z5W7ABCDEFGHJKMNPQR"}).
			ToSql()
		if err != nil {
			t.Fatalf("failed to build query: %v", err)
		}
		if !strings.Contains(query, "clock_timestamp()") {
			t.Errorf("expected clock_timestamp() in UpdateAPIKeyLastUsed query, got: %s", query)
		}
		if len(args) != 2 || args[0] != int64(5000) {
			t.Errorf("expected lastUsed duration in args, got: %v", args)
		}
	})

	t.Run("DeleteExpiredAPIKeys query", func(t *testing.T) {
		query, _, err := r.Builder.Delete("api_keys").
			Where(squirrel.Expr("expires_at IS NOT NULL AND expires_at < (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)")).
			ToSql()
		if err != nil {
			t.Fatalf("failed to build query: %v", err)
		}
		if !strings.Contains(query, "clock_timestamp()") {
			t.Errorf("expected clock_timestamp() in DeleteExpiredAPIKeys query, got: %s", query)
		}
	})

	t.Run("StoreRefreshToken query", func(t *testing.T) {
		query, args, err := r.Builder.Insert("refresh_tokens").
			Columns("user_id", "token_hash", "expiry").
			Values("user1", "hash1", squirrel.Expr("(EXTRACT(EPOCH FROM clock_timestamp()) * 1000 + ?)::BIGINT", int64(3600000))).
			ToSql()
		if err != nil {
			t.Fatalf("failed to build query: %v", err)
		}
		if !strings.Contains(query, "clock_timestamp()") {
			t.Errorf("expected clock_timestamp() in StoreRefreshToken query, got: %s", query)
		}
		if len(args) != 3 {
			t.Errorf("expected 3 args, got: %v", args)
		}
	})

	t.Run("ValidateRefreshToken query", func(t *testing.T) {
		query, args, err := r.Builder.Select("user_id").
			From("refresh_tokens").
			Where(squirrel.Expr("token_hash = ? AND expiry >= (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)", "hash1")).
			ToSql()
		if err != nil {
			t.Fatalf("failed to build query: %v", err)
		}
		if !strings.Contains(query, "clock_timestamp()") {
			t.Errorf("expected clock_timestamp() in ValidateRefreshToken query, got: %s", query)
		}
		if len(args) != 1 || args[0] != "hash1" {
			t.Errorf("expected hash1 in args, got: %v", args)
		}
	})

	t.Run("DeleteExpiredRefreshTokens query", func(t *testing.T) {
		query, _, err := r.Builder.Delete("refresh_tokens").
			Where(squirrel.Expr("expiry < (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)")).
			ToSql()
		if err != nil {
			t.Fatalf("failed to build query: %v", err)
		}
		if !strings.Contains(query, "clock_timestamp()") {
			t.Errorf("expected clock_timestamp() in DeleteExpiredRefreshTokens query, got: %s", query)
		}
	})

	t.Run("DeleteLogs query", func(t *testing.T) {
		query, args, err := r.Builder.Delete("audit_logs").
			Where(squirrel.Expr("timestamp < (EXTRACT(EPOCH FROM clock_timestamp()) * 1000 - ?)", int64(86400000))).
			ToSql()
		if err != nil {
			t.Fatalf("failed to build query: %v", err)
		}
		if !strings.Contains(query, "clock_timestamp()") {
			t.Errorf("expected clock_timestamp() in DeleteLogs query, got: %s", query)
		}
		if len(args) != 1 || args[0] != int64(86400000) {
			t.Errorf("expected maxAge in args, got: %v", args)
		}
	})

	t.Run("HouseKeepingWasCalled query", func(t *testing.T) {
		query, _, err := r.Builder.Update("databases").
			Set("hk_last_run", squirrel.Expr("(EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT")).
			Where(squirrel.Eq{"id": "01HGFB9Z5W7ABCDEFGHJKMNPQR"}).
			Suffix("RETURNING hk_last_run").
			ToSql()
		if err != nil {
			t.Fatalf("failed to build query: %v", err)
		}
		if !strings.Contains(query, "clock_timestamp()") {
			t.Errorf("expected clock_timestamp() in HouseKeepingWasCalled query, got: %s", query)
		}
		if !strings.Contains(query, "RETURNING hk_last_run") {
			t.Errorf("expected RETURNING in HouseKeepingWasCalled query, got: %s", query)
		}
	})
}

func TestPostgres_TimeFilterQueryBuilder(t *testing.T) {
	r := &PostgresRepository{
		Builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}

	t.Run("GetEntries with epoch 0 and pre-1970 timestamp builds WHERE clauses", func(t *testing.T) {
		tStart := time.Date(1950, 1, 1, 0, 0, 0, 0, time.UTC)
		tEnd := time.Unix(0, 0).UTC() // Epoch = 0

		opts := repo.QueryOptions{
			TStart:    tStart,
			TEnd:      tEnd,
			TimeField: "timestamp",
		}
		if err := opts.Validate(); err != nil {
			t.Fatalf("opts validate failed: %v", err)
		}

		builder := r.Builder.Select("*").From(`"entries_01HGFB9Z5W7ABCDEFGHJKMNPQR"`)
		if !opts.TStart.IsZero() {
			builder = builder.Where(squirrel.GtOrEq{opts.TimeField: opts.TStart.UnixMilli()})
		}
		if !opts.TEnd.IsZero() {
			builder = builder.Where(squirrel.LtOrEq{opts.TimeField: opts.TEnd.UnixMilli()})
		}

		query, args, err := builder.ToSql()
		if err != nil {
			t.Fatalf("failed to build query: %v", err)
		}

		if !strings.Contains(query, "timestamp >=") || !strings.Contains(query, "timestamp <=") {
			t.Errorf("expected timestamp >= and <= in query, got: %s", query)
		}
		if len(args) != 2 {
			t.Fatalf("expected 2 args, got %d (%v)", len(args), args)
		}
		if args[0] != tStart.UnixMilli() || args[1] != int64(0) {
			t.Errorf("expected args [%d, 0], got %v", tStart.UnixMilli(), args)
		}
	})

	t.Run("GetLogs with epoch 0 builds WHERE clauses", func(t *testing.T) {
		tEpoch := time.Unix(0, 0).UTC()
		opts := repo.QueryOptions{
			TStart: tEpoch,
			TEnd:   tEpoch,
		}
		if err := opts.Validate(); err != nil {
			t.Fatalf("opts validate failed: %v", err)
		}

		builder := r.Builder.Select("id", "timestamp", "action", "actor", "resource", "details").From("audit_logs")
		if !opts.TStart.IsZero() {
			builder = builder.Where(squirrel.GtOrEq{"timestamp": opts.TStart.UnixMilli()})
		}
		if !opts.TEnd.IsZero() {
			builder = builder.Where(squirrel.LtOrEq{"timestamp": opts.TEnd.UnixMilli()})
		}

		query, args, err := builder.ToSql()
		if err != nil {
			t.Fatalf("failed to build query: %v", err)
		}

		if !strings.Contains(query, "timestamp >=") || !strings.Contains(query, "timestamp <=") {
			t.Errorf("expected timestamp >= and <= in query, got: %s", query)
		}
		if len(args) != 2 || args[0] != int64(0) || args[1] != int64(0) {
			t.Errorf("expected args [0, 0], got %v", args)
		}
	})
}

