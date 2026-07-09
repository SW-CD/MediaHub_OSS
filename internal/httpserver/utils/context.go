package utils

import (
	"context"
	"mediahub_oss/internal/repository"
)

// ContextKey is a custom type for context keys to avoid collisions.
type ContextKey string

const (
	UserKey        ContextKey = "user"
	APIKeyKey      ContextKey = "apikey"
	PermissionsKey ContextKey = "permissions"
	IsAdminKey     ContextKey = "isadmin"
)

// GetUserFromContext is a helper to safely retrieve the strongly-typed User object.
func GetUserFromContext(ctx context.Context) *repository.User {
	val := ctx.Value(UserKey)
	if val == nil {
		return nil
	}
	return val.(*repository.User)
}

// GetAPIKeyFromContext is a helper to safely retrieve the strongly-typed APIKey object.
func GetAPIKeyFromContext(ctx context.Context) *repository.APIKey {
	val := ctx.Value(APIKeyKey)
	if val == nil {
		return nil
	}
	return val.(*repository.APIKey)
}

// GetPermissionsFromContext is a helper to safely retrieve the user permissions map from the context.
func GetPermissionsFromContext(ctx context.Context) map[repository.ULID]repository.UserPermissions {
	val := ctx.Value(PermissionsKey)
	if val == nil {
		return nil
	}
	return val.(map[repository.ULID]repository.UserPermissions)
}

// IsAdminFromContext retrieves the effective admin status from the context.
func IsAdminFromContext(ctx context.Context) bool {
	val := ctx.Value(IsAdminKey)
	if val == nil {
		return false
	}
	return val.(bool)
}
