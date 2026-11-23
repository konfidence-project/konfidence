// Package cmd provides the standalone conformance command, independent of kden.
package cmd

import (
	"github.com/konfidence-project/konfidence/cmd/kden-conformance/cmd/run"
	"github.com/konfidence-project/konfidence/cmd/kden-conformance/cmd/version"
	"github.com/spf13/cobra"
)

// NewCommand constructs a fresh command without parsing process-global flags.
func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:           "kden-conformance",
		Short:         "Run deployer conformance tests",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	command.AddCommand(run.NewRunCmd(), version.NewVersionCmd())
	return command
}
