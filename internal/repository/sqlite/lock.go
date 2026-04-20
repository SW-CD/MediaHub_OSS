package sqlite

import (
	"context"
	"time"
)

// For SQLite, a database lock is not required, as a single client will access it
func (r *SQLiteRepository) AcquireLock(ctx context.Context, lockName string, ownerID string, ttl time.Duration) (bool, error) {
	return true, nil
}

func (r *SQLiteRepository) ReleaseLock(ctx context.Context, lockName string, ownerID string) error {
	return nil
}
