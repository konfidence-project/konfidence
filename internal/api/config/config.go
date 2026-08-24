package config

import (
	"fmt"
	"maps"
	"net/url"
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
	UIAssetPath     string
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
	AllowReturnURLs  []string
	PKCEEnabled      bool
	StateExpiration  string
}

type SessionConfig struct {
	StorageType     string
	Cookie          SessionCookieConfig
	Expiration      string
	CleanupInterval string
}

type SessionCookieConfig struct {
	Name     string
	HTTPOnly bool
	Secure   bool
	SameSite string
}

type DatabaseConfig struct {
	Connection      string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime string
	MaxConnIdleTime string
}

type KubernetesConfig struct {
	Scheme *runtime.Scheme
}

// Parsed holds validated, pre-parsed values ready for use by the server.
type Parsed struct {
	Server     ParsedServerConfig
	OIDC       ParsedOIDCConfig
	Session    ParsedSessionConfig
	Database   ParsedDatabaseConfig
	Kubernetes KubernetesConfig
}

type ParsedServerConfig struct {
	Addr            string
	LogLevel        string
	UIAssetPath     string
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
	AllowReturnURLs  []string
	PKCEEnabled      bool
	StateExpiration  time.Duration
}

type ParsedSessionConfig struct {
	StorageType     string
	Cookie          SessionCookieConfig
	Expiration      time.Duration
	CleanupInterval time.Duration
}

type ParsedDatabaseConfig struct {
	Connection      string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// Validate checks all fields, parses durations, and returns a ready-to-use
// Parsed on success.
func (c Config) Validate() (Parsed, error) {
	parsedServer, err := c.Server.validate()
	if err != nil {
		return Parsed{}, err
	}
	parsedOidcConfig, err := c.OIDC.validate()
	if err != nil {
		return Parsed{}, err
	}
	parsedSessionConfig, err := c.Session.validate()
	if err != nil {
		return Parsed{}, err
	}
	parsedDatabase, err := c.Database.validate(parsedSessionConfig.StorageType)
	if err != nil {
		return Parsed{}, err
	}

	return Parsed{
		Server:     parsedServer,
		OIDC:       parsedOidcConfig,
		Session:    parsedSessionConfig,
		Database:   parsedDatabase,
		Kubernetes: c.Kubernetes,
	}, nil
}

func (c ServerConfig) validate() (ParsedServerConfig, error) {
	if c.Addr == "" {
		return ParsedServerConfig{}, fmt.Errorf("addr must not be empty")
	}
	read, err := time.ParseDuration(c.ReadTimeout)
	if err != nil {
		return ParsedServerConfig{}, fmt.Errorf("invalid read-timeout %q: %w", c.ReadTimeout, err)
	}
	write, err := time.ParseDuration(c.WriteTimeout)
	if err != nil {
		return ParsedServerConfig{}, fmt.Errorf("invalid write-timeout %q: %w", c.WriteTimeout, err)
	}
	shutdown, err := time.ParseDuration(c.ShutdownTimeout)
	if err != nil {
		return ParsedServerConfig{}, fmt.Errorf("invalid shutdown-timeout %q: %w", c.ShutdownTimeout, err)
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return ParsedServerConfig{}, fmt.Errorf("invalid log-level %q: must be one of debug, info, warn, error", c.LogLevel)
	}
	return ParsedServerConfig{
		Addr:            c.Addr,
		LogLevel:        c.LogLevel,
		UIAssetPath:     c.UIAssetPath,
		ReadTimeout:     read,
		WriteTimeout:    write,
		ShutdownTimeout: shutdown,
	}, nil
}

func (c OIDCConfig) validate() (ParsedOIDCConfig, error) {
	stateExpiration, err := time.ParseDuration(c.StateExpiration)
	if err != nil {
		return ParsedOIDCConfig{}, fmt.Errorf("invalid oidc-state-expiration %q: %w", c.StateExpiration, err)
	}
	if c.Enabled {
		if c.IssuerURL == "" {
			return ParsedOIDCConfig{}, fmt.Errorf("oidc-issuer-url must not be empty")
		}
		if c.ClientID == "" {
			return ParsedOIDCConfig{}, fmt.Errorf("oidc-client-id must not be empty")
		}
		if c.ClientSecret == "" {
			return ParsedOIDCConfig{}, fmt.Errorf("oidc-client-secret must not be empty")
		}
		if c.RedirectURL == "" {
			return ParsedOIDCConfig{}, fmt.Errorf("oidc-redirect-url must not be empty")
		}
		if len(c.AllowReturnURLs) == 0 {
			return ParsedOIDCConfig{}, fmt.Errorf("oidc-allow-return-url must not be empty")
		}
	}

	allowReturnURLs := make([]string, 0, len(c.AllowReturnURLs))
	for _, returnURL := range c.AllowReturnURLs {
		returnURL = strings.TrimSpace(returnURL)
		parsedURL, err := url.Parse(returnURL)
		if err != nil || !parsedURL.IsAbs() ||
			(parsedURL.Scheme != "http" && parsedURL.Scheme != "https") ||
			parsedURL.Hostname() == "" || parsedURL.User != nil {
			return ParsedOIDCConfig{}, fmt.Errorf("invalid oidc-allow-return-url %q: must be a fully qualified HTTP(S) URL", returnURL)
		}
		allowReturnURLs = append(allowReturnURLs, returnURL)
	}

	return ParsedOIDCConfig{
		Enabled: c.Enabled, IssuerURL: c.IssuerURL, TokenURL: c.TokenURL,
		AuthorizationURL: c.AuthorizationURL, DeviceAuthURL: c.DeviceAuthURL,
		UserInfoURL: c.UserInfoURL, JWKSURL: c.JWKSURL, ClientID: c.ClientID,
		ClientSecret: c.ClientSecret, Scopes: mergeScopes(parseCommaSeparatedList(c.Scopes)),
		RedirectURL: c.RedirectURL, AllowReturnURLs: allowReturnURLs,
		PKCEEnabled: c.PKCEEnabled, StateExpiration: stateExpiration,
	}, nil
}

func (c SessionConfig) validate() (ParsedSessionConfig, error) {
	expiration, err := time.ParseDuration(c.Expiration)
	if err != nil {
		return ParsedSessionConfig{}, fmt.Errorf("invalid session-expiration %q: %w", c.Expiration, err)
	}

	cleanupInterval, err := time.ParseDuration(c.CleanupInterval)
	if err != nil {
		return ParsedSessionConfig{}, fmt.Errorf(
			"invalid session-cleanup-interval %q: %w",
			c.CleanupInterval,
			err,
		)
	}
	if cleanupInterval <= 0 {
		return ParsedSessionConfig{}, fmt.Errorf(
			"session-cleanup-interval must be greater than zero",
		)
	}

	if err := c.Cookie.validate(); err != nil {
		return ParsedSessionConfig{}, err
	}
	switch c.StorageType {
	case "in-memory", "db-pg":
	default:
		return ParsedSessionConfig{}, fmt.Errorf("invalid session-storage-type %q: must be one of in-memory, db-pg", c.StorageType)
	}
	return ParsedSessionConfig{StorageType: c.StorageType, Cookie: c.Cookie, Expiration: expiration, CleanupInterval: cleanupInterval}, nil
}

func (c SessionCookieConfig) validate() error {
	if c.Name == "" {
		return fmt.Errorf("session-cookie-name must not be empty")
	}
	return nil
}

func (c DatabaseConfig) validate(storageType string) (ParsedDatabaseConfig, error) {
	parsed := ParsedDatabaseConfig{Connection: c.Connection, MaxConns: c.MaxConns, MinConns: c.MinConns}
	if storageType != "db-pg" {
		return parsed, nil
	}
	if c.Connection == "" {
		return ParsedDatabaseConfig{}, fmt.Errorf("db-connection must not be empty when session-storage-type is db-pg")
	}
	if c.MaxConns <= 0 {
		return ParsedDatabaseConfig{}, fmt.Errorf("db-max-conns must be greater than zero")
	}
	if c.MinConns < 0 {
		return ParsedDatabaseConfig{}, fmt.Errorf("db-min-conns must not be negative")
	}
	if c.MinConns > c.MaxConns {
		return ParsedDatabaseConfig{}, fmt.Errorf("db-min-conns must not exceed db-max-conns")
	}
	maxConnLifetime, err := time.ParseDuration(c.MaxConnLifetime)
	if err != nil {
		return ParsedDatabaseConfig{}, fmt.Errorf("invalid db-max-conn-lifetime %q: %w", c.MaxConnLifetime, err)
	}
	maxConnIdleTime, err := time.ParseDuration(c.MaxConnIdleTime)
	if err != nil {
		return ParsedDatabaseConfig{}, fmt.Errorf("invalid db-max-conn-idle-time %q: %w", c.MaxConnIdleTime, err)
	}
	parsed.MaxConnLifetime = maxConnLifetime
	parsed.MaxConnIdleTime = maxConnIdleTime
	return parsed, nil
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
