package recovery

import (
	"fmt"
	"log/slog"

	"mediahub_oss/internal/cli/config"
	"mediahub_oss/internal/repository"
	"mediahub_oss/internal/repository/postgres"
	"mediahub_oss/internal/repository/sqlite"
	"mediahub_oss/internal/storage"
	"mediahub_oss/internal/storage/localstorage"
	"mediahub_oss/internal/storage/s3storage"
)

// RecoveryService handles maintenance tasks to fix data inconsistencies.
type RecoveryService struct {
	repo    repository.Repository
	storage storage.StorageProvider
	logger  *slog.Logger
	dryRun  bool
}

// NewRecoveryService initializes the necessary repository and storage providers based on the config.
func NewRecoveryService(conf *config.Config, logger *slog.Logger, dryRun bool) (*RecoveryService, error) {
	var repo repository.Repository
	var err error

	// 1. Initialize the Repository based on the config driver
	switch conf.Database.Driver {
	case "sqlite":
		repo, err = sqlite.NewRepository(conf.Database.Source)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize sqlite repository: %w", err)
		}
	case "postgres":
		repo, err = postgres.NewRepository(conf.Database.Source)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize postgres repository: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", conf.Database.Driver)
	}

	// 2. Initialize the Storage Provider based on the config type
	var storageProvider storage.StorageProvider
	switch conf.Storage.Type {
	case "local":
		storageProvider = &localstorage.LocalStorage{
			RootPath: conf.Storage.Local.Root,
		}
	case "s3":
		s3prov, err := s3storage.NewS3StorageProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize s3 storage: %w", err)
		}
		storageProvider = &s3prov
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", conf.Storage.Type)
	}

	return &RecoveryService{
		repo:    repo,
		storage: storageProvider,
		logger:  logger,
		dryRun:  dryRun,
	}, nil
}

// Close cleans up underlying connections, like the database pool.
func (s *RecoveryService) Close() error {
	if s.repo != nil {
		return s.repo.Close()
	}
	return nil
}
