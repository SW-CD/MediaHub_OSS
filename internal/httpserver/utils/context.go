package utils

import (
	"context"
	"mediahub_oss/internal/repository"
)

// ContextKey is a custom type for context keys to avoid collisions.
type ContextKey string

const (
	UserKey             ContextKey = "user"
	APIKeyKey           ContextKey = "apikey"
	PermissionHolderKey ContextKey = "permholder"
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
func GetPermissionHolderFromContext(ctx context.Context) PermissionHolder {
	val := ctx.Value(PermissionHolderKey)
	if val == nil {
		return nil
	}
	return val.(PermissionHolder)
}
