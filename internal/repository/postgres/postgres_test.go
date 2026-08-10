package postgres

import (
	"errors"
	"strings"
	"testing"

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
