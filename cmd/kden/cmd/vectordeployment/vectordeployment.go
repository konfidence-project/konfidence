package vectordeployment

import (
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/output"
	"github.com/spf13/cobra"
)

var vectorDeploymentCmd = &cobra.Command{
	Use:     "vector-deployment",
	Aliases: []string{"vd"},
	Short:   "Manage vector deployments",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		output.PrintMessage(cmd.UsageString())
	},
}

func NewVectorDeploymentCmd(appConfig *cfg.AppConfig) *cobra.Command {
	listCmd, err := NewListCmd(appConfig)
	if err != nil {
		panic(err)
	}

	listCmd.Flags().StringP("projectId", "p", "", "The ID of the project the vector deployments belong to (required)")
	err = listCmd.MarkFlagRequired("projectId")
	if err != nil {
		panic(err)
	}

	listCmd.Flags().StringP("landscapeId", "l", "", "The ID of the landscape the vector deployments belong to")

	vectorDeploymentCmd.AddCommand(listCmd)
	return vectorDeploymentCmd
}
