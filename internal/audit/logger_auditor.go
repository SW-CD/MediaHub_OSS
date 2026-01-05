// filepath: internal/audit/logger_auditor.go
package audit

import (
	"context"
	"mediahub/internal/logging"
	"mediahub/internal/services"

	"github.com/sirupsen/logrus"
)

// Ensure LoggerAuditor implements services.Auditor
var _ services.Auditor = (*LoggerAuditor)(nil)

// LoggerAuditor is a simple implementation of Auditor that writes to the standard application log.
type LoggerAuditor struct {
	enabled bool
}

// NewLoggerAuditor creates a new instance of LoggerAuditor.
func NewLoggerAuditor(enabled bool) *LoggerAuditor {
	return &LoggerAuditor{enabled: enabled}
}

// Log records an event using logrus if auditing is enabled.
func (a *LoggerAuditor) Log(ctx context.Context, action string, actor string, resource string, details map[string]interface{}) {
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
	logging.Log.WithFields(fields).Info("AUDIT EVENT")
}
