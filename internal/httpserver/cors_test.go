package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"mediahub_oss/internal/httpserver"
)

func TestCORSMiddleware(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	t.Run("No Origin header passes through without CORS headers", func(t *testing.T) {
		mw := httpserver.CORSMiddleware([]string{"https://example.com"})(dummyHandler)
		req := httptest.NewRequest(http.MethodGet, "/api/entries", nil)
		rec := httptest.NewRecorder()

		mw.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("expected no Access-Control-Allow-Origin, got %q", got)
		}
	})

	t.Run("Exact match returns matching origin and credentials allowed", func(t *testing.T) {
		allowed := []string{"https://trusted-app.com", "http://localhost:4200"}
		mw := httpserver.CORSMiddleware(allowed)(dummyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/entries", nil)
		req.Header.Set("Origin", "https://trusted-app.com")
		rec := httptest.NewRecorder()

		mw.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://trusted-app.com" {
			t.Errorf("expected Access-Control-Allow-Origin 'https://trusted-app.com', got %q", got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
			t.Errorf("expected Access-Control-Allow-Credentials 'true', got %q", got)
		}
	})

	t.Run("Wildcard * allows origin with * and DOES NOT set allow credentials", func(t *testing.T) {
		allowed := []string{"*"}
		mw := httpserver.CORSMiddleware(allowed)(dummyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/entries", nil)
		req.Header.Set("Origin", "https://any-site.org")
		rec := httptest.NewRecorder()

		mw.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("expected Access-Control-Allow-Origin '*', got %q", got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
			t.Errorf("expected NO Access-Control-Allow-Credentials for wildcard, got %q", got)
		}
	})

	t.Run("Exact origin preferred over wildcard if both are present", func(t *testing.T) {
		allowed := []string{"https://trusted-app.com", "*"}
		mw := httpserver.CORSMiddleware(allowed)(dummyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/entries", nil)
		req.Header.Set("Origin", "https://trusted-app.com")
		rec := httptest.NewRecorder()

		mw.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://trusted-app.com" {
			t.Errorf("expected Access-Control-Allow-Origin 'https://trusted-app.com', got %q", got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
			t.Errorf("expected Access-Control-Allow-Credentials 'true' for exact match, got %q", got)
		}
	})

	t.Run("Disallowed origin without wildcard receives no CORS headers and 403 on preflight", func(t *testing.T) {
		allowed := []string{"https://trusted-app.com"}
		mw := httpserver.CORSMiddleware(allowed)(dummyHandler)

		// Regular request
		req := httptest.NewRequest(http.MethodGet, "/api/entries", nil)
		req.Header.Set("Origin", "https://untrusted.com")
		rec := httptest.NewRecorder()

		mw.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("expected no Access-Control-Allow-Origin for untrusted, got %q", got)
		}

		// Preflight OPTIONS request
		preflightReq := httptest.NewRequest(http.MethodOptions, "/api/entries", nil)
		preflightReq.Header.Set("Origin", "https://untrusted.com")
		preflightRec := httptest.NewRecorder()

		mw.ServeHTTP(preflightRec, preflightReq)

		if preflightRec.Code != http.StatusForbidden {
			t.Errorf("expected status 403 on disallowed preflight, got %d", preflightRec.Code)
		}
	})

	t.Run("Preflight OPTIONS on allowed exact origin returns 200 OK with CORS headers", func(t *testing.T) {
		allowed := []string{"https://trusted-app.com"}
		mw := httpserver.CORSMiddleware(allowed)(dummyHandler)

		req := httptest.NewRequest(http.MethodOptions, "/api/entries", nil)
		req.Header.Set("Origin", "https://trusted-app.com")
		rec := httptest.NewRecorder()

		mw.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://trusted-app.com" {
			t.Errorf("expected Access-Control-Allow-Origin 'https://trusted-app.com', got %q", got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
			t.Errorf("expected Access-Control-Allow-Credentials 'true', got %q", got)
		}
	})

	t.Run("Preflight OPTIONS on wildcard origin returns 200 OK with wildcard and no credentials", func(t *testing.T) {
		allowed := []string{"*"}
		mw := httpserver.CORSMiddleware(allowed)(dummyHandler)

		req := httptest.NewRequest(http.MethodOptions, "/api/entries", nil)
		req.Header.Set("Origin", "https://anywhere.io")
		rec := httptest.NewRecorder()

		mw.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("expected Access-Control-Allow-Origin '*', got %q", got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
			t.Errorf("expected NO Access-Control-Allow-Credentials for wildcard preflight, got %q", got)
		}
	})
}
