// filepath: internal/services/user_service.go
package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"mediahub/internal/config"
	"mediahub/internal/logging"
	"mediahub/internal/models"
	"mediahub/internal/repository" // Depends on the new repository
)

// --- REFACTOR: Compile-time check to ensure interface is implemented ---
var _ UserService = (*userService)(nil)

// --- REFACTOR: Struct renamed to lowercase ---
// userService handles business logic for user management.
type userService struct {
	Repo *repository.Repository
}

// NewUserService creates a new UserService.
// --- REFACTOR: Return type is the concrete struct, but it satisfies the interface ---
func NewUserService(repo *repository.Repository) *userService {
	return &userService{Repo: repo}
}

// === Pass-through Repository Methods ===

// GetUserByUsername retrieves a user by their username.
func (s *userService) GetUserByUsername(username string) (*models.User, error) {
	return s.Repo.GetUserByUsername(username)
}

// GetUserByID retrieves a user by their ID.
func (s *userService) GetUserByID(id int) (*models.User, error) {
	return s.Repo.GetUserByID(id)
}

// GetUsers retrieves all users.
func (s *userService) GetUsers() ([]models.User, error) {
	return s.Repo.GetUsers()
}

// UpdateUserPassword updates a single user's password (e.g., for /api/me).
func (s *userService) UpdateUserPassword(username, password string) error {
	return s.Repo.UpdateUserPassword(username, password)
}

// === Business Logic Methods ===

// CreateUser handles the logic for creating a new user.
func (s *userService) CreateUser(args repository.UserCreateArgs) (*models.User, error) {
	if args.Username == "" || args.Password == "" {
		return nil, fmt.Errorf("username and password are required")
	}
	// Logic from admin_handler.go is now here
	logging.Log.Debugf("UserService: Attempting to create user '%s'", args.Username)
	createdUser, err := s.Repo.CreateUser(&args)
	if err != nil {
		if errors.Is(err, repository.ErrUserExists) {
			return nil, err // Pass the specific error up
		}
		logging.Log.Errorf("UserService: Failed to create user '%s': %v", args.Username, err)
		return nil, fmt.Errorf("failed to create user")
	}
	return createdUser, nil
}

// UpdateUser handles the logic for updating a user's roles or password.
func (s *userService) UpdateUser(id int, req models.User, newPassword *string) (*models.User, error) {
	logging.Log.Debugf("UserService: Updating user ID %d", id)

	// Get original user for "last admin" check
	originalUser, err := s.Repo.GetUserByID(id)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Logic from admin_handler.go: Prevent removing the last admin's role
	if !req.IsAdmin && originalUser.IsAdmin {
		admins, err := s.Repo.GetAdminUsers()
		if err != nil {
			return nil, fmt.Errorf("failed to check for other admins")
		}
		if len(admins) == 1 && admins[0].ID == originalUser.ID {
			return nil, fmt.Errorf("cannot remove the last admin's admin role")
		}
	}

	// Create the model for the repository update
	userToUpdate := &models.User{
		ID:        int64(id),
		Username:  originalUser.Username, // Needed for cache invalidation
		CanView:   req.CanView,
		CanCreate: req.CanCreate,
		CanEdit:   req.CanEdit,
		CanDelete: req.CanDelete,
		IsAdmin:   req.IsAdmin,
	}

	// If a new password is provided, set it for hashing.
	if newPassword != nil {
		userToUpdate.PasswordHash = *newPassword // Pass plaintext for hashing
	} else {
		userToUpdate.PasswordHash = "" // Signal to skip password update
	}

	if err := s.Repo.UpdateUser(userToUpdate); err != nil {
		return nil, fmt.Errorf("failed to update user")
	}

	// Re-fetch the user to get the final updated state
	return s.Repo.GetUserByID(id)
}

// DeleteUser handles the logic for deleting a user.
func (s *userService) DeleteUser(id int) error {
	logging.Log.Debugf("UserService: Deleting user ID %d", id)

	// Logic from admin_handler.go: Prevent deleting the last admin
	user, err := s.Repo.GetUserByID(id)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if user.IsAdmin {
		admins, err := s.Repo.GetAdminUsers()
		if err != nil {
			return fmt.Errorf("failed to check for other admins")
		}
		if len(admins) == 1 {
			return fmt.Errorf("cannot delete the last admin user")
		}
	}

	if err := s.Repo.DeleteUser(id); err != nil {
		return fmt.Errorf("failed to delete user")
	}
	return nil
}

// InitializeAdminUser ensures the 'admin' user exists on startup and handles password resets.
func (s *userService) InitializeAdminUser(cfg *config.Config) error {
	adminExists, err := s.Repo.UserExists("admin")
	if err != nil {
		return fmt.Errorf("failed to check for admin user: %w", err)
	}

	if !adminExists {
		return s.createAdminUser(cfg.AdminPassword)
	}

	if cfg.ResetAdminPassword {
		return s.resetAdminPassword(cfg.AdminPassword)
	}

	return nil
}

// createAdminUser creates the initial 'admin' user with full permissions.
func (s *userService) createAdminUser(password string) error {
	if password == "" {
		password = generateRandomPassword(10)
		logging.Log.Infof("No admin password provided. Generated a random password for 'admin': %s", password)
	}

	user := &repository.UserCreateArgs{
		Username:  "admin",
		Password:  password,
		CanView:   true,
		CanCreate: true,
		CanEdit:   true,
		CanDelete: true,
		IsAdmin:   true,
	}
	if _, err := s.Repo.CreateUser(user); err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}
	logging.Log.Info("Admin user created successfully.")
	return nil
}

// resetAdminPassword updates the admin's password based on startup flags.
func (s *userService) resetAdminPassword(password string) error {
	if password == "" {
		return fmt.Errorf("cannot reset admin password: --reset_pw is true but no --password or IMS_PASSWORD was provided")
	}
	if err := s.Repo.UpdateUserPassword("admin", password); err != nil {
		return fmt.Errorf("failed to reset admin password: %w", err)
	}
	logging.Log.Info("Admin password has been reset.")
	return nil
}

// generateRandomPassword creates a cryptographically secure random password.
func generateRandomPassword(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		logging.Log.Fatalf("Failed to generate random password: %v", err)
	}
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}
