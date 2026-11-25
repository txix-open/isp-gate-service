package tests

import (
	"context"
	"fmt"
	"sync"

	"github.com/txix-open/isp-kit/log"
	"go.uber.org/zap/zapcore"
)

type capturedLogEntry struct {
	level   zapcore.Level
	message string
	fields  map[string]any
}

// logCapture ловит лог-записи в память. Zap-hook поля не получает,
// поэтому логгер реализован напрямую через интерфейс log.Logger.
type logCapture struct {
	mu      sync.Mutex
	entries []capturedLogEntry
}

func (c *logCapture) Error(_ context.Context, message any, fields ...log.Field) {
	c.log(zapcore.ErrorLevel, message, fields...)
}

func (c *logCapture) Warn(_ context.Context, message any, fields ...log.Field) {
	c.log(zapcore.WarnLevel, message, fields...)
}

func (c *logCapture) Info(_ context.Context, message any, fields ...log.Field) {
	c.log(zapcore.InfoLevel, message, fields...)
}

func (c *logCapture) Debug(_ context.Context, message any, fields ...log.Field) {
	c.log(zapcore.DebugLevel, message, fields...)
}

func (c *logCapture) log(level zapcore.Level, message any, fields ...log.Field) {
	encoder := zapcore.NewMapObjectEncoder()
	for _, field := range fields {
		field.AddTo(encoder)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = append(c.entries, capturedLogEntry{
		level:   level,
		message: captureMessage(message),
		fields:  encoder.Fields,
	})
}

func (c *logCapture) entriesWithMessage(message string) []capturedLogEntry {
	c.mu.Lock()
	defer c.mu.Unlock()

	found := make([]capturedLogEntry, 0, len(c.entries))
	for _, entry := range c.entries {
		if entry.message == message {
			found = append(found, entry)
		}
	}
	return found
}

func captureMessage(message any) string {
	if msg, ok := message.(string); ok {
		return msg
	}
	return fmt.Sprintf("%v", message)
}
