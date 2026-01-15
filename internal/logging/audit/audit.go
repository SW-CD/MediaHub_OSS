// filepath: internal/audit/logger_auditor.go
package audit

import (
	"context"
)

// Interface for AuditLogger
// Open source version implements logger to stdout,
// commercial version adds more logger options
type AuditLogger interface {
	Init(level string)
	Log(ctx context.Context, action string, actor string, resource string, details map[string]interface{})
}
