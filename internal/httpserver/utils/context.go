package utils

import (
	"context"
	"mediahub_oss/internal/repository"
)

// ContextKey is a custom type for context keys to avoid collisions.
type ContextKey string

const (
	UserKey ContextKey = "user"
)

// GetUserFromContext is a helper to safely retrieve the strongly-typed User object.
func GetUserFromContext(ctx context.Context) *repository.User {
	user := ctx.Value(UserKey).(*repository.User)
	return user
}
