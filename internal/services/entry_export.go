// filepath: internal/services/entry_export.go
package services

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mediahub/internal/logging"
	"mediahub/internal/models"
	"os"
	"strconv"
	"time"
)

// ExportEntries streams selected entries as a ZIP archive containing a CSV index and file folders.
func (s *entryService) ExportEntries(ctx context.Context, dbName string, ids []int64, w io.Writer) error {
	// 1. Fetch Database details (for custom fields)
	db, err := s.Repo.GetDatabase(dbName)
	if err != nil {
		return fmt.Errorf("database not found: %w", err)
	}

	// 2. Fetch Entry Metadata
	// For export, we fetch all metadata at once. If this list becomes massive (millions),
	// we might need cursor-based pagination, but for "selected IDs" in a frontend, this is fine.
	entries, err := s.Repo.GetEntriesByID(dbName, ids, db.CustomFields)
	if err != nil {
		return fmt.Errorf("failed to fetch entries: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no entries found matching the provided IDs")
	}

	// 3. Initialize ZIP Writer
	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	// 4. Write _metadata.json (Database Config)
	if err := s.writeDBMetadataToZip(zipWriter, db); err != nil {
		logging.Log.Errorf("Export: Failed to write _metadata.json: %v", err)
		return err
	}

	// 5. Write entries.csv
	if err := s.writeCSVToZip(zipWriter, db, entries); err != nil {
		logging.Log.Errorf("Export: Failed to write entries.csv: %v", err)
		return err
	}

	// 6. Stream Files
	for _, entry := range entries {
		// Check for context cancellation (client disconnect)
		if ctx.Err() != nil {
			logging.Log.Warnf("Export cancelled by client context: %v", ctx.Err())
			return ctx.Err()
		}

		if err := s.writeFileToZip(zipWriter, db.Name, entry); err != nil {
			// Log error but continue exporting other files
			id, _ := entry["id"]
			logging.Log.Warnf("Export: Failed to add file for entry %v: %v", id, err)
		}
	}

	return nil
}

// writeDBMetadataToZip adds the database configuration to the zip.
func (s *entryService) writeDBMetadataToZip(zw *zip.Writer, db *models.Database) error {
	f, err := zw.Create("_metadata.json")
	if err != nil {
		return err
	}

	// Pretty print JSON
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(db)
}

// writeCSVToZip generates the CSV index inside the zip.
func (s *entryService) writeCSVToZip(zw *zip.Writer, db *models.Database, entries []models.Entry) error {
	f, err := zw.Create("entries.csv")
	if err != nil {
		return err
	}

	// Handle BOM for Excel compatibility (Optional, but usually helpful)
	f.Write([]byte{0xEF, 0xBB, 0xBF})

	csvWriter := csv.NewWriter(f)
	defer csvWriter.Flush()

	// --- Build Header ---
	// Standard fields first
	header := []string{"id", "filename", "timestamp_iso", "filesize", "mime_type", "status"}

	// Type specific fields
	if db.ContentType == "image" {
		header = append(header, "width", "height")
	} else if db.ContentType == "audio" {
		header = append(header, "duration_sec", "channels")
	}

	// Custom fields
	for _, cf := range db.CustomFields {
		header = append(header, cf.Name)
	}

	if err := csvWriter.Write(header); err != nil {
		return err
	}

	// --- Write Rows ---
	for _, entry := range entries {
		row := make([]string, 0, len(header))

		// 1. Standard Fields
		id := entry["id"].(int64)
		row = append(row, strconv.FormatInt(id, 10))

		filename, _ := entry["filename"].(string)
		row = append(row, filename)

		ts := entry["timestamp"].(int64)
		row = append(row, time.Unix(ts, 0).Format(time.RFC3339))

		size, _ := entry["filesize"].(int64)
		row = append(row, strconv.FormatInt(size, 10))

		mime, _ := entry["mime_type"].(string)
		row = append(row, mime)

		status, _ := entry["status"].(string)
		row = append(row, status)

		// 2. Type Specific Fields
		if db.ContentType == "image" {
			// Careful with type assertions, SQLite drivers can return different int types
			w := toInt64(entry["width"])
			h := toInt64(entry["height"])
			row = append(row, strconv.FormatInt(w, 10))
			row = append(row, strconv.FormatInt(h, 10))
		} else if db.ContentType == "audio" {
			dur := toFloat64(entry["duration_sec"])
			ch := toInt64(entry["channels"])
			row = append(row, fmt.Sprintf("%.3f", dur))
			row = append(row, strconv.FormatInt(ch, 10))
		}

		// 3. Custom Fields
		for _, cf := range db.CustomFields {
			val := entry[cf.Name]
			row = append(row, fmt.Sprintf("%v", val))
		}

		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// writeFileToZip adds the actual media file to the zip.
func (s *entryService) writeFileToZip(zw *zip.Writer, dbName string, entry models.Entry) error {
	id := entry["id"].(int64)
	timestamp := entry["timestamp"].(int64)
	filename, ok := entry["filename"].(string)
	if !ok || filename == "" {
		// Fallback if filename is missing (shouldn't happen for valid entries)
		filename = fmt.Sprintf("%d.bin", id)
	}

	// Determine physical path
	srcPath, err := s.Storage.GetEntryPath(dbName, timestamp, id)
	if err != nil {
		return err
	}

	// Open file on disk
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Determine path inside ZIP
	// Structure: YYYY/MM/ID_Filename
	t := time.Unix(timestamp, 0)
	zipPath := fmt.Sprintf("%s/%s/%d_%s", t.Format("2006"), t.Format("01"), id, filename)

	// Create zip entry
	// We use CreateHeader to preserve modification time
	info, err := srcFile.Stat()
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = zipPath
	header.Method = zip.Deflate // Enable compression

	dstFile, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	// Stream copy
	_, err = io.Copy(dstFile, srcFile)
	return err
}

// Helpers for robust type assertion from generic maps
func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch i := v.(type) {
	case int64:
		return i
	case int:
		return int64(i)
	case float64:
		return int64(i)
	default:
		return 0
	}
}

func toFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch f := v.(type) {
	case float64:
		return f
	case float32:
		return float64(f)
	case int64:
		return float64(f)
	case int:
		return float64(f)
	default:
		return 0
	}
}
