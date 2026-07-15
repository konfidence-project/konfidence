package cmd

import (
	"os"

	"github.com/spf13/cobra"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/server"
	pkgk8s "github.com/konfidence-project/konfidence/pkg/k8s"
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
  API_KUBECONFIG         Path to kubeconfig (local dev only)`,
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
	rootCmd.Flags().String("addr", envOr("API_ADDR", ":8090"),
		"TCP address the API server listens on. Env: API_ADDR")
	rootCmd.Flags().String("log-level", envOr("API_LOG_LEVEL", "info"),
		"Log level (debug, info, warn, error). Env: API_LOG_LEVEL")
	rootCmd.Flags().String("read-timeout", envOr("API_READ_TIMEOUT", "10s"),
		"Maximum duration for reading an entire request. Env: API_READ_TIMEOUT")
	rootCmd.Flags().String("write-timeout", envOr("API_WRITE_TIMEOUT", "10s"),
		"Maximum duration before timing out writes of the response. Env: API_WRITE_TIMEOUT")
	rootCmd.Flags().String("shutdown-timeout", envOr("API_SHUTDOWN_TIMEOUT", "15s"),
		"Maximum duration for a graceful shutdown. Env: API_SHUTDOWN_TIMEOUT")
	rootCmd.Flags().String("kubeconfig", envOr("API_KUBECONFIG", ""),
		"Path to a kubeconfig file. Defaults to in-cluster config when empty. Env: API_KUBECONFIG")
}

func startServer(cmd *cobra.Command, _ []string) error {
	flags := cmd.Flags()

	addr, err := flags.GetString("addr")
	if err != nil {
		return err
	}
	logLevel, err := flags.GetString("log-level")
	if err != nil {
		return err
	}
	readTimeout, err := flags.GetString("read-timeout")
	if err != nil {
		return err
	}
	writeTimeout, err := flags.GetString("write-timeout")
	if err != nil {
		return err
	}
	shutdownTimeout, err := flags.GetString("shutdown-timeout")
	if err != nil {
		return err
	}
	kubeconfig, err := flags.GetString("kubeconfig")
	if err != nil {
		return err
	}

	cfg := config.Config{
		Addr:            addr,
		LogLevel:        logLevel,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		ShutdownTimeout: shutdownTimeout,
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	k8sClient, err := pkgk8s.NewClient(kubeconfig)
	if err != nil {
		return err
	}

	return server.New(cfg, k8sClient).Run(cmd.Context())
}

// envOr returns the value of the environment variable key, or fallback if unset.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
