package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/konfidence-project/konfidence/cmd/api/db/sqlc"
	"github.com/konfidence-project/konfidence/internal/api/handler"
	"github.com/konfidence-project/konfidence/internal/api/oidc"
	"github.com/konfidence-project/konfidence/internal/api/session"
	"github.com/konfidence-project/konfidence/internal/kden/log"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/server"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var scheme = runtime.NewScheme()

var cfg = config.Config{}

var dbPool *pgxpool.Pool

var rootCmd = &cobra.Command{
	Use:   "api",
	Short: "Run the Konfidence API server",
	Long: `api is the Konfidence API gateway.

It exposes an HTTP API consumed by the kden CLI, the Konfidence Dashboard,
and any external UI. Inbound requests are translated into reads and writes
against the Kubernetes API (Konfidence CRDs).

Configuration is read from environment variables (API_*) and can be overridden
by the flags below.

  API_ADDR               TCP address to listen on           (default: :8090)
  API_LOG_LEVEL          Log verbosity: debug/info/warn/error (default: info)
  API_READ_TIMEOUT       HTTP read deadline                 (default: 10s)
  API_WRITE_TIMEOUT      HTTP write deadline                (default: 10s)
  API_SHUTDOWN_TIMEOUT   Graceful shutdown window           (default: 15s)
  API_OIDC_ISSUER_URL    External IDP issuer URL
  API_OIDC_CLIENT_ID     External IDP client ID
  API_OIDC_CLIENT_SECRET External IDP client secret
  API_OIDC_SCOPES        External IDP client scopes
  API_OIDC_REDIRECT_URL  OAuth redirect URL
  API_OIDC_PKCE_ENABLED  Enable PKCE for OIDC auth flow     (default: true)
  API_OIDC_STATE_EXPIRATION OIDC state cache expiration      (default: 15m)
  API_SESSION_EXPIRY     Server-side session expiry          (default: 12h)
  API_DB_CONNECTION      Postgres DB Connection string

Kubernetes client config is resolved automatically via the standard KUBECONFIG
env var or in-cluster config when deployed as a pod.`,
	RunE: startServer,
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(konfidence.AddToScheme(scheme))

	rootCmd.Flags().StringVar(&cfg.Addr, "addr", envOr("API_ADDR", ":8090"),
		"TCP address the API server listens on. Env: API_ADDR")
	rootCmd.Flags().StringVar(&cfg.LogLevel, "log-level", envOr("API_LOG_LEVEL", "info"),
		"Log level (debug, info, warn, error). Env: API_LOG_LEVEL")
	rootCmd.Flags().StringVar(&cfg.ReadTimeout, "read-timeout", envOr("API_READ_TIMEOUT", "10s"),
		"Maximum duration for reading an entire request. Env: API_READ_TIMEOUT")
	rootCmd.Flags().StringVar(&cfg.WriteTimeout, "write-timeout", envOr("API_WRITE_TIMEOUT", "10s"),
		"Maximum duration before timing out writes of the response. Env: API_WRITE_TIMEOUT")
	rootCmd.Flags().StringVar(&cfg.ShutdownTimeout, "shutdown-timeout", envOr("API_SHUTDOWN_TIMEOUT", "15s"),
		"Maximum duration for a graceful shutdown. Env: API_SHUTDOWN_TIMEOUT")
	rootCmd.Flags().StringVar(&cfg.OIDCIssuerURL, "oidc-issuer-url", envOr("API_OIDC_ISSUER_URL", ""),
		"External identity provider issuer URL. Env: API_OIDC_ISSUER_URL")
	rootCmd.Flags().StringVar(&cfg.OIDCTokenURL, "oidc-token-url", envOr("API_OIDC_TOKEN_URL", ""),
		"External identity provider token URL. Env: API_OIDC_TOKEN_URL")
	rootCmd.Flags().StringVar(&cfg.OIDCAuthorizationURL, "oidc-authorization-url", envOr("API_OIDC_AUTHORIZATION_URL", ""),
		"External identity provider authorization URL. Env: API_OIDC_AUTHORIZATION_URL")
	rootCmd.Flags().StringVar(&cfg.OIDCDeviceAuthURL, "oidc-device-auth-url", envOr("API_OIDC_DEVICE_AUTH_URL", ""),
		"External identity provider device authorization URL. Env: API_OIDC_DEVICE_AUTH_URL")
	rootCmd.Flags().StringVar(&cfg.OIDCUserInfoURL, "oidc-user-info-url", envOr("API_OIDC_USER_INFO_URL", ""),
		"External identity provider user info URL. Env: API_OIDC_USER_INFO_URL")
	rootCmd.Flags().StringVar(&cfg.OIDCJWKSURL, "oidc-jwks-url", envOr("API_OIDC_JWKS_URL", ""),
		"External identity provider JWKS URL. Env: API_OIDC_JWKS_URL")
	rootCmd.Flags().StringVar(&cfg.OIDCClientID, "oidc-client-id", envOr("API_OIDC_CLIENT_ID", ""),
		"External identity provider client ID. Env: API_OIDC_CLIENT_ID")
	rootCmd.Flags().StringVar(&cfg.OIDCClientSecret, "oidc-client-secret", envOr("API_OIDC_CLIENT_SECRET", ""),
		"External identity provider client Secret. Env: API_OIDC_CLIENT_SECRET")
	rootCmd.Flags().StringVar(&cfg.OIDCScopes, "oidc-scopes", envOr("API_OIDC_SCOPES", ""),
		"External identity provider client scopes. Env: API_OIDC_SCOPES")
	rootCmd.Flags().StringVar(&cfg.OIDCRedirectURL, "oidc-redirect-url", envOr("API_OIDC_REDIRECT_URL", ""),
		"OAuth redirect URL for the API authentication flow. Env: API_OIDC_REDIRECT_URL")
	rootCmd.Flags().BoolVar(&cfg.OIDCPKCEEnabled, "oidc-pkce-enabled", envBoolOr("API_OIDC_PKCE_ENABLED", true),
		"Enable PKCE for the OIDC authentication flow. Env: API_OIDC_PKCE_ENABLED")
	rootCmd.Flags().StringVar(&cfg.OIDCStateExpiration, "oidc-state-expiration", envOr("API_OIDC_STATE_EXPIRATION", "15m"),
		"OIDC state cache expiration duration. Env: API_OIDC_STATE_EXPIRATION")
	rootCmd.Flags().StringVar(&cfg.SessionCookieName, "session-cookie-name", envOr("API_SESSION_COOKIE_NAME", "kden-session"),
		"Session cookie name. Env: API_SESSION_COOKIE_NAME")
	rootCmd.Flags().BoolVar(&cfg.SessionCookieHTTPOnly, "session-cookie-http-only", envBoolOr("API_SESSION_COOKIE_HTTP_ONLY", true),
		"Set HttpOnly on the session cookie. Env: API_SESSION_COOKIE_HTTP_ONLY")
	rootCmd.Flags().BoolVar(&cfg.SessionCookieSecure, "session-cookie-secure", envBoolOr("API_SESSION_COOKIE_SECURE", false),
		"Set Secure on the session cookie. Env: API_SESSION_COOKIE_SECURE")
	rootCmd.Flags().StringVar(&cfg.SessionCookieSameSite, "session-cookie-same-site", envOr("API_SESSION_COOKIE_SAME_SITE", "SameSiteStrictMode"),
		"Session cookie SameSite mode. Env: API_SESSION_COOKIE_SAME_SITE")
	rootCmd.Flags().StringVar(&cfg.SessionExpiry, "session-expiry", envOr("API_SESSION_EXPIRY", "12h"),
		"Server-side session expiry duration. Env: API_SESSION_EXPIRY")
	rootCmd.Flags().StringVar(&cfg.DBConnection, "db-connection", envOr("API_DB_CONNECTION", ""),
		"API DB connection string. Env: API_DB_CONNECTION")
}

func startServer(cmd *cobra.Command, _ []string) error {
	cfg.Scheme = scheme
	parsed, err := cfg.Validate()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logHandler, err := log.ResolveLogHandler(cfg.LogLevel, "json")
	if err != nil {
		return fmt.Errorf("failed to initialize log handler: %w", err)
	}

	logger := slog.New(logHandler)

	k8sConfig, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get k8s config (set KUBECONFIG for local dev): %w", err)
	}

	k8sClient, err := client.New(k8sConfig, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("failed to build k8s client: %w", err)
	}

	oidcConfig := oidc.Config{
		IdentityProviderURI: parsed.OIDCIssuerURL,
		TokenURL:            parsed.OIDCTokenURL,
		AuthorizationURL:    parsed.OIDCAuthorizationURL,
		DeviceAuthURL:       parsed.OIDCDeviceAuthURL,
		RedirectURI:         parsed.OIDCRedirectURL,
		ClientID:            parsed.OIDCClientID,
		ClientSecret:        parsed.OIDCClientSecret,
		Scopes:              parsed.OIDCScopes,
		PKCEEnabled:         parsed.OIDCPKCEEnabled,
	}

	oidcClient := oidc.NewOIDCClient(oidcConfig)
	if err := oidcClient.Setup(ctx); err != nil {
		return fmt.Errorf("failed to create oidc client: %w", err)
	}

	stateStore := oidc.NewStateCacheStore(parsed)

	var sessionStore session.SessionStore
	if parsed.DBConnection != "" {
		// init database config
		dbConfig, err := pgxpool.ParseConfig(parsed.DBConnection)
		if err != nil {
			return fmt.Errorf("unable to parse connection string: %w", err)
		}

		// TODO configure pool settings
		dbConfig.MaxConns = 10
		dbConfig.MinConns = 5
		dbConfig.MaxConnLifetime = 30 * time.Minute
		dbConfig.MaxConnIdleTime = 5 * time.Minute

		dbPool, err = pgxpool.NewWithConfig(ctx, dbConfig)
		if err != nil {
			return fmt.Errorf("unable to create connection pool: %w", err)
		}
		defer dbPool.Close()

		if err := dbPool.Ping(ctx); err != nil {
			return fmt.Errorf("database unreachable: %w", err)
		}
		queries := db.New(dbPool)
		sessionStore = session.NewDbSessionStore(*queries)
	} else {
		sessionStore = session.NewCacheSessionStore(parsed)
	}

	api, err := handler.NewAPIHandler(logger, k8sClient, *oidcClient, stateStore, sessionStore, parsed)
	if err != nil {
		return fmt.Errorf("failed to create API handler: %w", err)
	}

	return server.New(parsed, logger, api).ListenAndServe(ctx)
}

// envOr returns the value of the environment variable key, or fallback if unset.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBoolOr(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			return parsed
		}
	}
	return fallback
}
