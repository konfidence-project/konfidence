package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "0.0.1"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the kden CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("kden CLI version: %s\n", version)
	},
}

func NewVersionCmd() *cobra.Command {
	return versionCmd
}
