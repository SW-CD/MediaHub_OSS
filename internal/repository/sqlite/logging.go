package sqlite

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
func (r *SQLiteRepository) LogAudit(ctx context.Context, log repository.AuditLog) error {
	// Marshal the Details map into a JSON string
	detailsJSON, err := json.Marshal(log.Details)
	if err != nil {
		// Fallback to empty JSON if serialization fails, so we don't lose the log entry
		detailsJSON = []byte("{}")
	}

	// Build the query, relying on SQLite's AUTOINCREMENT and DEFAULT CURRENT_TIMESTAMP.
	query, args, err := r.Builder.Insert("audit_logs").
		Columns("timestamp", "action", "actor", "resource", "details").
		Values(time.Now().UnixMilli(), log.Action, log.Actor, log.Resource, string(detailsJSON)).
		ToSql()

	if err != nil {
		return err
	}

	// 3. Execute the insert
	_, _ = r.DB.ExecContext(ctx, query, args...)
	return nil
}

// GetLogs retrieves a paginated list of audit logs, optionally filtered by a time range.
func (r *SQLiteRepository) GetLogs(ctx context.Context, limit, offset int, order string, tstart, tend time.Time) ([]repository.AuditLog, error) {
	builder := r.Builder.Select("id", "timestamp", "action", "actor", "resource", "details").
		From("audit_logs")

	// Apply time filters (converting unix milliseconds to time.Time for the SQLite timestamp column)
	if !tstart.IsZero() && tstart.After(time.Unix(0, 0)) {
		builder = builder.Where(squirrel.GtOrEq{"timestamp": tstart.UnixMilli()})
	}
	if !tend.IsZero() && tend.After(time.Unix(0, 0)) {
		builder = builder.Where(squirrel.LtOrEq{"timestamp": tend.UnixMilli()})
	}

	// Apply sorting
	if strings.ToLower(order) == "asc" {
		builder = builder.OrderBy("timestamp ASC")
	} else {
		builder = builder.OrderBy("timestamp DESC") // Default to newest first
	}

	// Apply pagination
	if limit > 0 {
		builder = builder.Limit(uint64(limit))
	}
	if offset > 0 {
		builder = builder.Offset(uint64(offset))
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

		// Scan the row
		if err := rows.Scan(&log.ID, &logTimestamp, &log.Action, &log.Actor, &log.Resource, &detailsStr); err != nil {
			return nil, fmt.Errorf("failed to scan audit log row: %w", err)
		}

		if logTimestamp > 0 {
			log.Timestamp = time.UnixMilli(logTimestamp)
		}

		// Unmarshal the JSON string back into the map
		if detailsStr != "" {
			if err := json.Unmarshal([]byte(detailsStr), &log.Details); err != nil {
				// If parsing fails, initialize an empty map to avoid nil panics elsewhere
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
func (r *SQLiteRepository) DeleteLogs(ctx context.Context, maxAge time.Duration) error {

	// For SQLite it is ok to calculate the cutoff timestamp in Go and pass it as a parameter,
	// For Postgres, we would want to use a SQL function like NOW() - INTERVAL '1 month' to ensure the cutoff
	// is calculated on the database side, especially in distributed environments where server times might differ.
	cutoff := time.Now().Add(-maxAge).UnixMilli()

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
