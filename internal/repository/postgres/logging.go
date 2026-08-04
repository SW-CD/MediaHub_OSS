package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"mediahub_oss/internal/repository"

	"github.com/Masterminds/squirrel"
)

// LogAudit inserts a new audit log into the database.
func (r *PostgresRepository) LogAudit(ctx context.Context, log repository.AuditLog) error {
	detailsJSON, err := json.Marshal(log.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	nowMs := time.Now().UnixMilli()
	query, args, err := r.Builder.Insert("audit_logs").
		Columns("timestamp", "action", "actor", "resource", "details").
		Values(nowMs, log.Action, log.Actor, log.Resource, string(detailsJSON)).
		ToSql()

	if err != nil {
		return err
	}

	_, _ = r.DB.ExecContext(ctx, query, args...)
	return nil
}

// GetLogs retrieves a paginated list of audit logs, optionally filtered by a time range.
func (r *PostgresRepository) GetLogs(ctx context.Context, opts repository.QueryOptions) ([]repository.AuditLog, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	builder := r.Builder.Select("id", "timestamp", "action", "actor", "resource", "details").
		From("audit_logs")

	if !opts.TStart.IsZero() && opts.TStart.After(time.Unix(0, 0)) {
		builder = builder.Where(squirrel.GtOrEq{"timestamp": opts.TStart.UnixMilli()})
	}
	if !opts.TEnd.IsZero() && opts.TEnd.After(time.Unix(0, 0)) {
		builder = builder.Where(squirrel.LtOrEq{"timestamp": opts.TEnd.UnixMilli()})
	}

	if strings.ToLower(opts.Order) == "asc" {
		builder = builder.OrderBy("timestamp ASC")
	} else {
		builder = builder.OrderBy("timestamp DESC")
	}

	if opts.Limit > 0 {
		builder = builder.Limit(uint64(opts.Limit))
	}
	if opts.Offset > 0 {
		builder = builder.Offset(uint64(opts.Offset))
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build get logs query: %w", err)
	}

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []repository.AuditLog
	var logTimestamp int64
	for rows.Next() {
		var log repository.AuditLog
		var detailsStr string

		if err := rows.Scan(&log.ID, &logTimestamp, &log.Action, &log.Actor, &log.Resource, &detailsStr); err != nil {
			return nil, fmt.Errorf("failed to scan audit log row: %w", err)
		}

		if logTimestamp > 0 {
			log.Timestamp = time.UnixMilli(logTimestamp)
		}

		if detailsStr != "" {
			if err := json.Unmarshal([]byte(detailsStr), &log.Details); err != nil {
				log.Details = make(map[string]any)
			}
		} else {
			log.Details = make(map[string]any)
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return logs, nil
}

// DeleteLogs removes all audit logs older than the provided age.
func (r *PostgresRepository) DeleteLogs(ctx context.Context, maxAge time.Duration) error {
	now, err := r.GetDBTime(ctx)
	if err != nil {
		now = time.Now()
	}
	cutoff := now.Add(-maxAge).UnixMilli()

	query, args, err := r.Builder.Delete("audit_logs").
		Where(squirrel.Lt{"timestamp": cutoff}).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build delete logs query: %w", err)
	}

	_, err = r.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete older audit logs: %w", err)
	}

	return nil
}
