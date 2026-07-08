package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/server"
)

var rootCmd = &cobra.Command{
	Use:   "api",
	Short: "Run the Konfidence API server",
	Long: `api is the Konfidence API gateway.

It sits at the border of the star cluster and exposes an HTTP API consumed by
the kden CLI, the Star Dashboard, and any external UI. Inbound requests are
translated into reads and writes against the Kubernetes API (Konfidence CRDs).

Configuration is read from environment variables (API_*) and can be overridden
by the flags below. When deployed as a pod, environment variables are the
primary configuration mechanism. Flags are convenient for local development.

  API_ADDR               TCP address to listen on           (default: :8090)
  API_LOG_LEVEL          Log verbosity: debug/info/warn/error (default: info)
  API_READ_TIMEOUT       HTTP read deadline                 (default: 10s)
  API_WRITE_TIMEOUT      HTTP write deadline                (default: 10s)
  API_SHUTDOWN_TIMEOUT   Graceful shutdown window           (default: 15s)

  API_AUTH_AUTHORIZE_URL OIDC authorization endpoint
  API_AUTH_TOKEN_URL     OIDC token endpoint
  API_AUTH_USERINFO_URL  OIDC userinfo endpoint
  API_AUTH_CLIENT_ID     OIDC public client id
  API_AUTH_REDIRECT_URI  API callback URI registered at the IDP
  API_AUTH_SCOPES        Space-separated OIDC scopes (default: openid profile email groups)

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

	rootCmd.Flags().StringVar(&addr, "addr", envOr("API_ADDR", ":8090"),
		"TCP address the API server listens on. Env: API_ADDR")
	rootCmd.Flags().StringVar(&logLevel, "log-level", envOr("API_LOG_LEVEL", "info"),
		"Log level (debug, info, warn, error). Env: API_LOG_LEVEL")
	rootCmd.Flags().StringVar(&readTimeout, "read-timeout", envOr("API_READ_TIMEOUT", "10s"),
		"Maximum duration for reading an entire request. Env: API_READ_TIMEOUT")
	rootCmd.Flags().StringVar(&writeTimeout, "write-timeout", envOr("API_WRITE_TIMEOUT", "10s"),
		"Maximum duration before timing out writes of the response. Env: API_WRITE_TIMEOUT")
	rootCmd.Flags().StringVar(&shutdownTimeout, "shutdown-timeout", envOr("API_SHUTDOWN_TIMEOUT", "15s"),
		"Maximum duration for a graceful shutdown. Env: API_SHUTDOWN_TIMEOUT")
	rootCmd.Flags().StringVar(&authAuthorizeURL, "auth-authorize-url", envOr("API_AUTH_AUTHORIZE_URL", ""),
		"OIDC authorization endpoint. Env: API_AUTH_AUTHORIZE_URL")
	rootCmd.Flags().StringVar(&authTokenURL, "auth-token-url", envOr("API_AUTH_TOKEN_URL", ""),
		"OIDC token endpoint. Env: API_AUTH_TOKEN_URL")
	rootCmd.Flags().StringVar(&authUserInfoURL, "auth-userinfo-url", envOr("API_AUTH_USERINFO_URL", ""),
		"OIDC userinfo endpoint. Env: API_AUTH_USERINFO_URL")
	rootCmd.Flags().StringVar(&authClientID, "auth-client-id", envOr("API_AUTH_CLIENT_ID", ""),
		"OIDC public client ID. Env: API_AUTH_CLIENT_ID")
	rootCmd.Flags().StringVar(&authRedirectURI, "auth-redirect-uri", envOr("API_AUTH_REDIRECT_URI", ""),
		"OIDC redirect URI handled by the API. Env: API_AUTH_REDIRECT_URI")
	rootCmd.Flags().StringVar(&authScopes, "auth-scopes", envOr("API_AUTH_SCOPES", "openid profile email groups"),
		"Space-separated OIDC scopes. Env: API_AUTH_SCOPES")
}

func startServer(cmd *cobra.Command, _ []string) error {
	cfg := config.Config{
		Addr:            addr,
		LogLevel:        logLevel,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		ShutdownTimeout: shutdownTimeout,
		Kubeconfig:      kubeconfig,
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	return server.New(cfg).Run(cmd.Context())
}

// envOr returns the value of the environment variable key, or fallback if unset.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
