package config

import (
	"fmt"
	"strings"
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
	// Kubeconfig is the path to a kubeconfig file for local development.
	// Empty means in-cluster config, which is used when deployed as a pod.
	// The k8s client is built lazily — omitting this does not prevent the
	// server from starting; only domain endpoints that talk to the cluster
	// will fail if neither kubeconfig nor in-cluster config is available
	Kubeconfig string
}

// AuthConfig holds OIDC configuration for the auth middleware.
type AuthConfig struct {
	AuthorizeURL string
	TokenURL     string
	UserInfoURL  string
	ClientID     string
	RedirectURI  string
	Scopes       string
}

// ScopesSlice splits the space-separated scopes string into a slice.
func (a AuthConfig) ScopesSlice() []string {
	if a.Scopes == "" {
		return nil
	}
	return strings.Fields(a.Scopes)
}

// Parsed holds validated, pre-parsed values ready for use by the server.
type Parsed struct {
	Addr            string
	LogLevel        string
	Auth            AuthConfig
	Scheme          *runtime.Scheme
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// Validate checks all fields, parses durations, and returns a ready-to-use Parsed on success.
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
		Auth:            c.Auth,
		Scheme:          c.Scheme,
		ReadTimeout:     read,
		WriteTimeout:    write,
		ShutdownTimeout: shutdown,
	}, nil
}
