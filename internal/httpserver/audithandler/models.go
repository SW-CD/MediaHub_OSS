package audithandler

import (
	"log/slog"
	"mediahub_oss/internal/repository"
)

type AuditHandler struct {
	Logger *slog.Logger
	Repo   repository.Repository
}

// AuditLogResponse defines the JSON structure for the outbound audit logs.
type AuditLogResponse struct {
	ID        int64          `json:"id"`
	Timestamp int64          `json:"timestamp"`
	Action    string         `json:"action"`
	Actor     string         `json:"actor"`
	Resource  string         `json:"resource"`
	Details   map[string]any `json:"details"`
}
