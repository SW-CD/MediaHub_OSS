package storage

import "time"

// Returned by bulk delete requests, indicating which files were really deleted
type BulkDeleteResult struct {
	Success []int64
	Failed  []int64
}

// FileInfo holds storage-agnostic metadata about a stored file.
type FileInfo struct {
	Size         int64
	LastModified time.Time
}
