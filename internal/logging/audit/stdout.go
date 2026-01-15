package audit

import (
	"context"
	"mediahub/internal/logging"

	"github.com/sirupsen/logrus"
)

type AuditLoggerSTDOUT struct {
	enabled bool
	logger  *logrus.Logger
}

func (a *AuditLoggerSTDOUT) Init(level string, enabled bool) {
	a.logger = logging.NewLogger(level)
	a.enabled = enabled
}

func (a *AuditLoggerSTDOUT) Log(ctx context.Context, action string, actor string, resource string, details map[string]interface{}) {
	if !a.enabled {
		return
	}

	// Construct fields
	fields := logrus.Fields{
		"audit_action":   action,
		"audit_actor":    actor,
		"audit_resource": resource,
	}

	// Add details flattened into the fields
	// Range over nil map is safe in Go, so explicit nil check is not needed.
	for k, v := range details {
		fields["detail."+k] = v
	}

	// Log at INFO level with a specific prefix to make it easy to grep
	a.logger.WithFields(fields).Info("AUDIT EVENT")
}
