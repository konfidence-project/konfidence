package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

const (
	jsonLogFormat   = "json"
	textLogFormat   = "text"
	prettyLogFormat = "pretty"
)

var supportedLogLevels = []slog.Level{
	slog.LevelDebug,
	slog.LevelInfo,
	slog.LevelError,
}

// ResolveLogHandler returns the preferred type of logging handler.
func ResolveLogHandler(level, format string) (slog.Handler, error) {
	out := io.Writer(os.Stdout)
	logLevel, err := toLogLevel(level)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(format) {
	case jsonLogFormat:
		return slog.NewJSONHandler(out, &slog.HandlerOptions{Level: logLevel}), nil
	case textLogFormat:
		return slog.NewTextHandler(out, &slog.HandlerOptions{Level: logLevel}), nil
	case prettyLogFormat:
		return NewPrettyLogHandler(out, &PrettyOptions{Level: logLevel}), nil
	}
	return nil, fmt.Errorf("invalid log format provided: %s", format)
}

func toLogLevel(level string) (slog.Level, error) {
	for _, l := range supportedLogLevels {
		if strings.EqualFold(l.String(), level) {
			return l, nil
		}
	}
	return 0, fmt.Errorf("invalid log level provided: %s", level)
}
