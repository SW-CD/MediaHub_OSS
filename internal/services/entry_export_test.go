// filepath: internal/services/entry_export_test.go
package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"mediahub/internal/models"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestService_ExportEntries(t *testing.T) {
	service, repo, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	dbName := "ExportDB"

	dbSvc := NewDatabaseService(repo, NewStorageService(service.Cfg))
	_, err := dbSvc.CreateDatabase(models.DatabaseCreatePayload{
		Name:        dbName,
		ContentType: "image",
		CustomFields: []models.CustomField{
			{Name: "reviewer", Type: "TEXT"},
		},
	})
	assert.NoError(t, err)

	tx, _ := repo.BeginTx()
	entryMeta := models.Entry{
		"timestamp": time.Now().Unix(),
		"filesize":  int64(5),
		"filename":  "photo.jpg",
		"mime_type": "image/jpeg",
		"width":     800,
		"height":    600,
		"status":    "ready",
		"reviewer":  "Alice",
	}
	created, _ := tx.CreateEntryInTx(dbName, "image", entryMeta, []models.CustomField{{Name: "reviewer", Type: "TEXT"}})
	tx.Commit()

	id := created["id"].(int64)
	ts := created["timestamp"].(int64)

	entryPath, _ := service.Storage.GetEntryPath(dbName, ts, id)
	os.MkdirAll(filepath.Dir(entryPath), 0755)
	os.WriteFile(entryPath, []byte("IMAGE"), 0644)

	var buf bytes.Buffer
	err = service.ExportEntries(context.Background(), dbName, []int64{id}, &buf)
	assert.NoError(t, err)

	reader := bytes.NewReader(buf.Bytes())
	zipReader, err := zip.NewReader(reader, int64(buf.Len()))
	assert.NoError(t, err)

	filesInZip := make(map[string]*zip.File)
	for _, f := range zipReader.File {
		filesInZip[f.Name] = f
	}

	assert.Contains(t, filesInZip, "_metadata.json")
	assert.Contains(t, filesInZip, "entries.csv")

	csvFile, _ := filesInZip["entries.csv"].Open()
	csvReader := csv.NewReader(csvFile)
	csvRecords, _ := csvReader.ReadAll()
	csvFile.Close()

	header := csvRecords[0]
	assert.Contains(t, header, "reviewer")

	foundFile := false
	for name, f := range filesInZip {
		if name == "_metadata.json" || name == "entries.csv" {
			continue
		}
		rc, _ := f.Open()
		content, _ := io.ReadAll(rc)
		rc.Close()
		if string(content) == "IMAGE" {
			foundFile = true
		}
	}
	assert.True(t, foundFile, "Exported file content mismatch")
}

func TestService_ExportEntries_ContextCancel(t *testing.T) {
	service, repo, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	dbSvc := NewDatabaseService(repo, NewStorageService(service.Cfg))
	dbSvc.CreateDatabase(models.DatabaseCreatePayload{Name: "CancelDB", ContentType: "file"})

	tx, _ := repo.BeginTx()
	// --- FIX: Add "filename" to prevent DB NOT NULL constraint violation ---
	created, err := tx.CreateEntryInTx("CancelDB", "file", models.Entry{
		"timestamp": int64(123),
		"mime_type": "x",
		"status":    "ready",
		"filename":  "cancel_test.bin",
	}, nil)
	assert.NoError(t, err, "Setup failed: CreateEntryInTx returned error")

	tx.Commit()
	id := created["id"].(int64)

	path, _ := service.Storage.GetEntryPath("CancelDB", 123, id)
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, []byte("x"), 0644)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	err = service.ExportEntries(ctx, "CancelDB", []int64{id}, &buf)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}
