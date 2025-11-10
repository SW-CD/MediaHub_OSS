// filepath: internal/repository/user_repo.go
package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"mediahub/internal/logging"
	"mediahub/internal/models"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ErrUserExists is returned when trying to create a user that already exists.
var ErrUserExists = errors.New("user already exists")

// UserCreateArgs is a struct used for creating users in the database layer.
// It is separate from the models.User to include the plaintext password for creation.
type UserCreateArgs struct {
	Username  string
	Password  string
	CanView   bool
	CanCreate bool
	CanEdit   bool
	CanDelete bool
	IsAdmin   bool
}

// GetUserByUsername retrieves a user by their username, using a cache for performance.
func (s *Repository) GetUserByUsername(username string) (*models.User, error) {
	cacheKey := fmt.Sprintf("user_by_name_%s", username)
	if user, found := s.Cache.Get(cacheKey); found {
		// logging.Log.Debugf("GetUserByUsername: CACHE HIT for '%s'", username)
		return user.(*models.User), nil
	}

	logging.Log.Debugf("GetUserByUsername: CACHE MISS for '%s'. Querying DB.", username)
	query := "SELECT id, username, password_hash, can_view, can_create, can_edit, can_delete, is_admin FROM users WHERE username = ?"
	row := s.DB.QueryRow(query, username)

	var user models.User
	if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CanView, &user.CanCreate, &user.CanEdit, &user.CanDelete, &user.IsAdmin); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	logging.Log.Debugf("GetUserByUsername: Setting cache for '%s' (ID: %d)", user.Username, user.ID)
	s.Cache.Set(cacheKey, &user, 5*time.Minute)
	s.Cache.Set(fmt.Sprintf("user_by_id_%d", user.ID), &user, 5*time.Minute)

	return &user, nil
}

// UserExists checks if a user with the given username exists.
func (s *Repository) UserExists(username string) (bool, error) {
	_, err := s.GetUserByUsername(username)
	if err != nil {
		if err.Error() == "user not found" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateUser creates a new user in the database.
func (s *Repository) CreateUser(user *UserCreateArgs) (*models.User, error) {
	logging.Log.Debugf("CreateUser: Hashing password for '%s'", user.Username)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO users (username, password_hash, can_view, can_create, can_edit, can_delete, is_admin)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := s.DB.Exec(query, user.Username, string(hashedPassword), user.CanView, user.CanCreate, user.CanEdit, user.CanDelete, user.IsAdmin)
	if err != nil {
		// Check for UNIQUE constraint violation
		if err.Error() == "UNIQUE constraint failed: users.username" {
			return nil, ErrUserExists
		}
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	logging.Log.Debugf("CreateUser: User '%s' created with ID %d", user.Username, id)

	// Return the model, not the internal DB struct
	return &models.User{
		ID:           id,
		Username:     user.Username,
		PasswordHash: string(hashedPassword),
		CanView:      user.CanView,
		CanCreate:    user.CanCreate,
		CanEdit:      user.CanEdit,
		CanDelete:    user.CanDelete,
		IsAdmin:      user.IsAdmin,
	}, nil
}

// UpdateUser updates a user's roles and optionally their password.
func (s *Repository) UpdateUser(user *models.User) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update roles
	query := `
		UPDATE users
		SET can_view = ?, can_create = ?, can_edit = ?, can_delete = ?, is_admin = ?
		WHERE id = ?
	`
	logging.Log.Debugf("UpdateUser: Updating roles for user '%s'", user.Username)
	if _, err := tx.Exec(query, user.CanView, user.CanCreate, user.CanEdit, user.CanDelete, user.IsAdmin, user.ID); err != nil {
		return err
	}

	// Update password only if a new one is provided (i.e., not blank).
	// The handler sets the PasswordHash field to the new plaintext password.
	if user.PasswordHash != "" {
		logging.Log.Debugf("UpdateUser: New password provided for user '%s'. Re-hashing.", user.Username)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.PasswordHash), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		query = "UPDATE users SET password_hash = ? WHERE id = ?"
		if _, err := tx.Exec(query, string(hashedPassword), user.ID); err != nil {
			return err
		}
	} else {
		logging.Log.Debugf("UpdateUser: No new password for '%s'. Skipping password update.", user.Username)
	}

	// Invalidate cache entries for the updated user
	logging.Log.Debugf("UpdateUser: Invalidating cache for user '%s' (ID: %d)", user.Username, user.ID)
	s.Cache.Delete(fmt.Sprintf("user_by_name_%s", user.Username))
	s.Cache.Delete(fmt.Sprintf("user_by_id_%d", user.ID))
	logging.Log.Debugf("UpdateUser: Cache invalidated for user '%s' (ID: %d)", user.Username, user.ID)

	return tx.Commit()
}

// UpdateUserPassword updates a single user's password.
func (s *Repository) UpdateUserPassword(username, password string) error {
	user, err := s.GetUserByUsername(username)
	if err != nil {
		return err
	}

	logging.Log.Debugf("UpdateUserPassword: Hashing new password for user '%s'", username)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	query := "UPDATE users SET password_hash = ? WHERE id = ?"
	if _, err := s.DB.Exec(query, string(hashedPassword), user.ID); err != nil {
		return err
	}

	// Invalidate cache entries for the updated user
	logging.Log.Debugf("UpdateUserPassword: Invalidating cache for user '%s' (ID: %d)", username, user.ID)
	s.Cache.Delete(fmt.Sprintf("user_by_name_%s", user.Username))
	s.Cache.Delete(fmt.Sprintf("user_by_id_%d", user.ID))

	return nil
}

// GetUsers retrieves all users from the database.
func (s *Repository) GetUsers() ([]models.User, error) {
	query := "SELECT id, username, password_hash, can_view, can_create, can_edit, can_delete, is_admin FROM users"
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CanView, &user.CanCreate, &user.CanEdit, &user.CanDelete, &user.IsAdmin); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// GetUserByID retrieves a user by their ID, using a cache for performance.
func (s *Repository) GetUserByID(id int) (*models.User, error) {
	cacheKey := fmt.Sprintf("user_by_id_%d", id)
	if user, found := s.Cache.Get(cacheKey); found {
		logging.Log.Debugf("GetUserByID: CACHE HIT for ID %d", id)
		return user.(*models.User), nil
	}

	logging.Log.Debugf("GetUserByID: CACHE MISS for ID %d. Querying DB.", id)
	query := "SELECT id, username, password_hash, can_view, can_create, can_edit, can_delete, is_admin FROM users WHERE id = ?"
	row := s.DB.QueryRow(query, id)

	var user models.User
	if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CanView, &user.CanCreate, &user.CanEdit, &user.CanDelete, &user.IsAdmin); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	logging.Log.Debugf("GetUserByID: Setting cache for ID %d (User: '%s')", id, user.Username)
	s.Cache.Set(cacheKey, &user, 5*time.Minute)
	s.Cache.Set(fmt.Sprintf("user_by_name_%s", user.Username), &user, 5*time.Minute)

	return &user, nil
}

// GetAdminUsers retrieves all users with the IsAdmin role.
func (s *Repository) GetAdminUsers() ([]models.User, error) {
	query := "SELECT id, username, password_hash, can_view, can_create, can_edit, can_delete, is_admin FROM users WHERE is_admin = 1"
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CanView, &user.CanCreate, &user.CanEdit, &user.CanDelete, &user.IsAdmin); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// DeleteUser deletes a user by their ID.
func (s *Repository) DeleteUser(id int) error {
	user, err := s.GetUserByID(id)
	if err != nil {
		return err
	}

	query := "DELETE FROM users WHERE id = ?"
	if _, err := s.DB.Exec(query, id); err != nil {
		return err
	}

	// Invalidate cache entries for the deleted user
	logging.Log.Debugf("DeleteUser: Invalidating cache for user '%s' (ID: %d)", user.Username, user.ID)
	s.Cache.Delete(fmt.Sprintf("user_by_name_%s", user.Username))
	s.Cache.Delete(fmt.Sprintf("user_by_id_%d", id))

	return nil
}
