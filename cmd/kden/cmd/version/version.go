package version

import (
	"fmt"

	"github.com/konfidence-project/konfidence/pkg/build"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the kden CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "kden CLI version: %s\n", build.Version)
	},
}

func NewVersionCmd() *cobra.Command {
	return versionCmd
}
