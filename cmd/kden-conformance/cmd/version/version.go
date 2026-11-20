// Package version provides the conformance executable's build information command.
package version

import (
	"encoding/json"

	"github.com/konfidence-project/konfidence/pkg/build"
	"github.com/spf13/cobra"
)

// NewVersionCmd prints the same build metadata as kden version as JSON.
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the conformance CLI version",
		Long:  "Print the conformance CLI version, build metadata and platform as JSON.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(build.Current())
		},
	}
}
