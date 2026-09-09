package version

import (
	"fmt"
	"strings"

	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/output"
	"github.com/konfidence-project/konfidence/pkg/build"
	"github.com/spf13/cobra"
)

const installCommand = "curl -fsSL https://konfidence.cloud/install.sh | sh"

func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the kden CLI version",
		Long:  `Print the kden CLI version, build metadata and platform.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			info := build.Current()

			// plain is a bare version string for scripts (VER=$(kden version
			// --output plain)); no update hint, nothing else on stdout or stderr.
			if output.FormattedOutput(strings.ToLower(cfg.Config.Output)) == output.PlainOutputFormat {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), info.Version)
				return nil
			}

			formatted, err := output.ResolveFormat(info, "version")
			if err != nil {
				return fmt.Errorf("formatting version failed: %w", err)
			}

			if output.FormattedOutput(strings.ToLower(cfg.Config.Output)) == output.TablePrettyOutputFormat {
				// The pretty table renders itself to stdout (and carries the update
				// hint as its footer), so there's nothing more to print here.
				return nil
			}

			// json and yaml print the payload to stdout, then the update hint to
			// stderr so a piped stdout stays clean.
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), strings.TrimRight(formatted, "\n"))
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "\nTo update, re-run the installer:\n  %s\n", installCommand)
			return nil
		},
	}

	cmd.Flags().String(
		"output",
		"",
		"Output format. Supported values: 'json', 'yaml', 'pretty' and 'plain' (bare version string)")

	return cmd
}
