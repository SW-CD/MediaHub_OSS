package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mediahub_oss/internal/httpserver/auth"
	"mediahub_oss/internal/httpserver/utils"
	repo "mediahub_oss/internal/repository"
)

type mockPermHolder struct {
	isGlobalAdmin bool
	userULID      repo.ULID
	hasPerm       bool
}

func (m *mockPermHolder) IsGlobalAdmin() bool {
	return m.isGlobalAdmin
}

func (m *mockPermHolder) HasPermission(database repo.ULID, ag repo.AccessGrant) bool {
	return m.hasPerm
}

func (m *mockPermHolder) GetUserULID() repo.ULID {
	return m.userULID
}

func (m *mockPermHolder) GetAllPermissions(ctx context.Context) (map[repo.ULID]repo.AccessGrant, error) {
	return map[repo.ULID]repo.AccessGrant{}, nil
}

func TestAuthMiddleware_JSONErrorResponses(t *testing.T) {
	am := auth.NewAuthMiddleware(nil, "test_jwt_secret_key_123456789012")

	handler := am.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("missing auth header returns 401 JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
		if ctype := rec.Header().Get("Content-Type"); ctype != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %q", ctype)
		}

		var errResp utils.ErrorResponse
		if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
			t.Fatalf("failed to decode JSON error response: %v", err)
		}
		if errResp.Error == "" {
			t.Fatalf("expected non-empty error message in JSON payload")
		}
	})

	t.Run("invalid credentials format returns 401 JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set("Authorization", "InvalidHeaderFormat")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
		if ctype := rec.Header().Get("Content-Type"); ctype != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %q", ctype)
		}

		var errResp utils.ErrorResponse
		if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
			t.Fatalf("failed to decode JSON error response: %v", err)
		}
		if errResp.Error == "" {
			t.Fatalf("expected non-empty error message in JSON payload")
		}
	})
}

func TestRequireGlobalAdmin_JSONErrorResponse(t *testing.T) {
	am := auth.NewAuthMiddleware(nil, "test_jwt_secret_key_123456789012")

	handler := am.RequireGlobalAdmin()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("non-admin returns 403 JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/admin", nil)
		holder := &mockPermHolder{isGlobalAdmin: false}
		req = req.WithContext(context.WithValue(req.Context(), utils.PermissionHolderKey, holder))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status 403, got %d", rec.Code)
		}
		if ctype := rec.Header().Get("Content-Type"); ctype != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %q", ctype)
		}

		var errResp utils.ErrorResponse
		if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
			t.Fatalf("failed to decode JSON error response: %v", err)
		}
		if errResp.Error != "Forbidden: Admin access required" {
			t.Fatalf("unexpected error message: %q", errResp.Error)
		}
	})
}

func TestRequireDatabasePermission_JSONErrorResponse(t *testing.T) {
	am := auth.NewAuthMiddleware(nil, "test_jwt_secret_key_123456789012")

	handler := am.RequireDatabasePermission(repo.AccessView)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("missing database_id returns 400 JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/database", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
		if ctype := rec.Header().Get("Content-Type"); ctype != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %q", ctype)
		}

		var errResp utils.ErrorResponse
		if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
			t.Fatalf("failed to decode JSON error response: %v", err)
		}
		if errResp.Error != "Bad Request: Missing database context" {
			t.Fatalf("unexpected error message: %q", errResp.Error)
		}
	})

	t.Run("lacking database permission returns 403 JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/database/01HGFB9Z5W7ABCDEFGHJKMNPQR", nil)
		req.SetPathValue("database_id", "01HGFB9Z5W7ABCDEFGHJKMNPQR")
		holder := &mockPermHolder{isGlobalAdmin: false, hasPerm: false}
		req = req.WithContext(context.WithValue(req.Context(), utils.PermissionHolderKey, holder))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status 403, got %d", rec.Code)
		}
		if ctype := rec.Header().Get("Content-Type"); ctype != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %q", ctype)
		}

		var errResp utils.ErrorResponse
		if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
			t.Fatalf("failed to decode JSON error response: %v", err)
		}
		if errResp.Error == "" {
			t.Fatalf("expected non-empty error message")
		}
	})
}
