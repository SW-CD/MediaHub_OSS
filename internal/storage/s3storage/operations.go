package s3storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"

	"mediahub_oss/internal/shared/customerrors"
	"mediahub_oss/internal/storage"
)

// Write uploads a file stream to the S3 bucket.
func (s *S3StorageProvider) Write(ctx context.Context, dbID string, id int64, content io.Reader) (int64, error) {
	objectKey := getObjectKey(dbID, id)
	size := getStreamSize(content)
	info, err := s.client.PutObject(ctx, s.bucket, objectKey, content, size, minio.PutObjectOptions{
		ContentType:          "application/octet-stream",
		DisableContentSha256: true,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to write object to s3: %w", err)
	}
	return info.Size, nil
}

// WritePreview uploads a preview file stream to the S3 bucket.
func (s *S3StorageProvider) WritePreview(ctx context.Context, dbID string, id int64, preview io.Reader) (int64, error) {
	objectKey := getPreviewObjectKey(dbID, id)
	size := getStreamSize(preview)
	info, err := s.client.PutObject(ctx, s.bucket, objectKey, preview, size, minio.PutObjectOptions{
		ContentType:          "image/webp",
		DisableContentSha256: true,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to write preview object to s3: %w", err)
	}
	return info.Size, nil
}

// Stat retrieves metadata about the main file without downloading the content.
func (s *S3StorageProvider) Stat(ctx context.Context, dbID string, id int64) (storage.FileInfo, error) {
	objectKey := getObjectKey(dbID, id)
	info, err := s.client.StatObject(ctx, s.bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		if isNotFoundError(err) {
			return storage.FileInfo{}, customerrors.ErrNotFound
		}
		return storage.FileInfo{}, err
	}
	return storage.FileInfo{
		Size:         info.Size,
		LastModified: info.LastModified,
	}, nil
}

// StatPreview retrieves metadata about the preview file without downloading the content.
func (s *S3StorageProvider) StatPreview(ctx context.Context, dbID string, id int64) (storage.FileInfo, error) {
	objectKey := getPreviewObjectKey(dbID, id)
	info, err := s.client.StatObject(ctx, s.bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		if isNotFoundError(err) {
			return storage.FileInfo{}, customerrors.ErrNotFound
		}
		return storage.FileInfo{}, err
	}
	return storage.FileInfo{
		Size:         info.Size,
		LastModified: info.LastModified,
	}, nil
}

// Read retrieves a stream of the file content, supporting range requests.
func (s *S3StorageProvider) Read(ctx context.Context, dbID string, id int64, offset int64, length int64) (io.ReadCloser, error) {
	objectKey := getObjectKey(dbID, id)
	var opts minio.GetObjectOptions
	if offset > 0 && length >= 0 {
		if err := opts.SetRange(offset, offset+length-1); err != nil {
			return nil, err
		}
	} else if offset > 0 && length < 0 {
		if err := opts.SetRange(offset, 0); err != nil {
			return nil, err
		}
	} else if offset == 0 && length >= 0 {
		if err := opts.SetRange(0, length-1); err != nil {
			return nil, err
		}
	}

	obj, err := s.client.GetObject(ctx, s.bucket, objectKey, opts)
	if err != nil {
		if isNotFoundError(err) {
			return nil, customerrors.ErrNotFound
		}
		return nil, err
	}

	if _, err := obj.Stat(); err != nil {
		obj.Close()
		if isNotFoundError(err) {
			return nil, customerrors.ErrNotFound
		}
		return nil, err
	}

	if offset > 0 {
		if _, err := obj.Seek(offset, io.SeekStart); err != nil {
			obj.Close()
			return nil, err
		}
	}

	if length >= 0 {
		return &limitedReadCloser{
			Reader: io.LimitReader(obj, length),
			Closer: obj,
		}, nil
	}

	return obj, nil
}

// ReadPreview retrieves a stream of the preview file content.
func (s *S3StorageProvider) ReadPreview(ctx context.Context, dbID string, id int64) (io.ReadCloser, error) {
	objectKey := getPreviewObjectKey(dbID, id)
	obj, err := s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		if isNotFoundError(err) {
			return nil, customerrors.ErrNotFound
		}
		return nil, err
	}

	if _, err := obj.Stat(); err != nil {
		obj.Close()
		if isNotFoundError(err) {
			return nil, customerrors.ErrNotFound
		}
		return nil, err
	}

	return obj, nil
}
