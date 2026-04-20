package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"

	"mediahub_oss/internal/media"
	repo "mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared/customerrors"
)

// CreateEntry inserts a new entry into the database's specific table and updates global stats.
func (r *SQLiteRepository) CreateEntry(ctx context.Context, db repo.Database, entry repo.Entry) (repo.Entry, error) {
	// Verify mime type matching DB's content type
	isValidMime, err := media.IsMimeOfType(db.ContentType, entry.MimeType)
	if !isValidMime {
		return repo.Entry{}, customerrors.ErrBadMimeType
	}
	if err != nil {
		return repo.Entry{}, err
	}

	// Establish timing (SQLite case, with single client, we take client time)
	now := time.Now()
	var entryTime time.Time
	if !entry.Timestamp.IsZero() {
		entryTime = entry.Timestamp
	}

	// Map standard columns
	// Squirrel's SetMap is perfect for our highly dynamic schema
	insertData := map[string]any{
		"timestamp":        entryTime.UnixMilli(),
		"created_at":       now.UnixMilli(),
		"updated_at":       now.UnixMilli(),
		"filesize":         entry.Size,
		"preview_filesize": entry.PreviewSize,
		"filename":         entry.FileName,
		"status":           entry.Status,
		"mime_type":        entry.MimeType,
	}

	// Dynamically append Media and Custom fields
	for key, value := range entry.MediaFields {
		insertData[key] = value
	}
	for key, value := range entry.CustomFields {
		insertData[customFieldsPrefix+key] = value
	}

	// Begin Transaction
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return repo.Entry{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert the Entry
	tableName := fmt.Sprintf(`"entries_%s"`, db.Name)
	insertQuery, args, err := r.Builder.Insert(tableName).SetMap(insertData).ToSql()
	if err != nil {
		return repo.Entry{}, fmt.Errorf("failed to build insert query: %w", err)
	}

	res, err := tx.ExecContext(ctx, insertQuery, args...)
	if err != nil {
		return repo.Entry{}, fmt.Errorf("failed to insert entry: %w", err)
	}

	insertedID, err := res.LastInsertId()
	if err != nil {
		return repo.Entry{}, fmt.Errorf("failed to retrieve insert ID: %w", err)
	}
	entry.ID = insertedID

	// Atomically update parent Database stats
	// Calculate total size delta (main file + preview)
	totalSizeDelta := entry.Size + entry.PreviewSize

	statsQuery, statsArgs, err := r.Builder.Update("databases").
		Set("entry_count", squirrel.Expr("entry_count + 1")).
		Set("total_disk_space_bytes", squirrel.Expr("total_disk_space_bytes + ?", totalSizeDelta)).
		Where(squirrel.Eq{"name": db.Name}).
		ToSql()
	if err != nil {
		return repo.Entry{}, fmt.Errorf("failed to build stats update query: %w", err)
	}

	if _, err := tx.ExecContext(ctx, statsQuery, statsArgs...); err != nil {
		return repo.Entry{}, fmt.Errorf("failed to update database stats: %w", err)
	}

	// 7. Commit
	if err := tx.Commit(); err != nil {
		return repo.Entry{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return entry, nil
}

// GetEntry retrieves a single entry by its ID using a dynamic row scanner.
func (r *SQLiteRepository) GetEntry(ctx context.Context, dbname string, id int64) (repo.Entry, error) {
	tableName := fmt.Sprintf(`"entries_%s"`, dbname)
	query, args, err := r.Builder.Select("*").From(tableName).Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		return repo.Entry{}, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return repo.Entry{}, fmt.Errorf("failed to query entry: %w", err)
	}
	defer rows.Close()

	entry, err := r.scanEntryRow(rows)
	if err != nil {
		return repo.Entry{}, fmt.Errorf("failed to scan entry: %w", err)
	}

	return entry, nil
}

// GetEntries retrieves a paginated list of entries, optionally filtered by a time range.
func (r *SQLiteRepository) GetEntries(ctx context.Context, dbname string, limit, offset int, order string, tstart, tend time.Time) ([]repo.Entry, error) {
	tableName := fmt.Sprintf(`"entries_%s"`, dbname)
	builder := r.Builder.Select("*").From(tableName)

	// Apply time filters only if they differ from the absolute minimum/maximum
	if !tstart.IsZero() && tstart.After(time.Unix(0, 0)) {
		builder = builder.Where(squirrel.GtOrEq{"timestamp": tstart.UnixMilli()})
	}
	if !tend.IsZero() && tend.After(time.Unix(0, 0)) {
		builder = builder.Where(squirrel.LtOrEq{"timestamp": tend.UnixMilli()})
	}

	if strings.ToLower(order) == "asc" {
		builder = builder.OrderBy("timestamp ASC")
	} else {
		builder = builder.OrderBy("timestamp DESC") // Default to newest first
	}

	if limit > 0 {
		builder = builder.Limit(uint64(limit))
	}
	if offset > 0 {
		builder = builder.Offset(uint64(offset))
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}
	defer rows.Close()

	entries, err := r.scanEntryRows(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to scan entry: %w", err)
	}

	return entries, nil
}

// UpdateEntry modifies an existing entry's metadata and safely adjusts the parent database's size statistics.
func (r *SQLiteRepository) UpdateEntry(ctx context.Context, dbname string, entry repo.Entry) (repo.Entry, error) {
	tableName := fmt.Sprintf(`"entries_%s"`, dbname)

	var entryTime time.Time
	if !entry.Timestamp.IsZero() {
		entryTime = entry.Timestamp
	}

	// 1. Begin SQL Transaction
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return repo.Entry{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 2. Query the current size of the entry before updating
	var oldSize, oldPreviewSize uint64
	queryOld, argsOld, err := r.Builder.Select("filesize", "preview_filesize").
		From(tableName).
		Where(squirrel.Eq{"id": entry.ID}).
		ToSql()
	if err != nil {
		return repo.Entry{}, fmt.Errorf("failed to build select old sizes query: %w", err)
	}

	err = tx.QueryRowContext(ctx, queryOld, argsOld...).Scan(&oldSize, &oldPreviewSize)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repo.Entry{}, customerrors.ErrNotFound
		}
		return repo.Entry{}, fmt.Errorf("failed to query old sizes: %w", err)
	}

	// 3. Update the entry row with new data
	now := time.Now().UnixMilli()
	updateData := map[string]any{
		"timestamp":        entryTime.UnixMilli(),
		"updated_at":       now,
		"filesize":         entry.Size,
		"preview_filesize": entry.PreviewSize,
		"filename":         entry.FileName,
		"status":           entry.Status,
		"mime_type":        entry.MimeType,
	}

	for key, value := range entry.MediaFields {
		updateData[key] = value
	}
	for key, value := range entry.CustomFields {
		updateData[customFieldsPrefix+key] = value
	}

	updateQuery, argsUpdate, err := r.Builder.Update(tableName).
		SetMap(updateData).
		Where(squirrel.Eq{"id": entry.ID}).
		ToSql()
	if err != nil {
		return repo.Entry{}, fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err = tx.ExecContext(ctx, updateQuery, argsUpdate...); err != nil {
		return repo.Entry{}, fmt.Errorf("failed to update entry: %w", err)
	}

	// 4. Calculate the delta and atomically apply it to the main database stats
	delta := (int64(entry.Size) + int64(entry.PreviewSize)) - (int64(oldSize) + int64(oldPreviewSize))

	if delta != 0 {
		statsQuery, statsArgs, err := r.Builder.Update("databases").
			Set("total_disk_space_bytes", squirrel.Expr("total_disk_space_bytes + ?", delta)).
			Where(squirrel.Eq{"name": dbname}).
			ToSql()
		if err != nil {
			return repo.Entry{}, fmt.Errorf("failed to build stats update query: %w", err)
		}

		if _, err := tx.ExecContext(ctx, statsQuery, statsArgs...); err != nil {
			return repo.Entry{}, fmt.Errorf("failed to update database stats: %w", err)
		}
	}

	// 5. Commit Transaction
	if err := tx.Commit(); err != nil {
		return repo.Entry{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return entry, nil
}

// UpdateEntriesStatus efficiently modifies the async processing status of multiple entries at once.
func (r *SQLiteRepository) UpdateEntriesStatus(ctx context.Context, dbname string, entryIDs []int64, status uint8) error {
	if len(entryIDs) == 0 {
		return nil
	}

	tableName := fmt.Sprintf(`"entries_%s"`, dbname)
	now := time.Now().UnixMilli()

	// squirrel.Eq with a slice automatically translates to an 'IN (?, ?, ...)' SQL clause
	query, args, err := r.Builder.Update(tableName).
		Set("status", status).
		Set("updated_at", now).
		Where(squirrel.Eq{"id": entryIDs}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update status query: %w", err)
	}

	res, err := r.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update entries status: %w", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return customerrors.ErrNotFound
	}

	return nil
}

// DeleteEntry removes a single entry and atomically decrements the parent database's statistics.
func (r *SQLiteRepository) DeleteEntry(ctx context.Context, dbname string, id int64) (repo.DeletedEntryMeta, error) {
	tableName := fmt.Sprintf(`"entries_%s"`, dbname)

	// 1. Begin SQL Transaction
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return repo.DeletedEntryMeta{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 2. Delete the row and retrieve its sizes using RETURNING
	deleteQuery, deleteArgs, err := r.Builder.Delete(tableName).
		Where(squirrel.Eq{"id": id}).
		Suffix("RETURNING id, filesize, preview_filesize").
		ToSql()
	if err != nil {
		return repo.DeletedEntryMeta{}, fmt.Errorf("failed to build delete query: %w", err)
	}

	var meta repo.DeletedEntryMeta
	err = tx.QueryRowContext(ctx, deleteQuery, deleteArgs...).Scan(&meta.ID, &meta.Filesize, &meta.PreviewSize)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repo.DeletedEntryMeta{}, customerrors.ErrNotFound
		}
		return repo.DeletedEntryMeta{}, fmt.Errorf("failed to execute delete and retrieve sizes: %w", err)
	}

	// 3. Atomically decrement the parent database stats
	totalDeletedSize := meta.Filesize + meta.PreviewSize
	statsQuery, statsArgs, err := r.Builder.Update("databases").
		Set("entry_count", squirrel.Expr("MAX(0, entry_count - 1)")).
		Set("total_disk_space_bytes", squirrel.Expr("MAX(0, total_disk_space_bytes - ?)", totalDeletedSize)).
		Where(squirrel.Eq{"name": dbname}).
		ToSql()
	if err != nil {
		return repo.DeletedEntryMeta{}, fmt.Errorf("failed to build stats update query: %w", err)
	}

	if _, err := tx.ExecContext(ctx, statsQuery, statsArgs...); err != nil {
		return repo.DeletedEntryMeta{}, fmt.Errorf("failed to update database stats: %w", err)
	}

	// 4. Commit Transaction
	if err := tx.Commit(); err != nil {
		return repo.DeletedEntryMeta{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return meta, nil
}

// DeleteEntries removes multiple entries in a single transaction and updates the database statistics once.
func (r *SQLiteRepository) DeleteEntries(ctx context.Context, dbname string, entryIDs []int64) ([]repo.DeletedEntryMeta, error) {
	if len(entryIDs) == 0 {
		return nil, customerrors.ErrNotFound
	}

	tableName := fmt.Sprintf(`"entries_%s"`, dbname)

	// 1. Begin SQL Transaction
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 2. Delete the rows and retrieve their sizes using RETURNING
	deleteQuery, deleteArgs, err := r.Builder.Delete(tableName).
		Where(squirrel.Eq{"id": entryIDs}).
		Suffix("RETURNING id, filesize, preview_filesize").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build bulk delete query: %w", err)
	}

	rows, err := tx.QueryContext(ctx, deleteQuery, deleteArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute bulk delete: %w", err)
	}
	defer rows.Close()

	var deletedMetas []repo.DeletedEntryMeta
	var totalDeletedSize uint64
	var deletedCount int

	for rows.Next() {
		var meta repo.DeletedEntryMeta
		if err := rows.Scan(&meta.ID, &meta.Filesize, &meta.PreviewSize); err != nil {
			return nil, fmt.Errorf("failed to scan deleted entry meta: %w", err)
		}
		deletedMetas = append(deletedMetas, meta)
		totalDeletedSize += meta.Filesize + meta.PreviewSize
		deletedCount++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error during bulk delete: %w", err)
	}
	rows.Close() // close read lock immediately instead of waiting on defer

	// If no rows were actually deleted (e.g., IDs didn't exist), we can safely commit and return
	if deletedCount == 0 {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit empty transaction: %w", err)
		}
		return deletedMetas, nil
	}

	// 3. Atomically decrement the parent database stats in one operation
	statsQuery, statsArgs, err := r.Builder.Update("databases").
		Set("entry_count", squirrel.Expr("MAX(0, entry_count - ?)", deletedCount)).
		Set("total_disk_space_bytes", squirrel.Expr("MAX(0, total_disk_space_bytes - ?)", totalDeletedSize)).
		Where(squirrel.Eq{"name": dbname}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build stats bulk update query: %w", err)
	}

	if _, err := tx.ExecContext(ctx, statsQuery, statsArgs...); err != nil {
		return nil, fmt.Errorf("failed to update database stats: %w", err)
	}

	// 4. Commit Transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return deletedMetas, nil
}

// SearchEntries retrieves entries matching complex nested filter criteria.
func (r *SQLiteRepository) SearchEntries(ctx context.Context, dbname string, req repo.SearchRequest, customFields []repo.CustomField) ([]repo.Entry, error) {
	tableName := fmt.Sprintf(`"entries_%s"`, dbname)
	builder := r.Builder.Select("*").From(tableName)

	// 1. Build Filter Conditions securely
	if req.Filter != nil && len(req.Filter.Conditions) > 0 {
		var andExpr squirrel.And
		var orExpr squirrel.Or
		isOr := strings.ToLower(req.Filter.Operator) == "or"

		for _, cond := range req.Filter.Conditions {
			safeField, err := r.validateAndFormatSearchField(cond.Field, customFields)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", customerrors.ErrValidation, err)
			}

			if !isValidOperator(cond.Operator) {
				return nil, fmt.Errorf("%w: invalid operator '%s'", customerrors.ErrValidation, cond.Operator)
			}

			// Safely assemble the SQL condition using squirrel.Expr
			expr := squirrel.Expr(fmt.Sprintf("%s %s ?", safeField, cond.Operator), cond.Value)
			if isOr {
				orExpr = append(orExpr, expr)
			} else {
				andExpr = append(andExpr, expr)
			}
		}

		if isOr {
			builder = builder.Where(orExpr)
		} else {
			builder = builder.Where(andExpr)
		}
	}

	// 2. Build Sorting securely
	if req.Sort != nil && req.Sort.Field != "" {
		safeField, err := r.validateAndFormatSearchField(req.Sort.Field, customFields)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", customerrors.ErrValidation, err)
		}

		dir := "DESC"
		if strings.ToLower(req.Sort.Direction) == "asc" {
			dir = "ASC"
		}
		builder = builder.OrderBy(fmt.Sprintf("%s %s", safeField, dir))
	} else {
		builder = builder.OrderBy("timestamp DESC")
	}

	// 3. Build Pagination
	if req.Pagination.Limit > 0 {
		builder = builder.Limit(uint64(req.Pagination.Limit))
	}
	if req.Pagination.Offset > 0 {
		builder = builder.Offset(uint64(req.Pagination.Offset))
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build search query: %w", err)
	}

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	entries, err := r.scanEntryRows(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to scan search results: %w", err)
	}

	return entries, nil
}
