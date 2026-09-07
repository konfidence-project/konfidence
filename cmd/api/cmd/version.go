package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/konfidence-project/konfidence/pkg/build"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the API server build version",
		Long:  `Print the konfidence API server version, git commit, build date and platform.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			switch output {
			case "plain":
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), build.Current().Version)
				return nil
			case "json":
				out, err := json.MarshalIndent(build.Current(), "", "  ")
				if err != nil {
					return fmt.Errorf("marshalling version failed: %w", err)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(out))
				return nil
			default:
				return fmt.Errorf("invalid --output %q: supported values are 'json' and 'plain'", output)
			}
		},
	}

	cmd.Flags().StringVar(&output, "output", "json",
		"output format: 'json' (default) or 'plain' (bare version string)")

	return cmd
}
