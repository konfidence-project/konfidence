package config

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
)

// Config holds all runtime configuration for the API server.
type Config struct {
	Addr            string
	ReadTimeout     string
	WriteTimeout    string
	ShutdownTimeout string
	LogLevel        string
	Scheme          *runtime.Scheme
}

// Parsed holds validated, pre-parsed values ready for use by the server.
type Parsed struct {
	Addr            string
	LogLevel        string
	Scheme          *runtime.Scheme
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// Validate checks all fields, parses durations, and returns a ready-to-use
// Parsed on success.
func (c Config) Validate() (Parsed, error) {
	if c.Addr == "" {
		return Parsed{}, fmt.Errorf("addr must not be empty")
	}
	read, err := time.ParseDuration(c.ReadTimeout)
	if err != nil {
		return Parsed{}, fmt.Errorf("invalid read-timeout %q: %w", c.ReadTimeout, err)
	}
	write, err := time.ParseDuration(c.WriteTimeout)
	if err != nil {
		return Parsed{}, fmt.Errorf("invalid write-timeout %q: %w", c.WriteTimeout, err)
	}
	shutdown, err := time.ParseDuration(c.ShutdownTimeout)
	if err != nil {
		return Parsed{}, fmt.Errorf("invalid shutdown-timeout %q: %w", c.ShutdownTimeout, err)
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Parsed{}, fmt.Errorf("invalid log-level %q: must be one of debug, info, warn, error", c.LogLevel)
	}
	return Parsed{
		Addr:            c.Addr,
		LogLevel:        c.LogLevel,
		Scheme:          c.Scheme,
		ReadTimeout:     read,
		WriteTimeout:    write,
		ShutdownTimeout: shutdown,
	}, nil
}
