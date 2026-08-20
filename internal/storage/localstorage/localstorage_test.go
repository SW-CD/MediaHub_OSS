package localstorage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"mediahub_oss/internal/shared/customerrors"
	"mediahub_oss/internal/storage/localstorage"
)

type errorReader struct {
	readBytes int
	maxBytes  int
	err       error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	if r.readBytes >= r.maxBytes {
		return 0, r.err
	}
	remaining := r.maxBytes - r.readBytes
	if len(p) > remaining {
		p = p[:remaining]
	}
	for i := range p {
		p[i] = 'A'
	}
	r.readBytes += len(p)
	return len(p), nil
}

func TestLocalStorage_Read_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := &localstorage.LocalStorage{RootPath: tmpDir}
	ctx := context.Background()

	t.Run("Read non-existent file returns ErrNotFound", func(t *testing.T) {
		_, err := store.Read(ctx, "01HGFB9Z5W7ABCDEFGHJKMNPQR", 99999, 0, -1)
		if !errors.Is(err, customerrors.ErrNotFound) {
			t.Fatalf("expected customerrors.ErrNotFound, got: %v", err)
		}
	})

	t.Run("ReadPreview non-existent file returns ErrNotFound", func(t *testing.T) {
		_, err := store.ReadPreview(ctx, "01HGFB9Z5W7ABCDEFGHJKMNPQR", 99999)
		if !errors.Is(err, customerrors.ErrNotFound) {
			t.Fatalf("expected customerrors.ErrNotFound, got: %v", err)
		}
	})
}

func TestLocalStorage_Write_AtomicCleanupOnError(t *testing.T) {
	tmpDir := t.TempDir()
	store := &localstorage.LocalStorage{RootPath: tmpDir}
	ctx := context.Background()
	dbID := "01HGFB9Z5W7ABCDEFGHJKMNPQR"
	var entryID int64 = 10232

	// Writer fails midway
	faultyReader := &errorReader{
		maxBytes: 100,
		err:      errors.New("network connection dropped"),
	}

	_, err := store.Write(ctx, dbID, entryID, faultyReader)
	if err == nil {
		t.Fatalf("expected write error, got nil")
	}

	// Verify that no partial file or temporary file is left on disk
	expectedDir := filepath.Join(tmpDir, dbID, "10")
	expectedPath := filepath.Join(expectedDir, "10232")

	if _, err := os.Stat(expectedPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected target file %s to not exist after failed write, but found it", expectedPath)
	}

	// Check that no orphaned temp files exist in directory
	entries, err := os.ReadDir(expectedDir)
	if err == nil {
		for _, e := range entries {
			t.Fatalf("unexpected leftover file in directory %s: %s", expectedDir, e.Name())
		}
	}
}

func TestLocalStorage_WriteAndReadOperations(t *testing.T) {
	tmpDir := t.TempDir()
	store := &localstorage.LocalStorage{RootPath: tmpDir}
	ctx := context.Background()
	dbID := "01HGFB9Z5W7ABCDEFGHJKMNPQR"
	var entryID int64 = 10232

	content := []byte("hello mediahub local storage byte range test")
	written, err := store.Write(ctx, dbID, entryID, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if written != int64(len(content)) {
		t.Fatalf("written = %d, want %d", written, len(content))
	}

	// 1. Stat
	stat, err := store.Stat(ctx, dbID, entryID)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if stat.Size != int64(len(content)) {
		t.Fatalf("stat size = %d, want %d", stat.Size, len(content))
	}

	// 2. Read full file
	rc, err := store.Read(ctx, dbID, entryID, 0, -1)
	if err != nil {
		t.Fatalf("Read full file failed: %v", err)
	}
	readAll, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if !bytes.Equal(readAll, content) {
		t.Fatalf("Read content mismatch: got %s, want %s", string(readAll), string(content))
	}

	// 3. Read range
	offset := int64(6)
	length := int64(8) // "mediahub"
	rcRange, err := store.Read(ctx, dbID, entryID, offset, length)
	if err != nil {
		t.Fatalf("Read range failed: %v", err)
	}
	rangeBytes, err := io.ReadAll(rcRange)
	rcRange.Close()
	if err != nil {
		t.Fatalf("ReadAll range failed: %v", err)
	}
	if string(rangeBytes) != "mediahub" {
		t.Fatalf("expected 'mediahub', got %q", string(rangeBytes))
	}

	// 4. Write Preview and Read Preview
	previewContent := []byte("mock webp preview")
	pWritten, err := store.WritePreview(ctx, dbID, entryID, bytes.NewReader(previewContent))
	if err != nil {
		t.Fatalf("WritePreview failed: %v", err)
	}
	if pWritten != int64(len(previewContent)) {
		t.Fatalf("pWritten = %d, want %d", pWritten, len(previewContent))
	}

	pStat, err := store.StatPreview(ctx, dbID, entryID)
	if err != nil {
		t.Fatalf("StatPreview failed: %v", err)
	}
	if pStat.Size != int64(len(previewContent)) {
		t.Fatalf("pStat size = %d, want %d", pStat.Size, len(previewContent))
	}

	pRc, err := store.ReadPreview(ctx, dbID, entryID)
	if err != nil {
		t.Fatalf("ReadPreview failed: %v", err)
	}
	pRead, err := io.ReadAll(pRc)
	pRc.Close()
	if err != nil {
		t.Fatalf("ReadPreview read failed: %v", err)
	}
	if !bytes.Equal(pRead, previewContent) {
		t.Fatalf("ReadPreview mismatch: got %s, want %s", string(pRead), string(previewContent))
	}
}
