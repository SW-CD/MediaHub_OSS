package s3storage

import (
	"context"
	"io"

	"mediahub_oss/internal/shared/customerrors"
	"mediahub_oss/internal/storage"
)

type S3StorageProvider struct{}

func NewS3StorageProvider() (S3StorageProvider, error) {
	return S3StorageProvider{}, customerrors.ErrNotImplemented
}

func (s *S3StorageProvider) Write(ctx context.Context, dbID string, id int64, content io.Reader) (int64, error) {
	return 0, customerrors.ErrNotImplemented
}

func (s *S3StorageProvider) WritePreview(ctx context.Context, dbID string, id int64, preview io.Reader) (int64, error) {
	return 0, customerrors.ErrNotImplemented
}

// Stat retrieves metadata about the main file without downloading the content.
func (s *S3StorageProvider) Stat(ctx context.Context, dbID string, id int64) (storage.FileInfo, error) {
	return storage.FileInfo{}, customerrors.ErrNotImplemented
}

// StatPreview retrieves metadata about the preview file without downloading the content.
func (s *S3StorageProvider) StatPreview(ctx context.Context, dbID string, id int64) (storage.FileInfo, error) {
	return storage.FileInfo{}, customerrors.ErrNotImplemented
}

func (s *S3StorageProvider) Read(ctx context.Context, dbID string, id int64, offset int64, length int64) (io.ReadCloser, error) {
	return nil, customerrors.ErrNotImplemented
}

func (s *S3StorageProvider) ReadPreview(ctx context.Context, dbID string, id int64) (io.ReadCloser, error) {
	return nil, customerrors.ErrNotImplemented
}

func (s *S3StorageProvider) Delete(ctx context.Context, dbID string, id int64) error {
	return customerrors.ErrNotImplemented
}

func (s *S3StorageProvider) DeleteMultiple(ctx context.Context, dbID string, ids []int64) (storage.BulkDeleteResult, error) {
	return storage.BulkDeleteResult{}, customerrors.ErrNotImplemented
}

func (s *S3StorageProvider) DeletePreview(ctx context.Context, dbID string, id int64) error {
	return customerrors.ErrNotImplemented
}

func (s *S3StorageProvider) DeleteMultiplePreviews(ctx context.Context, dbID string, ids []int64) (storage.BulkDeleteResult, error) {
	return storage.BulkDeleteResult{}, customerrors.ErrNotImplemented
}

func (s *S3StorageProvider) Walk(ctx context.Context, dbID string, walkFn func(id int64, info storage.FileInfo) error) error {
	return customerrors.ErrNotImplemented
}

func (s *S3StorageProvider) WalkPreview(ctx context.Context, dbID string, walkFn func(id int64, info storage.FileInfo) error) error {
	return customerrors.ErrNotImplemented
}
