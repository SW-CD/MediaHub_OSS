package audit

import "context"

type AlNoopLogger struct{}

func NewAlNoopLogger() *AlNoopLogger {
	return &AlNoopLogger{}
}

func (a *AlNoopLogger) Log(ctx context.Context, action string, actor string, resource string, details map[string]any) {
	// noop
}
