package userhandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"mediahub_oss/internal/httpserver/userhandler"
	"mediahub_oss/internal/httpserver/utils"
	"mediahub_oss/internal/logging/audit"
	repo "mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared/customerrors"
)

type mockUserRepo struct {
	repo.Repository
	users map[repo.ULID]repo.User
}

func (m *mockUserRepo) GetUserByID(ctx context.Context, id repo.ULID) (repo.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return repo.User{}, customerrors.ErrNotFound
}

func (m *mockUserRepo) CreateUser(ctx context.Context, u repo.User) (repo.User, error) {
	u.ID = "01HGFB9Z5W7ABCDEFGHJKMNPQR"
	if m.users == nil {
		m.users = make(map[repo.ULID]repo.User)
	}
	m.users[u.ID] = u
	return u, nil
}

func (m *mockUserRepo) UpdateUser(ctx context.Context, u repo.User) (repo.User, error) {
	if _, ok := m.users[u.ID]; !ok {
		return repo.User{}, customerrors.ErrNotFound
	}
	m.users[u.ID] = u
	return u, nil
}

func (m *mockUserRepo) SetUserPermissions(ctx context.Context, p repo.UserPermissions) error {
	return nil
}

func (m *mockUserRepo) GetAllUserPermissions(ctx context.Context, userID repo.ULID) ([]repo.UserPermissions, error) {
	return nil, nil
}

func TestCreateUser_DuplicateDatabasePermissionsRejected(t *testing.T) {
	mockRepo := &mockUserRepo{
		users: make(map[repo.ULID]repo.User),
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditor := audit.NewAuditLogger(false, "stdio", logger, mockRepo)
	handler := userhandler.UserHandler{
		Logger:  logger,
		Auditor: auditor,
		Repo:    mockRepo,
	}

	payload := userhandler.CreateUserPayload{
		Username: "newuser",
		Password: "password123",
		Permissions: []userhandler.DatabasePermission{
			{DatabaseID: "01HGFB9Z5W7ABCDEFGHJKMNPQR", CanView: true},
			{DatabaseID: "01HGFB9Z5W7ABCDEFGHJKMNPQR", CanEdit: true}, // duplicate
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/user", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), utils.UserKey, &repo.User{Username: "admin", IsAdmin: true}))
	rec := httptest.NewRecorder()

	handler.CreateUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request for duplicate DB permissions, got %d", rec.Code)
	}
}

func TestUpdateUser_DuplicateDatabasePermissionsRejected(t *testing.T) {
	targetULID := repo.ULID("01HGFB9Z5W7ABCDEFGHJKMNPQR")
	mockRepo := &mockUserRepo{
		users: map[repo.ULID]repo.User{
			targetULID: {
				ID:           targetULID,
				Username:     "existinguser",
				PasswordHash: "$2a$10$abcdef",
			},
		},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditor := audit.NewAuditLogger(false, "stdio", logger, mockRepo)
	handler := userhandler.UserHandler{
		Logger:  logger,
		Auditor: auditor,
		Repo:    mockRepo,
	}

	payload := userhandler.UpdateUserPayload{
		Permissions: []userhandler.DatabasePermission{
			{DatabaseID: "01HGFB9Z5W7ABCDEFGHJKMNPQR", CanView: true},
			{DatabaseID: "01HGFB9Z5W7ABCDEFGHJKMNPQR", CanEdit: true}, // duplicate
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/api/user/"+string(targetULID), bytes.NewReader(body))
	req.SetPathValue("user_ulid", string(targetULID))
	req = req.WithContext(context.WithValue(req.Context(), utils.UserKey, &repo.User{Username: "admin", IsAdmin: true}))
	rec := httptest.NewRecorder()

	handler.UpdateUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request for duplicate DB permissions in UpdateUser, got %d", rec.Code)
	}
}
