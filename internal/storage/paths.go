// internal/storage/paths.go
// New file for path generation logic.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// getStoragePath is an internal helper to create and validate storage paths.
// rootSubDirs are joined *after* storageRoot (e.g., "previews", dbName)
func getStoragePath(storageRoot string, timestamp, entryID int64, rootSubDirs ...string) (string, error) {
	t := time.Unix(timestamp, 0)
	year := t.Format("2006")
	month := t.Format("01")

	// Combine the dynamic time parts with the other sub-directories
	allDirs := append(rootSubDirs, year, month)

	dir := filepath.Join(storageRoot, filepath.Join(allDirs...))

	// --- SECURITY: Prevent Path Traversal ---
	cleanedDir := filepath.Clean(dir)
	cleanedRoot := filepath.Clean(storageRoot)
	if !strings.HasPrefix(cleanedDir, cleanedRoot) || cleanedDir == cleanedRoot {
		return "", fmt.Errorf("invalid path: potential path traversal")
	}

	if err := os.MkdirAll(cleanedDir, 0755); err != nil {
		return "", fmt.Errorf("could not create directory structure: %w", err)
	}

	fileName := fmt.Sprintf("%d", entryID)
	return filepath.Join(cleanedDir, fileName), nil
}

// GetEntryPath generates the full, absolute path for an entry file.
// It creates the necessary year/month subdirectories if they don't exist.
func GetEntryPath(storageRoot, dbName string, timestamp, entryID int64) (string, error) {
	// Pass dbName as the only root subdirectory
	return getStoragePath(storageRoot, timestamp, entryID, dbName)
}

// GetPreviewPath generates the full, absolute path for an entry preview file.
// It creates the necessary year/month subdirectories if they don't exist.
func GetPreviewPath(storageRoot, dbName string, timestamp, entryID int64) (string, error) {
	// Pass "previews" *and* dbName as the root subdirectories
	return getStoragePath(storageRoot, timestamp, entryID, "previews", dbName)
}
