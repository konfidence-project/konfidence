package cmd

import (
	"context"
	"os"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd/artifact"
	"github.com/konfidence-project/konfidence/cmd/kden/cmd/completion"
	"github.com/konfidence-project/konfidence/cmd/kden/cmd/config"
	"github.com/konfidence-project/konfidence/cmd/kden/cmd/man"
	"github.com/konfidence-project/konfidence/cmd/kden/cmd/vector"
	"github.com/konfidence-project/konfidence/cmd/kden/cmd/version"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/log"
	"github.com/spf13/cobra"
)

var rootCmdDescription = `Kden is an extensible command-line interface tool with Go & Cobra.

Kden CLI supports developers, DevOps engineers, and release managers working with Konfidence -
a cloud-native continuous delivery and application lifecycle management
framework built on Kubernetes.

Example usage:
  kden version
  kden help`

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

	rootCmd.AddCommand(completion.NewCompletionCmd())
	rootCmd.AddCommand(config.NewConfigCmd())
	rootCmd.AddCommand(man.NewManCmd())
	rootCmd.AddCommand(artifact.NewArtifactCmd())
	rootCmd.AddCommand(version.NewVersionCmd())
	rootCmd.AddCommand(vector.NewVectorCmd())
}
