//go:build !windows

package ffmpeg

import (
	"os"
)

// getTempDir checks if a native Linux RAM disk is available.
// If it is, it returns that path to ensure zero disk-wear during conversions.
func getTempDir(requestedDir string) string {
	// If the user explicitly requested a specific directory, respect it
	if requestedDir != "" {
		return requestedDir
	}

	// Probe for the Linux RAM disk
	if _, err := os.Stat("/dev/shm"); err == nil {
		return "/dev/shm"
	}

	// Fallback to "" allows os.CreateTemp to use the default OS temp folder (e.g., /tmp)
	return ""
}

// createInMemoryFile
// actively seeks out a tmpfs RAM disk to completely bypass the physical drive.
func createInMemoryFile(dir, pattern string) (string, error) {
	optimalDir := getTempDir(dir)

	tmpFile, err := os.CreateTemp(optimalDir, pattern)
	if err != nil {
		return "", err
	}

	tmpPath := tmpFile.Name()

	// Close the handle immediately so FFmpeg can take exclusive ownership
	tmpFile.Close()

	return tmpPath, nil
}
