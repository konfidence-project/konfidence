package config

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/coreos/go-oidc/v3/oidc"
	"k8s.io/apimachinery/pkg/runtime"
)

// Config holds all runtime configuration for the API server.
type Config struct {
	Server     ServerConfig
	OIDC       OIDCConfig
	Session    SessionConfig
	Database   DatabaseConfig
	Kubernetes KubernetesConfig
}

type ServerConfig struct {
	Addr            string
	ReadTimeout     string
	WriteTimeout    string
	ShutdownTimeout string
	LogLevel        string
}

type OIDCConfig struct {
	Enabled          bool
	IssuerURL        string
	TokenURL         string
	AuthorizationURL string
	DeviceAuthURL    string
	UserInfoURL      string
	JWKSURL          string
	ClientID         string
	ClientSecret     string
	Scopes           string
	RedirectURL      string
	PKCEEnabled      bool
	StateExpiration  string
}

type SessionConfig struct {
	Cookie SessionCookieConfig
	Expiry string
}

type SessionCookieConfig struct {
	Name     string
	HTTPOnly bool
	Secure   bool
	SameSite string
}

type DatabaseConfig struct {
	Connection string
}

type KubernetesConfig struct {
	Scheme *runtime.Scheme
}

// Parsed holds validated, pre-parsed values ready for use by the server.
type Parsed struct {
	Server     ParsedServerConfig
	OIDC       ParsedOIDCConfig
	Session    ParsedSessionConfig
	Database   DatabaseConfig
	Kubernetes KubernetesConfig
}

type ParsedServerConfig struct {
	Addr            string
	LogLevel        string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type ParsedOIDCConfig struct {
	Enabled          bool
	IssuerURL        string
	TokenURL         string
	AuthorizationURL string
	DeviceAuthURL    string
	UserInfoURL      string
	JWKSURL          string
	ClientID         string
	ClientSecret     string
	Scopes           []string
	RedirectURL      string
	PKCEEnabled      bool
	StateExpiration  time.Duration
}

type ParsedSessionConfig struct {
	Cookie SessionCookieConfig
	Expiry time.Duration
}

// Validate checks all fields, parses durations, and returns a ready-to-use
// Parsed on success.
func (c Config) Validate() (Parsed, error) {
	if c.Server.Addr == "" {
		return Parsed{}, fmt.Errorf("addr must not be empty")
	}
	read, err := time.ParseDuration(c.Server.ReadTimeout)
	if err != nil {
		return Parsed{}, fmt.Errorf("invalid read-timeout %q: %w", c.Server.ReadTimeout, err)
	}
	write, err := time.ParseDuration(c.Server.WriteTimeout)
	if err != nil {
		return Parsed{}, fmt.Errorf("invalid write-timeout %q: %w", c.Server.WriteTimeout, err)
	}
	shutdown, err := time.ParseDuration(c.Server.ShutdownTimeout)
	if err != nil {
		return Parsed{}, fmt.Errorf("invalid shutdown-timeout %q: %w", c.Server.ShutdownTimeout, err)
	}
	sessionExpiry, err := time.ParseDuration(c.Session.Expiry)
	if err != nil {
		return Parsed{}, fmt.Errorf("invalid session-expiry %q: %w", c.Session.Expiry, err)
	}
	oidcStateExpiration, err := time.ParseDuration(c.OIDC.StateExpiration)
	if err != nil {
		return Parsed{}, fmt.Errorf("invalid oidc-state-expiration %q: %w", c.OIDC.StateExpiration, err)
	}
	switch c.Server.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Parsed{}, fmt.Errorf("invalid log-level %q: must be one of debug, info, warn, error", c.Server.LogLevel)
	}
	if c.OIDC.Enabled {
		if c.OIDC.IssuerURL == "" {
			return Parsed{}, fmt.Errorf("oidc-issuer-url must not be empty")
		}
		if c.OIDC.ClientID == "" {
			return Parsed{}, fmt.Errorf("oidc-client-id must not be empty")
		}
		if c.OIDC.ClientSecret == "" {
			return Parsed{}, fmt.Errorf("oidc-client-secret must not be empty")
		}
		if c.OIDC.RedirectURL == "" {
			return Parsed{}, fmt.Errorf("oidc-redirect-url must not be empty")
		}
	}
	if c.Session.Cookie.Name == "" {
		return Parsed{}, fmt.Errorf("session-cookie-name must not be empty")
	}

	scopes := mergeScopes(parseCommaSeparatedList(c.OIDC.Scopes))
	return Parsed{
		Server: ParsedServerConfig{
			Addr:            c.Server.Addr,
			LogLevel:        c.Server.LogLevel,
			ReadTimeout:     read,
			WriteTimeout:    write,
			ShutdownTimeout: shutdown,
		},
		OIDC: ParsedOIDCConfig{
			Enabled:          c.OIDC.Enabled,
			IssuerURL:        c.OIDC.IssuerURL,
			TokenURL:         c.OIDC.TokenURL,
			AuthorizationURL: c.OIDC.AuthorizationURL,
			DeviceAuthURL:    c.OIDC.DeviceAuthURL,
			UserInfoURL:      c.OIDC.UserInfoURL,
			JWKSURL:          c.OIDC.JWKSURL,
			ClientID:         c.OIDC.ClientID,
			ClientSecret:     c.OIDC.ClientSecret,
			Scopes:           scopes,
			RedirectURL:      c.OIDC.RedirectURL,
			PKCEEnabled:      c.OIDC.PKCEEnabled,
			StateExpiration:  oidcStateExpiration,
		},
		Session: ParsedSessionConfig{
			Cookie: c.Session.Cookie,
			Expiry: sessionExpiry,
		},
		Database:   c.Database,
		Kubernetes: c.Kubernetes,
	}, nil
}

func parseCommaSeparatedList(csl string) []string {
	return strings.FieldsFunc(csl, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
}

func mergeScopes(scopes []string) []string {
	// default scopes
	scopeMap := map[string]bool{
		oidc.ScopeOpenID: true, "profile": true,
	}

	for _, scope := range scopes {
		scopeMap[scope] = true
	}

	return slices.Collect(maps.Keys(scopeMap))
}
