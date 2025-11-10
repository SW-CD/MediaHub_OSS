// filepath: internal/storage/file.go
// Package storage provides functionality for storing and managing files.
// This file handles saving the *original* file.
package storage

import (
	"fmt"
	"io"
	"os"
)

// SaveFile saves file data from a reader to a specified path.
// It streams the file to avoid loading it entirely into memory.
func SaveFile(fileData io.Reader, path string) (int64, error) {
	// Create the destination file.
	f, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("could not create file: %w", err)
	}
	defer f.Close()

	// Stream the upload data to the file. This is the main data transfer.
	fileSize, err := io.Copy(f, fileData)
	if err != nil {
		return 0, fmt.Errorf("could not write file: %w", err)
	}

	return fileSize, nil
}
