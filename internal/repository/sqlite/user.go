package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	repo "mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared/customerrors"
	"strings"

	"github.com/Masterminds/squirrel"
)

// CreateUser inserts a new user into the database and returns the populated user object (with ID).
func (r *SQLiteRepository) CreateUser(ctx context.Context, user repo.User) (repo.User, error) {
	query, args, err := r.Builder.Insert("users").
		Columns("username", "password_hash", "is_admin").
		Values(user.Username, user.PasswordHash, user.IsAdmin).
		ToSql()
	if err != nil {
		return repo.User{}, fmt.Errorf("failed to build insert user query: %w", err)
	}

	res, err := r.DB.ExecContext(ctx, query, args...)
	if err != nil {
		// SQLite error for duplicate unique constraint (e.g., username already taken)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return repo.User{}, customerrors.ErrUserExists
		}
		return repo.User{}, fmt.Errorf("failed to insert user: %w", err)
	}

	// Retrieve the auto-incremented ID assigned by SQLite
	insertedID, err := res.LastInsertId()
	if err != nil {
		return repo.User{}, fmt.Errorf("failed to retrieve last insert id: %w", err)
	}

	user.ID = insertedID
	return user, nil
}

// CountAdminUsers returns the total number of users who have the global 'is_admin' flag set to true.
func (r *SQLiteRepository) CountAdminUsers(ctx context.Context) (int64, error) {
	query, args, err := r.Builder.Select("COUNT(*)").
		From("users").
		Where(squirrel.Eq{"is_admin": true}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build count admin query: %w", err)
	}

	var count int64
	err = r.DB.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to execute count admin query: %w", err)
	}

	return count, nil
}

// DeleteUser removes a user from the database based on their ID.
// Note: Due to ON DELETE CASCADE, this automatically clears related permissions and tokens.
func (r *SQLiteRepository) DeleteUser(ctx context.Context, id int64) error {
	query, args, err := r.Builder.Delete("users").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete user query: %w", err)
	}

	res, err := r.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return customerrors.ErrNotFound
	}

	return nil
}

// UpdateUser modifies the global properties of an existing user account.
func (r *SQLiteRepository) UpdateUser(ctx context.Context, user repo.User) (repo.User, error) {
	query, args, err := r.Builder.Update("users").
		Set("username", user.Username).
		Set("password_hash", user.PasswordHash).
		Set("is_admin", user.IsAdmin).
		Where(squirrel.Eq{"id": user.ID}).
		ToSql()
	if err != nil {
		return repo.User{}, fmt.Errorf("failed to build update user query: %w", err)
	}

	res, err := r.DB.ExecContext(ctx, query, args...)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return repo.User{}, customerrors.ErrUserExists
		}
		return repo.User{}, fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return repo.User{}, fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return repo.User{}, customerrors.ErrNotFound
	}

	return user, nil
}

// GetUsers retrieves a list of all user accounts from the database.
func (r *SQLiteRepository) GetUsers(ctx context.Context) ([]repo.User, error) {
	query, args, err := r.Builder.Select("id", "username", "password_hash", "is_admin").
		From("users").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build get users query: %w", err)
	}

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []repo.User
	for rows.Next() {
		var user repo.User
		if err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin); err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return users, nil
}

// GetUserByID retrieves a single user record by its unique ID.
func (r *SQLiteRepository) GetUserByID(ctx context.Context, id int64) (repo.User, error) {
	query, args, err := r.Builder.Select("id", "username", "password_hash", "is_admin").
		From("users").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return repo.User{}, fmt.Errorf("failed to build get user by id query: %w", err)
	}

	var user repo.User
	err = r.DB.QueryRowContext(ctx, query, args...).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repo.User{}, customerrors.ErrNotFound
		}
		return repo.User{}, fmt.Errorf("failed to scan user by id: %w", err)
	}

	return user, nil
}

// GetUserByUsername retrieves a single user record by their unique username.
func (r *SQLiteRepository) GetUserByUsername(ctx context.Context, username string) (repo.User, error) {
	query, args, err := r.Builder.Select("id", "username", "password_hash", "is_admin").
		From("users").
		Where(squirrel.Eq{"username": username}).
		ToSql()
	if err != nil {
		return repo.User{}, fmt.Errorf("failed to build get user by username query: %w", err)
	}

	var user repo.User
	err = r.DB.QueryRowContext(ctx, query, args...).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repo.User{}, customerrors.ErrNotFound
		}
		return repo.User{}, fmt.Errorf("failed to scan user by username: %w", err)
	}

	return user, nil
}

// SetUserPermissions creates, updates, or deletes database-specific permissions for a user.
func (r *SQLiteRepository) SetUserPermissions(ctx context.Context, permissions repo.UserPermissions) error {
	// If Roles is empty, the intention is to delete the permission entry.
	if permissions.Roles == "" {
		query, args, err := r.Builder.Delete("database_permissions").
			Where(squirrel.Eq{"user_id": permissions.UserID, "database_id": permissions.DatabaseID}).
			ToSql()
		if err != nil {
			return fmt.Errorf("failed to build delete permissions query: %w", err)
		}

		_, err = r.DB.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to delete permissions: %w", err)
		}
		return nil
	}

	// Otherwise, we perform an Upsert (Insert or Replace)
	canView, canCreate, canEdit, canDelete := parseRolesString(permissions.Roles)

	query, args, err := r.Builder.Insert("database_permissions").
		Columns("user_id", "database_id", "can_view", "can_create", "can_edit", "can_delete").
		Values(permissions.UserID, permissions.DatabaseID, canView, canCreate, canEdit, canDelete).
		Suffix("ON CONFLICT (user_id, database_id) DO UPDATE SET can_view = excluded.can_view, can_create = excluded.can_create, can_edit = excluded.can_edit, can_delete = excluded.can_delete").
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build upsert permissions query: %w", err)
	}

	_, err = r.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to upsert user permissions: %w", err)
	}

	return nil
}

// GetUserPermissions retrieves the exact rights a user has for a specific database.
func (r *SQLiteRepository) GetUserPermissions(ctx context.Context, userID int64, dbID string) (repo.UserPermissions, error) {
	query, args, err := r.Builder.Select("can_view", "can_create", "can_edit", "can_delete").
		From("database_permissions").
		Where(squirrel.Eq{"user_id": userID, "database_id": dbID}).
		ToSql()
	if err != nil {
		return repo.UserPermissions{}, fmt.Errorf("failed to build get permissions query: %w", err)
	}

	var canView, canCreate, canEdit, canDelete bool
	err = r.DB.QueryRowContext(ctx, query, args...).Scan(&canView, &canCreate, &canEdit, &canDelete)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repo.UserPermissions{}, customerrors.ErrNotFound
		}
		return repo.UserPermissions{}, fmt.Errorf("failed to scan user permissions: %w", err)
	}

	return repo.UserPermissions{
		UserID:     userID,
		DatabaseID: dbID,
		Roles:      buildRolesString(canView, canCreate, canEdit, canDelete),
	}, nil
}

// GetAllUserPermissions retrieves every specific database right assigned to a given user.
func (r *SQLiteRepository) GetAllUserPermissions(ctx context.Context, userID int64) ([]repo.UserPermissions, error) {
	query, args, err := r.Builder.Select("database_id", "can_view", "can_create", "can_edit", "can_delete").
		From("database_permissions").
		Where(squirrel.Eq{"user_id": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build get all permissions query: %w", err)
	}

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query all user permissions: %w", err)
	}
	defer rows.Close()

	var permissions []repo.UserPermissions
	for rows.Next() {
		var dbID string
		var canView, canCreate, canEdit, canDelete bool

		if err := rows.Scan(&dbID, &canView, &canCreate, &canEdit, &canDelete); err != nil {
			return nil, fmt.Errorf("failed to scan permissions row: %w", err)
		}

		permissions = append(permissions, repo.UserPermissions{
			UserID:     userID,
			DatabaseID: dbID,
			Roles:      buildRolesString(canView, canCreate, canEdit, canDelete),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return permissions, nil
}
