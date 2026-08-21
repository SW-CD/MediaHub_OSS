package tokenhandler_test

import (
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mediahub_oss/internal/httpserver/tokenhandler"
	"mediahub_oss/internal/logging/audit"
	repo "mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared/customerrors"
	"golang.org/x/crypto/bcrypt"
)

type mockTokenRepo struct {
	repo.Repository
	user repo.User
}

func (m *mockTokenRepo) GetUserByUsername(ctx context.Context, username string) (repo.User, error) {
	if m.user.Username == username {
		return m.user, nil
	}
	return repo.User{}, customerrors.ErrNotFound
}

func (m *mockTokenRepo) StoreRefreshToken(ctx context.Context, userID repo.ULID, tokenHash string, duration time.Duration) error {
	return nil
}

func TestGetToken_ServiceAccountBasicAuthRejected(t *testing.T) {
	password := "service_secret_pass"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	mockRepo := &mockTokenRepo{
		user: repo.User{
			ID:               "01HGFB9Z5W7ABCDEFGHJKMNPQR",
			Username:         "service_worker",
			PasswordHash:     string(hash),
			IsServiceAccount: true,
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditor := audit.NewAuditLogger(false, "stdio", logger, mockRepo)

	handler := tokenhandler.TokenHandler{
		Logger:          logger,
		Auditor:         auditor,
		Repo:            mockRepo,
		JWTSecret:       []byte("test_secret_12345678901234567890"),
		AccessDuration:  5 * time.Minute,
		RefreshDuration: 24 * time.Hour,
	}

	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte("service_worker:"+password))
	req := httptest.NewRequest(http.MethodPost, "/api/token", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()

	handler.GetToken(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized for service account basic auth, got %d", rec.Code)
	}
}

func TestGetToken_RegularUserBasicAuthAccepted(t *testing.T) {
	password := "user_secret_pass"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	mockRepo := &mockTokenRepo{
		user: repo.User{
			ID:               "01HGFB9Z5W7ABCDEFGHJKMNPQR",
			Username:         "regular_user",
			PasswordHash:     string(hash),
			IsServiceAccount: false,
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditor := audit.NewAuditLogger(false, "stdio", logger, mockRepo)

	handler := tokenhandler.TokenHandler{
		Logger:          logger,
		Auditor:         auditor,
		Repo:            mockRepo,
		JWTSecret:       []byte("test_secret_12345678901234567890"),
		AccessDuration:  5 * time.Minute,
		RefreshDuration: 24 * time.Hour,
	}

	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte("regular_user:"+password))
	req := httptest.NewRequest(http.MethodPost, "/api/token", nil)
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()

	handler.GetToken(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK for regular user basic auth, got %d", rec.Code)
	}
}
