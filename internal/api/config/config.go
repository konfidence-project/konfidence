package config

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
)

// Config holds all runtime configuration for the API server.
// Values are populated from CLI flags (see cmd/api/cmd/root.go).
// The flag → env → default precedence is handled at the cobra layer.
type Config struct {
	Addr            string
	ReadTimeout     string
	WriteTimeout    string
	ShutdownTimeout string
	LogLevel        string
	// Scheme holds the registered API types. Built in cmd/api/cmd/root.go
	// following the same pattern as the galaxy and star operators.
	Scheme *runtime.Scheme
}

// Parsed holds pre-parsed duration values from Config.
type Parsed struct {
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// Validate checks all fields and returns the first error found.
func (c Config) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("addr must not be empty")
	}
	if _, err := time.ParseDuration(c.ReadTimeout); err != nil {
		return fmt.Errorf("invalid read-timeout %q: %w", c.ReadTimeout, err)
	}
	if _, err := time.ParseDuration(c.WriteTimeout); err != nil {
		return fmt.Errorf("invalid write-timeout %q: %w", c.WriteTimeout, err)
	}
	if _, err := time.ParseDuration(c.ShutdownTimeout); err != nil {
		return fmt.Errorf("invalid shutdown-timeout %q: %w", c.ShutdownTimeout, err)
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("invalid log-level %q: must be one of debug, info, warn, error", c.LogLevel)
	}
	return nil
}

// Parse returns the pre-parsed durations. Call Validate first to ensure the
// string fields are well-formed; invalid values produce zero durations.
func (c Config) Parse() Parsed {
	read, _ := time.ParseDuration(c.ReadTimeout)
	write, _ := time.ParseDuration(c.WriteTimeout)
	shutdown, _ := time.ParseDuration(c.ShutdownTimeout)
	return Parsed{
		ReadTimeout:     read,
		WriteTimeout:    write,
		ShutdownTimeout: shutdown,
	}
}
