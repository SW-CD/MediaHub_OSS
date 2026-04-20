package audit

import (
	"context"
	"log/slog"
	"mediahub_oss/internal/repository"
)

type AuditLogger interface {
	Log(ctx context.Context, action string, actor string, resource string, details map[string]any)
}

func NewAuditLogger(enabled bool, ltype string, logger *slog.Logger, repo repository.Repository) AuditLogger {
	if !enabled {
		return NewAlNoopLogger()
	}

	switch ltype {
	case "stdio":
		return NewAlStdout(logger)
	case "database":
		return NewAlDatabase(repo)
	default:
		logger.Warn("Undefined audit logger type, falling back to noop logger", "type", ltype)
		return NewAlNoopLogger()
	}

}
