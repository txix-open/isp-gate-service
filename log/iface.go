package log

import (
	"context"

	"go.uber.org/zap"
)

type Logger interface {
	Info(ctx context.Context, message interface{}, fields ...zap.Field)
	Error(ctx context.Context, message interface{}, fields ...zap.Field)
	Debug(ctx context.Context, message interface{}, fields ...zap.Field)
	Log(ctx context.Context, level Level, message interface{}, fields ...zap.Field)
}
