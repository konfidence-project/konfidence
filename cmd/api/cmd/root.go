package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/server"
	stageapi "github.com/konfidence-project/konfidence/internal/stage/api"
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

It sits at the border of the star cluster and exposes an HTTP API consumed by
the kden CLI, the Star Dashboard, and any external UI. Inbound requests are
translated into reads and writes against the Kubernetes API (Konfidence CRDs).

Configuration is read from environment variables (API_*) and can be overridden
by the flags below.

  API_ADDR               TCP address to listen on           (default: :8090)
  API_LOG_LEVEL          Log verbosity: debug/info/warn/error (default: info)
  API_READ_TIMEOUT       HTTP read deadline                 (default: 10s)
  API_WRITE_TIMEOUT      HTTP write deadline                (default: 10s)
  API_SHUTDOWN_TIMEOUT   Graceful shutdown window           (default: 15s)

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
}

func startServer(cmd *cobra.Command, _ []string) error {
	cfg.Scheme = scheme

	parsed, err := cfg.Validate()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return server.New(parsed, stageapi.Mount).Run(ctx)
}

// envOr returns the value of the environment variable key, or fallback if unset.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}