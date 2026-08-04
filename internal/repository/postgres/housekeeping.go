package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"

	repo "mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared/customerrors"
)

// HouseKeepingRequired returns a list of databases where a housekeeping run is required.
func (r *PostgresRepository) HouseKeepingRequired(ctx context.Context) ([]repo.Database, error) {
	query, args, err := r.Builder.Select(
		"id", "name", "content_type", "hk_interval", "hk_disk_space", "hk_max_age",
		"create_preview", "auto_conversion", "n_max_queued", "hk_last_run",
		"entry_count", "total_disk_space_bytes").
		From("databases").
		Where("hk_interval > 0 AND hk_last_run + hk_interval <= CAST(EXTRACT(EPOCH FROM clock_timestamp()) * 1000 AS BIGINT)").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build housekeeping required query: %w", err)
	}

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query databases for housekeeping: %w", err)
	}
	defer rows.Close()

	var databases []repo.Database
	for rows.Next() {
		db, err := scanDatabaseRow(rows)
		if err != nil {
			return nil, err
		}
		databases = append(databases, db)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return databases, nil
}

// HouseKeepingWasCalled sets the LastHkRun to now (server timestamp).
func (r *PostgresRepository) HouseKeepingWasCalled(ctx context.Context, dbID repo.ULID) (time.Time, error) {
	now, err := r.GetDBTime(ctx)
	if err != nil {
		now = time.Now()
	}

	query, args, err := r.Builder.Update("databases").
		Set("hk_last_run", now.UnixMilli()).
		Where(squirrel.Eq{"id": dbID.String()}).
		ToSql()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to build housekeeping update query: %w", err)
	}

	res, err := r.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to update last housekeeping run: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to retrieve rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return time.Time{}, customerrors.ErrNotFound
	}

	return now, nil
}
