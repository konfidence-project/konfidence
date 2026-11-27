package artifact

import (
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
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

func NewArtifactCmd(appConfig *cfg.AppConfig) *cobra.Command {
	artifactCmd.AddCommand(newValidateCmd())
	pushCmd, err := NewPushCmd()
	if err != nil {
		panic(err)
	}
	artifactCmd.AddCommand(pushCmd)
	artifactCmd.AddCommand(NewSignCmd())
	artifactCmd.AddCommand(NewAliasCmd())

	contentCmd, err := NewContentCmd(appConfig)
	if err != nil {
		panic(err)
	}
	contentCmd.Flags().StringP("projectId", "p", "", "The ID of the project the artifact belongs to (required)")
	err = contentCmd.MarkFlagRequired("projectId")
	if err != nil {
		panic(err)
	}
	contentCmd.Flags().String("ref", "", "The OCM ref of the artifact to retrieve content for (required)")
	err = contentCmd.MarkFlagRequired("ref")
	if err != nil {
		panic(err)
	}
	artifactCmd.AddCommand(contentCmd)
	return artifactCmd
}
