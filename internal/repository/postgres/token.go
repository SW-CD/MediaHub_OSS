package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	repo "mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared/customerrors"
	"time"

	"github.com/Masterminds/squirrel"
)

// StoreRefreshToken inserts a new hashed refresh token into the database along with its expiry time.
func (r *PostgresRepository) StoreRefreshToken(ctx context.Context, userID repo.ULID, tokenHash string, validDuration time.Duration) error {
	query, args, err := r.Builder.Insert("refresh_tokens").
		Columns("user_id", "token_hash", "expiry").
		Values(userID.String(), tokenHash, squirrel.Expr("(EXTRACT(EPOCH FROM clock_timestamp()) * 1000 + ?)::BIGINT", validDuration.Milliseconds())).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert token query: %w", err)
	}

	_, err = r.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	return nil
}

// ValidateRefreshToken checks if a refresh token hash exists and is not expired.
func (r *PostgresRepository) ValidateRefreshToken(ctx context.Context, tokenHash string) (repo.ULID, error) {
	query, args, err := r.Builder.Select("user_id").
		From("refresh_tokens").
		Where(squirrel.Expr("token_hash = ? AND expiry >= (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)", tokenHash)).
		ToSql()
	if err != nil {
		return "", fmt.Errorf("failed to build validate token query: %w", err)
	}

	var userIDStr string
	err = r.DB.QueryRowContext(ctx, query, args...).Scan(&userIDStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", customerrors.ErrNotFound
		}
		return "", fmt.Errorf("failed to query refresh token: %w", err)
	}

	return repo.ULID(userIDStr), nil
}

// DeleteRefreshToken removes a specific refresh token from the database using its hash.
func (r *PostgresRepository) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	query, args, err := r.Builder.Delete("refresh_tokens").
		Where(squirrel.Eq{"token_hash": tokenHash}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete token query: %w", err)
	}

	res, err := r.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return customerrors.ErrNotFound
	}

	return nil
}

// DeleteExpiredRefreshTokens removes all tokens that have passed their expiration date.
func (r *PostgresRepository) DeleteExpiredRefreshTokens(ctx context.Context) (int64, error) {
	query, args, err := r.Builder.Delete("refresh_tokens").
		Where(squirrel.Expr("expiry < (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)")).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build delete expired tokens query: %w", err)
	}

	res, err := r.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired refresh tokens: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve rows affected: %w", err)
	}

	return rowsAffected, nil
}

// DeleteAllRefreshTokensForUser removes all active sessions for a specific user.
func (r *PostgresRepository) DeleteAllRefreshTokensForUser(ctx context.Context, userID repo.ULID) error {
	query, args, err := r.Builder.Delete("refresh_tokens").
		Where(squirrel.Eq{"user_id": userID.String()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete all tokens query: %w", err)
	}

	_, err = r.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete all refresh tokens for user: %w", err)
	}

	return nil
}
