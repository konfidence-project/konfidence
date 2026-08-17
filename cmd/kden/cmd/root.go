package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd/artifact"
	"github.com/konfidence-project/konfidence/cmd/kden/cmd/auth"
	"github.com/konfidence-project/konfidence/cmd/kden/cmd/completion"
	"github.com/konfidence-project/konfidence/cmd/kden/cmd/config"
	"github.com/konfidence-project/konfidence/cmd/kden/cmd/man"
	"github.com/konfidence-project/konfidence/cmd/kden/cmd/project"
	"github.com/konfidence-project/konfidence/cmd/kden/cmd/vector"
	"github.com/konfidence-project/konfidence/cmd/kden/cmd/version"
	kdenauth "github.com/konfidence-project/konfidence/internal/kden/auth"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/log"
	"github.com/spf13/cobra"
	"ocm.software/open-component-model/cli/cmd/setup"
)

var rootCmdDescription = `Kden is an extensible command-line interface tool with Go & Cobra.

Kden CLI supports developers, DevOps engineers, and release managers working with Konfidence -
a cloud-native continuous delivery and application lifecycle management
framework built on Kubernetes.

Example usage:
  kden version
  kden help`
var appConfig = &cfg.AppConfig{}

var rootCmd = &cobra.Command{
	Use:   cfg.RootCommandName,
	Short: "Kden CLI tool for working with Konfidence",
	Long:  rootCmdDescription,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := cfg.Configure(cmd)
		if err != nil {
			log.Errorf("failed to configure kden CLI: %s\n", err)
			os.Exit(1)
		}

		handler, err := log.ResolveLogHandler(cfg.Config.LogLevel, cfg.Config.LogFormat)
		if err != nil {
			log.Errorf("failed to resolve log handler: %s\n", err)
			os.Exit(1)
		}

		log.InitLogger(handler)

		appConfig.APIProvider = cfg.NewAPIClientProvider(func() (*kdenauth.Client, error) {
			loginTimeout, err := time.ParseDuration(cfg.Config.LoginTimeout)
			if err != nil {
				return nil, fmt.Errorf("parsing login timeout failed: %w", err)
			}

			requestTimeout, err := time.ParseDuration(cfg.Config.RequestTimeout)
			if err != nil {
				return nil, fmt.Errorf("parsing request timeout failed: %w", err)
			}

			return kdenauth.NewClient(
				normalizeAPIEndpoint(cfg.Config.APIEndpoint),
				kdenauth.KeyringCookieStore{},
				loginTimeout,
				requestTimeout,
			)
		})

		setup.Syscalls(cmd)
	},
	SilenceErrors: true,
}

func executeWith(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func GetRootCommand() *cobra.Command {
	return rootCmd
}

func initCmd() {
	rootCmd.PersistentFlags().String(
		"log-level",
		"",
		"Defines the base log level for the application. Supported values are: 'info', 'debug' and 'error'")
	rootCmd.PersistentFlags().String(
		"log-format",
		"",
		"Defines the output format of the application's logs . Supported values are: 'json', 'text' and 'pretty'")
	rootCmd.PersistentFlags().String(
		"output",
		"",
		"Defines the output format for the application. Supported values are: 'json', 'yaml' and 'pretty'")
	rootCmd.PersistentFlags().String(
		"api-endpoint",
		"",
		"Address of the Konfidence API gateway. Env: KDEN_API_ENDPOINT (default: http://localhost:8090)")
	rootCmd.PersistentFlags().String(
		"login-timeout",
		"",
		"Maximum time to wait for browser login. Env: KDEN_LOGIN_TIMEOUT (default: 2m)",
	)
	rootCmd.PersistentFlags().String(
		"request-timeout",
		"",
		"Maximum duration for an API request. Env: KDEN_REQUEST_TIMEOUT (default: 30s)",
	)

	rootCmd.AddCommand(completion.NewCompletionCmd())
	rootCmd.AddCommand(config.NewConfigCmd())
	rootCmd.AddCommand(man.NewManCmd())
	rootCmd.AddCommand(artifact.NewArtifactCmd())
	rootCmd.AddCommand(version.NewVersionCmd())
	rootCmd.AddCommand(vector.NewVectorCmd())
	rootCmd.AddCommand(project.NewProjectCmd(appConfig))

	loginCmd, err := auth.NewLoginCmd(appConfig)
	if err != nil {
		panic(err)
	}

	logoutCmd, err := auth.NewLogoutCmd(appConfig)
	if err != nil {
		panic(err)
	}

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
}

func normalizeAPIEndpoint(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}

	if parsed.Path == "" || parsed.Path == "/" {
		parsed.Path = "/api"
	}

	return strings.TrimRight(parsed.String(), "/")
}
