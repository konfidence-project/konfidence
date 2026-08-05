package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/konfidence-project/konfidence/internal/api/handler"
	"github.com/konfidence-project/konfidence/internal/api/oidc"
	"github.com/konfidence-project/konfidence/internal/api/session"
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
  API_AUTH_ISSUER_URL    External IDP issuer URL            (default: http://localhost:5556/dex)
  API_AUTH_CLIENT_ID     External IDP client ID             (default: konfidence)
  API_AUTH_REDIRECT_URL  OAuth redirect URL                 (default: http://localhost:8090/api/v1/auth/callback)

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
	rootCmd.Flags().StringVar(&cfg.AuthIssuerURL, "auth-issuer-url", envOr("API_AUTH_ISSUER_URL", "http://localhost:5556/dex"),
		"External identity provider issuer URL. Env: API_AUTH_ISSUER_URL")
	rootCmd.Flags().StringVar(&cfg.AuthClientID, "auth-client-id", envOr("API_AUTH_CLIENT_ID", "konfidence"),
		"External identity provider client ID. Env: API_AUTH_CLIENT_ID")
	rootCmd.Flags().StringVar(&cfg.AuthRedirectURL, "auth-redirect-url", envOr("API_AUTH_REDIRECT_URL", "http://localhost:8090/api/v1/auth/callback"),
		"OAuth redirect URL for the API authentication flow. Env: API_AUTH_REDIRECT_URL")
}

func startServer(cmd *cobra.Command, _ []string) error {
	cfg.Scheme = scheme
	parsed, err := cfg.Validate()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	level := resolveLogLevel(cfg.LogLevel)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	k8sConfig, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get k8s config (set KUBECONFIG for local dev): %w", err)
	}

	k8sClient, err := client.New(k8sConfig, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("failed to build k8s client: %w", err)
	}

	// TODO add PKCEEnabled to config
	oidcConfig := oidc.Config{
		IdentityProviderURI: cfg.AuthIssuerURL,
		RedirectURI:         cfg.AuthRedirectURL,
		ClientID:            cfg.AuthClientID,
		PKCEEnabled:         true,
	}

	oidcClient := oidc.NewOIDCClient(oidcConfig)
	if err := oidcClient.Setup(ctx); err != nil {
		return fmt.Errorf("failed to create oidc client: %w", err)
	}

	stateStore := oidc.NewStateCacheStore(parsed)
	sessionStore := session.NewSessionCacheStore(parsed)

	serverHandler, err := handler.NewServerHandler(logger, k8sClient, *oidcClient, stateStore, sessionStore)
	if err != nil {
		return fmt.Errorf("failed to create server handler: %w", err)
	}

	return server.New(parsed, logger, serverHandler.Mount).Run(ctx)
}

// envOr returns the value of the environment variable key, or fallback if unset.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func resolveLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
