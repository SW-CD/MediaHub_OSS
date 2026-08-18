package repository_test

import (
	"context"
	"errors"
	"testing"

	repo "mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared/customerrors"
)

// mockUserRepo implements only the repository methods needed for UserExists tests.
type mockUserRepo struct {
	repo.Repository
	getUserByUsernameFunc func(ctx context.Context, username string) (repo.User, error)
}

func (m *mockUserRepo) GetUserByUsername(ctx context.Context, username string) (repo.User, error) {
	if m.getUserByUsernameFunc != nil {
		return m.getUserByUsernameFunc(ctx, username)
	}
	return repo.User{}, customerrors.ErrNotFound
}

func TestUserExists(t *testing.T) {
	ctx := context.Background()

	t.Run("user exists", func(t *testing.T) {
		m := &mockUserRepo{
			getUserByUsernameFunc: func(ctx context.Context, username string) (repo.User, error) {
				return repo.User{Username: username}, nil
			},
		}
		exists, err := repo.UserExists(ctx, m, "alice")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Fatalf("expected exists to be true, got false")
		}
	})

	t.Run("user does not exist - ErrNotFound", func(t *testing.T) {
		m := &mockUserRepo{
			getUserByUsernameFunc: func(ctx context.Context, username string) (repo.User, error) {
				return repo.User{}, customerrors.ErrNotFound
			},
		}
		exists, err := repo.UserExists(ctx, m, "bob")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists {
			t.Fatalf("expected exists to be false, got true")
		}
	})

	t.Run("unexpected repository error", func(t *testing.T) {
		expectedErr := errors.New("db connection timeout")
		m := &mockUserRepo{
			getUserByUsernameFunc: func(ctx context.Context, username string) (repo.User, error) {
				return repo.User{}, expectedErr
			},
		}
		exists, err := repo.UserExists(ctx, m, "dave")
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected error %v, got %v", expectedErr, err)
		}
		if exists {
			t.Fatalf("expected exists to be false on error")
		}
	})
}
