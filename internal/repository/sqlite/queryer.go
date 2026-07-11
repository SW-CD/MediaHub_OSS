package sqlite

import (
	"context"
	"database/sql"
)

// Queryer defines the set of database operations that can be performed
// either directly on a database pool (*sql.DB) or within an active transaction (*sql.Tx).
//
// WHY THIS EXISTS:
// In SQLite configurations (especially with MaxOpenConns = 1 to serialize writes),
// performing nested queries on separate database connections will starve the connection
// pool and cause a permanent deadlock.
//
// By passing a Queryer to repository helper functions (e.g. getCustomFields), they can
// execute queries on the same connection as the active transaction (when passed a *sql.Tx)
// or use the general pool (when passed a *sql.DB). This prevents connection starvation
// deadlocks and ensures transactional consistency (helpers see uncommitted transaction modifications).
//
// HOW TO USE:
// 1. Define query helper functions to accept Queryer:
//    func (r *SQLiteRepository) getCustomFields(ctx context.Context, q Queryer, dbID string)
//
// 2. Call the helper passing r.DB when outside a transaction:
//    r.getCustomFields(ctx, r.DB, dbID)
//
// 3. Call the helper passing tx when inside a transaction:
//    r.getCustomFields(ctx, tx, dbID)
type Queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

var _ Queryer = (*sql.DB)(nil)
var _ Queryer = (*sql.Tx)(nil)
