package artifact

import (
	"github.com/konfidence-project/konfidence/internal/kden/output"
	"github.com/spf13/cobra"
)

var artifactCmd = &cobra.Command{
	Use:     "artifact",
	Aliases: []string{"a"},
	Short:   "Manage artifacts",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		output.PrintMessage(cmd.UsageString())
	},
}

func NewArtifactCmd() *cobra.Command {
	artifactCmd.AddCommand(newValidateCmd())
	pushCmd, err := NewPushCmd()
	if err != nil {
		panic(err)
	}
	artifactCmd.AddCommand(pushCmd)
	artifactCmd.AddCommand(NewSignCmd())
	return artifactCmd
}
