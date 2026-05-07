//go:build windows

package ffmpeg

import (
	"os"
	"syscall"
)

// createInMemoryFile creates a temp file and explicitly flags it
// as temporary (to keep it in RAM) and non-indexed (to block the Windows Search Indexer).
func createInMemoryFile(dir, pattern string) (string, error) {
	// 1. Create the file using the standard Go library
	tmpFile, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}

	tmpPath := tmpFile.Name()

	// 2. Close the handle immediately so FFmpeg can access it exclusively
	tmpFile.Close()

	// 3. Convert the Go string to a UTF-16 pointer (required by Windows APIs)
	pathPtr, err := syscall.UTF16PtrFromString(tmpPath)
	if err == nil {
		// 4. Fetch the current attributes (like standard Archive flags)
		currentAttributes, err := syscall.GetFileAttributes(pathPtr)
		if err == nil {
			// 5. Apply our ultimate optimizations using bitwise OR:
			// 0x100  = FILE_ATTRIBUTE_TEMPORARY (Force to RAM cache)
			// 0x2000 = FILE_ATTRIBUTE_NOT_CONTENT_INDEXED (Block Search Indexer locks)
			optimizedAttributes := currentAttributes | 0x100 | 0x2000

			syscall.SetFileAttributes(pathPtr, optimizedAttributes)
		}
	}

	return tmpPath, nil
}
