package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/konfidence-project/konfidence/cmd/api/db/sqlc"
	"github.com/konfidence-project/konfidence/internal/api/handler"
	"github.com/konfidence-project/konfidence/internal/api/oidc"
	"github.com/konfidence-project/konfidence/internal/api/session"
	"github.com/konfidence-project/konfidence/internal/kden/log"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"

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
by environment variables.

	Kubernetes client config is resolved automatically via the standard KUBECONFIG
	env var or in-cluster config when deployed as a pod.`,
	PreRun: loadOIDCClientSecret,
	RunE:   startServer,
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

	rootCmd.Flags().StringVar(&cfg.Server.Addr, "addr", envOr("API_ADDR", ":8090"),
		"TCP address the API server listens on. Env: API_ADDR")
	rootCmd.Flags().StringVar(&cfg.Server.LogLevel, "log-level", envOr("API_LOG_LEVEL", "info"),
		"Log level (debug, info, warn, error). Env: API_LOG_LEVEL")
	rootCmd.Flags().StringVar(&cfg.Server.UIAssetPath, "ui-asset-path", envOr("API_UI_ASSET_PATH", ""),
		"Path to the directory containing dashboard UI assets. Env: API_UI_ASSET_PATH")
	rootCmd.Flags().StringVar(&cfg.Server.ReadTimeout, "read-timeout", envOr("API_READ_TIMEOUT", "10s"),
		"Maximum duration for reading an entire request. Env: API_READ_TIMEOUT")
	rootCmd.Flags().StringVar(&cfg.Server.WriteTimeout, "write-timeout", envOr("API_WRITE_TIMEOUT", "10s"),
		"Maximum duration before timing out writes of the response. Env: API_WRITE_TIMEOUT")
	rootCmd.Flags().StringVar(&cfg.Server.ShutdownTimeout, "shutdown-timeout", envOr("API_SHUTDOWN_TIMEOUT", "15s"),
		"Maximum duration for a graceful shutdown. Env: API_SHUTDOWN_TIMEOUT")
	rootCmd.Flags().BoolVar(&cfg.OIDC.Enabled, "oidc-enabled", envBoolOr("API_OIDC_ENABLED", true),
		"Enable OIDC authentication. When false, all requests run as an admin user. Env: API_OIDC_ENABLED")
	rootCmd.Flags().StringVar(&cfg.OIDC.IssuerURL, "oidc-issuer-url", envOr("API_OIDC_ISSUER_URL", ""),
		"External identity provider issuer URL. Env: API_OIDC_ISSUER_URL")
	rootCmd.Flags().StringVar(&cfg.OIDC.TokenURL, "oidc-token-url", envOr("API_OIDC_TOKEN_URL", ""),
		"External identity provider token URL. Env: API_OIDC_TOKEN_URL")
	rootCmd.Flags().StringVar(&cfg.OIDC.AuthorizationURL, "oidc-authorization-url", envOr("API_OIDC_AUTHORIZATION_URL", ""),
		"External identity provider authorization URL. Env: API_OIDC_AUTHORIZATION_URL")
	rootCmd.Flags().StringVar(&cfg.OIDC.DeviceAuthURL, "oidc-device-auth-url", envOr("API_OIDC_DEVICE_AUTH_URL", ""),
		"External identity provider device authorization URL. Env: API_OIDC_DEVICE_AUTH_URL")
	rootCmd.Flags().StringVar(&cfg.OIDC.UserInfoURL, "oidc-user-info-url", envOr("API_OIDC_USER_INFO_URL", ""),
		"External identity provider user info URL. Env: API_OIDC_USER_INFO_URL")
	rootCmd.Flags().StringVar(&cfg.OIDC.JWKSURL, "oidc-jwks-url", envOr("API_OIDC_JWKS_URL", ""),
		"External identity provider JWKS URL. Env: API_OIDC_JWKS_URL")
	rootCmd.Flags().StringVar(&cfg.OIDC.ClientID, "oidc-client-id", envOr("API_OIDC_CLIENT_ID", ""),
		"External identity provider client ID. Env: API_OIDC_CLIENT_ID")
	rootCmd.Flags().StringVar(&cfg.OIDC.ClientSecret, "oidc-client-secret", "",
		"External identity provider client Secret. Env: API_OIDC_CLIENT_SECRET")
	rootCmd.Flags().StringVar(&cfg.OIDC.Scopes, "oidc-scopes", envOr("API_OIDC_SCOPES", ""),
		"External identity provider client scopes. Env: API_OIDC_SCOPES")
	rootCmd.Flags().StringVar(&cfg.OIDC.RedirectURL, "oidc-redirect-url", envOr("API_OIDC_REDIRECT_URL", ""),
		"OAuth redirect URL for the API authentication flow. Env: API_OIDC_REDIRECT_URL")
	rootCmd.Flags().StringSliceVar(&cfg.OIDC.AllowReturnURLs, "oidc-allow-return-urls", envList("API_OIDC_ALLOW_RETURN_URLS"),
		"Fully qualified return URLs allowed after login. Env: API_OIDC_ALLOW_RETURN_URLS")
	rootCmd.Flags().BoolVar(&cfg.OIDC.PKCEEnabled, "oidc-pkce-enabled", envBoolOr("API_OIDC_PKCE_ENABLED", true),
		"Enable PKCE for the OIDC authentication flow. Env: API_OIDC_PKCE_ENABLED")
	rootCmd.Flags().StringVar(&cfg.OIDC.StateExpiration, "oidc-state-expiration", envOr("API_OIDC_STATE_EXPIRATION", "15m"),
		"OIDC state cache expiration duration. Env: API_OIDC_STATE_EXPIRATION")
	rootCmd.Flags().StringVar(&cfg.Session.Cookie.Name, "session-cookie-name", envOr("API_SESSION_COOKIE_NAME", "kden-session"),
		"Session cookie name. Env: API_SESSION_COOKIE_NAME")
	rootCmd.Flags().BoolVar(&cfg.Session.Cookie.HTTPOnly, "session-cookie-http-only", envBoolOr("API_SESSION_COOKIE_HTTP_ONLY", true),
		"Set HttpOnly on the session cookie. Env: API_SESSION_COOKIE_HTTP_ONLY")
	rootCmd.Flags().BoolVar(&cfg.Session.Cookie.Secure, "session-cookie-secure", envBoolOr("API_SESSION_COOKIE_SECURE", false),
		"Set Secure on the session cookie. Env: API_SESSION_COOKIE_SECURE")
	rootCmd.Flags().StringVar(&cfg.Session.Cookie.SameSite, "session-cookie-same-site", envOr("API_SESSION_COOKIE_SAME_SITE", "SameSiteStrictMode"),
		"Session cookie SameSite mode. Env: API_SESSION_COOKIE_SAME_SITE")
	rootCmd.Flags().StringVar(&cfg.Session.Expiry, "session-expiry", envOr("API_SESSION_EXPIRY", "12h"),
		"Server-side session expiry duration. Env: API_SESSION_EXPIRY")
	rootCmd.Flags().StringVar(&cfg.Session.StorageType, "session-storage-type", envOr("API_SESSION_STORAGE_TYPE", "in-memory"),
		"Session storage backend (in-memory, db-pg). Env: API_SESSION_STORAGE_TYPE")
	rootCmd.Flags().StringVar(&cfg.Session.CleanupInterval, "session-cleanup-interval", envOr("API_SESSION_CLEANUP_INTERVAL", "15m"),
		"Expired database session cleanup interval. Env: API_SESSION_CLEANUP_INTERVAL",
	)
	rootCmd.Flags().StringVar(&cfg.Database.Connection, "db-connection", envOr("API_DB_CONNECTION", ""),
		"API DB connection string. Env: API_DB_CONNECTION")
	rootCmd.Flags().Int32Var(&cfg.Database.MaxConns, "db-max-conns", envInt32Or("API_DB_MAX_CONNS", 10),
		"Maximum number of database pool connections. Env: API_DB_MAX_CONNS")
	rootCmd.Flags().Int32Var(&cfg.Database.MinConns, "db-min-conns", envInt32Or("API_DB_MIN_CONNS", 5),
		"Minimum number of database pool connections. Env: API_DB_MIN_CONNS")
	rootCmd.Flags().StringVar(&cfg.Database.MaxConnLifetime, "db-max-conn-lifetime", envOr("API_DB_MAX_CONN_LIFETIME", "30m"),
		"Maximum lifetime of a database pool connection. Env: API_DB_MAX_CONN_LIFETIME")
	rootCmd.Flags().StringVar(&cfg.Database.MaxConnIdleTime, "db-max-conn-idle-time", envOr("API_DB_MAX_CONN_IDLE_TIME", "5m"),
		"Maximum idle time of a database pool connection. Env: API_DB_MAX_CONN_IDLE_TIME")
}

func loadOIDCClientSecret(cmd *cobra.Command, _ []string) {
	if !cmd.Flags().Changed("oidc-client-secret") {
		cfg.OIDC.ClientSecret = os.Getenv("API_OIDC_CLIENT_SECRET")
	}
}

func startServer(cmd *cobra.Command, _ []string) error {
	cfg.Kubernetes.Scheme = scheme
	parsed, err := cfg.Validate()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logHandler, err := log.ResolveLogHandler(cfg.Server.LogLevel, "json")
	if err != nil {
		return fmt.Errorf("failed to initialize log handler: %w", err)
	}

	logger := slog.New(logHandler)

	k8sConfig, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get k8s config (set KUBECONFIG for local dev): %w", err)
	}

	k8sClient, err := newInformerBackedClient(ctx, k8sConfig, scheme,
		&konfidence.Project{},
		&konfidence.Landscape{},
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := k8sClient.Close(); err != nil {
			logger.Error("informer-backed Kubernetes client stopped with an error", "error", err)
		}
	}()
	oidcClient := oidc.NewOIDCClient(oidc.Config{})
	if parsed.OIDC.Enabled {
		oidcClient = oidc.NewOIDCClient(oidc.Config{
			IdentityProviderURI: parsed.OIDC.IssuerURL,
			TokenURL:            parsed.OIDC.TokenURL,
			AuthorizationURL:    parsed.OIDC.AuthorizationURL,
			DeviceAuthURL:       parsed.OIDC.DeviceAuthURL,
			RedirectURI:         parsed.OIDC.RedirectURL,
			UserInfoURL:         parsed.OIDC.UserInfoURL,
			JWKSURL:             parsed.OIDC.JWKSURL,
			ClientID:            parsed.OIDC.ClientID,
			ClientSecret:        parsed.OIDC.ClientSecret,
			Scopes:              parsed.OIDC.Scopes,
			PKCEEnabled:         parsed.OIDC.PKCEEnabled,
		})
		if err := oidcClient.Setup(ctx); err != nil {
			return fmt.Errorf("failed to create oidc client: %w", err)
		}
	}

	var sessionStore session.Store
	switch parsed.Session.StorageType {
	case "db-pg":
		// init database config
		dbConfig, err := pgxpool.ParseConfig(parsed.Database.Connection)
		if err != nil {
			return fmt.Errorf("unable to parse connection string: %w", err)
		}

		dbConfig.MaxConns = parsed.Database.MaxConns
		dbConfig.MinConns = parsed.Database.MinConns
		dbConfig.MaxConnLifetime = parsed.Database.MaxConnLifetime
		dbConfig.MaxConnIdleTime = parsed.Database.MaxConnIdleTime

		dbPool, err = pgxpool.NewWithConfig(ctx, dbConfig)
		if err != nil {
			return fmt.Errorf("unable to create connection pool: %w", err)
		}
		defer dbPool.Close()

		if err := dbPool.Ping(ctx); err != nil {
			return fmt.Errorf("database unreachable: %w", err)
		}

		queries := db.New(dbPool)
		dbStore := session.NewDBStore(queries, parsed.Session.Expiry)
		sessionStore = dbStore
		cleanupCtx, cancelCleanup := context.WithCancel(ctx)
		cleanupDone := make(chan struct{})
		go func() {
			defer close(cleanupDone)
			dbStore.RunCleanup(
				cleanupCtx,
				logger,
				parsed.Session.CleanupInterval,
			)
		}()

		defer func() {
			cancelCleanup()
			<-cleanupDone
		}()
	case "in-memory":
		sessionStore = session.NewInMemoryStore(parsed)
	}

	api, err := handler.NewAPIHandler(logger, k8sClient, *oidcClient, sessionStore, parsed)
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

func envInt32Or(key string, fallback int32) int32 {
	if v := os.Getenv(key); v != "" {
		parsed, err := strconv.ParseInt(v, 10, 32)
		if err == nil {
			return int32(parsed)
		}
	}
	return fallback
}

func envList(key string) []string {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}
	return strings.Split(value, ",")
}
