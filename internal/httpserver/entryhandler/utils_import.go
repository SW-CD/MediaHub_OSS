package entryhandler

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	repo "mediahub_oss/internal/repository"
)

// processImportJob handles the asynchronous extraction and database insertion for bulk imports.
func (h *EntryHandler) processImportJob(ctx context.Context, db repo.Database, username string, tempZipPath string, config ImportConfigPayload) {
	// Ensure the temporary ZIP file is deleted from the server once the job finishes
	defer os.Remove(tempZipPath)

	h.Logger.Info("Background import job started", "database_id", db.ID, "user", username, "mode", config.Mode)

	// 1. Open the ZIP archive
	zr, err := zip.OpenReader(tempZipPath)
	if err != nil {
		h.Logger.Error("Import failed: Could not open ZIP archive", "database_id", db.ID, "error", err)
		return
	}
	defer zr.Close()

	// 2. Index the ZIP contents in a map for O(1) lookups
	zipFiles := make(map[string]*zip.File)
	var csvZipFile *zip.File

	for _, f := range zr.File {
		zipFiles[f.Name] = f
		if f.Name == "entries.csv" {
			csvZipFile = f
		}
	}

	if csvZipFile == nil {
		h.Logger.Error("Import failed: entries.csv not found in archive", "database_id", db.ID)
		return
	}

	// 3. Open and Parse entries.csv
	csvFile, err := csvZipFile.Open()
	if err != nil {
		h.Logger.Error("Import failed: Could not read entries.csv", "database_id", db.ID, "error", err)
		return
	}
	defer csvFile.Close()

	csvReader := csv.NewReader(csvFile)
	headers, err := csvReader.Read()
	if err != nil {
		h.Logger.Error("Import failed: Could not read CSV headers", "database_id", db.ID, "error", err)
		return
	}

	// 4. Validate Headers
	expectedHeaders := []string{"id", "filename", "timestamp", "filesize", "previewsize", "mime_type", "status"}
	if len(headers) < 7 {
		h.Logger.Error("Import failed: CSV has missing standard headers", "database_id", db.ID)
		return
	}
	for i, expected := range expectedHeaders {
		if headers[i] != expected {
			h.Logger.Error("Import failed: Invalid CSV standard header format", "database_id", db.ID, "expected", expected, "got", headers[i])
			return
		}
	}

	// 5. Track Statistics
	var successCount, skipCount, errorCount int

	// 6. Iterate through CSV rows
	for rowNum := 2; ; rowNum++ { // Start at 2 because row 1 is the header
		row, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			h.Logger.Warn("Import warning: Could not read CSV row", "row", rowNum, "error", err)
			errorCount++
			continue
		}

		// Parse standard columns
		csvID, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil {
			h.Logger.Warn("Import warning: Invalid ID format in CSV", "row", rowNum, "id", row[0])
			errorCount++
			continue
		}

		// Determine target ID based on mode
		var targetID int64 = 0 // 0 tells the Repo to generate a new Auto-Increment ID
		if config.Mode != "generate_new" {
			targetID = csvID

			// Check if entry exists for skip/overwrite
			_, err := h.Repo.GetEntry(ctx, db.ID, targetID)
			exists := err == nil

			if config.Mode == "skip" && exists {
				skipCount++
				continue
			}
			// If mode is overwrite, we proceed and just update the existing record
		}

		// Parse remaining standard fields
		filename := row[1]
		timestamp, _ := time.Parse(time.RFC3339, row[2])
		filesize, _ := strconv.ParseUint(row[3], 10, 64)
		previewsize, _ := strconv.ParseUint(row[4], 10, 64)
		mimeType := row[5]
		status, _ := strconv.ParseUint(row[6], 10, 8)

		// Map Custom Fields
		mappedCustomFields := make(map[string]any)
		failedMapping := false

		for i := 7; i < len(headers); i++ {
			if i >= len(row) {
				break
			}

			csvHeader := headers[i]
			dbField := csvHeader

			// Apply mapping if defined
			if mapped, ok := config.CustomFieldMapping[csvHeader]; ok {
				dbField = mapped
			}

			// Validate against database schema
			validField := false
			for _, cf := range db.CustomFields {
				if cf.Name == dbField {
					validField = true
					// Type cast based on schema
					switch cf.Type {
					case "INTEGER":
						val, _ := strconv.ParseInt(row[i], 10, 64)
						mappedCustomFields[dbField] = val
					case "REAL":
						val, _ := strconv.ParseFloat(row[i], 64)
						mappedCustomFields[dbField] = val
					case "BOOLEAN":
						val, _ := strconv.ParseBool(row[i])
						mappedCustomFields[dbField] = val
					default: // TEXT
						mappedCustomFields[dbField] = row[i]
					}
					break
				}
			}

			if !validField && config.UnmappedFields == "fail" {
				h.Logger.Error("Import aborted: Unmapped field encountered", "database_id", db.ID, "field", csvHeader)
				return // Hard abort per configuration
			}
		}

		if failedMapping {
			errorCount++
			continue
		}

		// 7. Locate Files in ZIP
		mainZipPath := fmt.Sprintf("files/%d_%s", csvID, filename)
		previewZipPath := fmt.Sprintf("previews/%d.webp", csvID)

		mainFileZipped, ok := zipFiles[mainZipPath]
		if !ok {
			h.Logger.Warn("Import warning: Main media file missing in archive", "row", rowNum, "file", mainZipPath)
			errorCount++
			continue
		}

		// 8. Construct Entry Model
		entry := repo.Entry{
			ID:           targetID,
			FileName:     filename,
			Size:         filesize,
			PreviewSize:  previewsize,
			Timestamp:    timestamp,
			MimeType:     mimeType,
			Status:       uint8(status),
			CustomFields: mappedCustomFields,
		}

		// 9. Write to Database
		var savedEntry repo.Entry
		if config.Mode == "overwrite" && targetID != 0 {
			// Ensure it actually exists before updating, or fallback to create
			_, errCheck := h.Repo.GetEntry(ctx, db.ID, targetID)
			if errCheck == nil {
				savedEntry, err = h.Repo.UpdateEntry(ctx, db.ID, entry)
			} else {
				savedEntry, err = h.Repo.CreateEntry(ctx, db, entry)
			}
		} else {
			savedEntry, err = h.Repo.CreateEntry(ctx, db, entry)
		}

		if err != nil {
			h.Logger.Warn("Import warning: Failed to insert entry into database", "row", rowNum, "error", err)
			errorCount++
			continue
		}

		// 10. Write Main File to Storage
		srcFile, _ := mainFileZipped.Open()
		_, err = h.Storage.Write(ctx, db.ID, savedEntry.ID, srcFile)
		srcFile.Close()

		if err != nil {
			h.Logger.Warn("Import warning: Failed to write main file to storage", "row", rowNum, "error", err)
			h.Repo.DeleteEntry(ctx, db.ID, savedEntry.ID) // Rollback DB on storage failure
			errorCount++
			continue
		}

		// 11. Write Preview to Storage (if exists)
		if previewZipped, exists := zipFiles[previewZipPath]; exists {
			pSrcFile, _ := previewZipped.Open()
			_, err = h.Storage.WritePreview(ctx, db.ID, savedEntry.ID, pSrcFile)
			pSrcFile.Close()
			if err != nil {
				h.Logger.Warn("Import warning: Failed to write preview file to storage", "row", rowNum, "error", err)
				// We don't rollback the whole entry just for a failed preview, but log it
			}
		}

		successCount++
	}

	// 12. Log Summary
	h.Logger.Info("Background import job completed",
		"database_id", db.ID,
		"successful", successCount,
		"skipped", skipCount,
		"errors", errorCount,
	)
}
