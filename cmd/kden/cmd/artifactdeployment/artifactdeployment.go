package artifactdeployment

import (
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/output"
	"github.com/spf13/cobra"
)

var artifactDeploymentCmd = &cobra.Command{
	Use:     "artifact-deployment",
	Aliases: []string{"ad"},
	Short:   "Manage artifact deployments",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		output.PrintMessage(cmd.UsageString())
	},
}

func NewArtifactDeploymentCmd(appConfig *cfg.AppConfig) *cobra.Command {
	listCmd, err := NewListCmd(appConfig)
	if err != nil {
		panic(err)
	}

	listCmd.Flags().StringP("projectId", "p", "", "The ID of the project the artifact deployments belong to (required)")
	err = listCmd.MarkFlagRequired("projectId")
	if err != nil {
		panic(err)
	}

	listCmd.Flags().StringP("landscapeId", "l", "", "The ID of the landscape the artifact deployments belong to")
	listCmd.Flags().StringP("vectorDeploymentId", "d", "", "The ID of the vector deployment the artifact deployments belong to")

	artifactDeploymentCmd.AddCommand(listCmd)
	return artifactDeploymentCmd
}
