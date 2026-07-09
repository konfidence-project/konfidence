package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/server"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	scheme = runtime.NewScheme()
)

var (
	addr            string
	logLevel        string
	readTimeout     string
	writeTimeout    string
	shutdownTimeout string
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

Kubernetes client config is resolved automatically via the standard KUBECONFIG
env var or in-cluster config when deployed as a pod — no flag required.`,
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
	utilruntime.Must(galaxy.AddToScheme(scheme))
	utilruntime.Must(star.AddToScheme(scheme))

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
}

func startServer(cmd *cobra.Command, _ []string) error {
	cfg := config.Config{
		Addr:            addr,
		LogLevel:        logLevel,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		ShutdownTimeout: shutdownTimeout,
		Scheme:          scheme,
	}

	parsed, err := cfg.Validate()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return server.New(parsed).Run(ctx)
}

// envOr returns the value of the environment variable key, or fallback if unset.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
