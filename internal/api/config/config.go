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
	Addr                  string
	ReadTimeout           string
	WriteTimeout          string
	ShutdownTimeout       string
	LogLevel              string
	OIDCIssuerURL         string
	OIDCTokenURL          string
	OIDCAuthorizationURL  string
	OIDCDeviceAuthURL     string
	OIDCUserInfoURL       string
	OIDCJWKSURL           string
	OIDCClientID          string
	OIDCClientSecret      string
	OIDCScopes            string
	OIDCRedirectURL       string
	OIDCPKCEEnabled       bool
	OIDCStateExpiration   string
	SessionCookieName     string
	SessionCookieHTTPOnly bool
	SessionCookieSecure   bool
	SessionCookieSameSite string
	SessionExpiry         string
	DBConnection          string
	Scheme                *runtime.Scheme
}

// Parsed holds validated, pre-parsed values ready for use by the server.
type Parsed struct {
	Addr                  string
	LogLevel              string
	OIDCIssuerURL         string
	OIDCTokenURL          string
	OIDCAuthorizationURL  string
	OIDCDeviceAuthURL     string
	OIDCUserInfoURL       string
	OIDCJWKSURL           string
	OIDCClientID          string
	OIDCClientSecret      string
	OIDCScopes            []string
	OIDCRedirectURL       string
	OIDCPKCEEnabled       bool
	OIDCStateExpiration   time.Duration
	SessionCookieName     string
	SessionCookieHTTPOnly bool
	SessionCookieSecure   bool
	SessionCookieSameSite string
	SessionExpiry         time.Duration
	DBConnection          string
	Scheme                *runtime.Scheme
	ReadTimeout           time.Duration
	WriteTimeout          time.Duration
	ShutdownTimeout       time.Duration
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
	sessionExpiry, err := time.ParseDuration(c.SessionExpiry)
	if err != nil {
		return Parsed{}, fmt.Errorf("invalid session-expiry %q: %w", c.SessionExpiry, err)
	}
	oidcStateExpiration, err := time.ParseDuration(c.OIDCStateExpiration)
	if err != nil {
		return Parsed{}, fmt.Errorf("invalid oidc-state-expiration %q: %w", c.OIDCStateExpiration, err)
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Parsed{}, fmt.Errorf("invalid log-level %q: must be one of debug, info, warn, error", c.LogLevel)
	}
	if c.OIDCIssuerURL == "" {
		return Parsed{}, fmt.Errorf("oidc-issuer-url must not be empty")
	}
	if c.OIDCClientID == "" {
		return Parsed{}, fmt.Errorf("oidc-client-id must not be empty")
	}
	if c.OIDCClientSecret == "" {
		return Parsed{}, fmt.Errorf("oidc-client-secret must not be empty")
	}
	if c.OIDCRedirectURL == "" {
		return Parsed{}, fmt.Errorf("oidc-redirect-url must not be empty")
	}
	if c.SessionCookieName == "" {
		return Parsed{}, fmt.Errorf("session-cookie-name must not be empty")
	}

	scopes := c.mergeScopes(parseCommaSeparatedList(c.OIDCScopes))
	return Parsed{
		Addr:                  c.Addr,
		LogLevel:              c.LogLevel,
		OIDCIssuerURL:         c.OIDCIssuerURL,
		OIDCTokenURL:          c.OIDCTokenURL,
		OIDCAuthorizationURL:  c.OIDCAuthorizationURL,
		OIDCDeviceAuthURL:     c.OIDCDeviceAuthURL,
		OIDCUserInfoURL:       c.OIDCUserInfoURL,
		OIDCJWKSURL:           c.OIDCJWKSURL,
		OIDCClientID:          c.OIDCClientID,
		OIDCClientSecret:      c.OIDCClientSecret,
		OIDCScopes:            scopes,
		OIDCRedirectURL:       c.OIDCRedirectURL,
		OIDCPKCEEnabled:       c.OIDCPKCEEnabled,
		OIDCStateExpiration:   oidcStateExpiration,
		SessionCookieName:     c.SessionCookieName,
		SessionCookieHTTPOnly: c.SessionCookieHTTPOnly,
		SessionCookieSecure:   c.SessionCookieSecure,
		SessionCookieSameSite: c.SessionCookieSameSite,
		SessionExpiry:         sessionExpiry,
		DBConnection:          c.DBConnection,
		Scheme:                c.Scheme,
		ReadTimeout:           read,
		WriteTimeout:          write,
		ShutdownTimeout:       shutdown,
	}, nil
}

func parseCommaSeparatedList(csl string) []string {
	return strings.FieldsFunc(csl, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
}

func (c Config) mergeScopes(scopes []string) []string {
	// default scopes
	scopeMap := map[string]bool{
		oidc.ScopeOpenID: true, "profile": true, "groups": true, "offline_access": true,
	}

	for _, scope := range scopes {
		scopeMap[scope] = true
	}

	return slices.Collect(maps.Keys(scopeMap))
}
