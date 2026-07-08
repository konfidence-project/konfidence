package config

import (
	"fmt"
	"time"
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
	// Kubeconfig is the path to a kubeconfig file for local development.
	// Empty means in-cluster config, which is used when deployed as a pod.
	// The k8s client is built lazily — omitting this does not prevent the
	// server from starting; only domain endpoints that talk to the cluster
	// will fail if neither kubeconfig nor in-cluster config is available
	Kubeconfig string
}

// Parsed returns duration values already parsed from the string fields.
// It assumes Validate has been called first.
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
