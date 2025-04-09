package logger

import "context"

type Interface interface {
	DebugContext(ctx context.Context, message string, fields ...any)
	InfoContext(ctx context.Context, message string, fields ...any)
	WarnContext(ctx context.Context, message string, fields ...any)
	ErrorContext(ctx context.Context, message string, fields ...any)
}
