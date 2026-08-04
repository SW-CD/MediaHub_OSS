package postgres

import (
	"context"
	"database/sql"
)

// Queryer is an interface satisfied by both *sql.DB and *sql.Tx
type Queryer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
