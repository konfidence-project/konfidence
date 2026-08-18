package log

import (
	"fmt"
	"log/slog"
)

var (
	Debug = slog.Debug
	Info  = slog.Info
	Error = slog.Error
)

// newLogger creates a new Logger with the given non-nil Handler.
func newLogger(handler slog.Handler) *slog.Logger {
	return slog.New(handler)
}

// InitLogger initializes the logger variable.
func InitLogger(handler slog.Handler) {
	slog.SetDefault(newLogger(handler))
}

func Debugf(format string, args ...any) { logf(Debug, format, args...) }
func Infof(format string, args ...any)  { logf(Info, format, args...) }
func Errorf(format string, args ...any) { logf(Error, format, args...) }

// logf is a helper to format messages and call the appropriate slog function
func logf(logFunc func(msg string, args ...any), format string, args ...any) {
	logFunc(fmt.Sprintf(format, args...))
}
