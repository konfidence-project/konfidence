package stage

import (
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/output"
	"github.com/spf13/cobra"
)

var stageCmd = &cobra.Command{
	Use:     "stage",
	Aliases: []string{"s"},
	Short:   "Manage stages",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		output.PrintMessage(cmd.UsageString())
	},
}

func NewStageCmd(appConfig *cfg.AppConfig) *cobra.Command {
	listCmd, err := NewListCmd(appConfig)
	if err != nil {
		panic(err)
	}

	listCmd.Flags().StringP("projectId", "p", "", "The ID of the project the stages belong to (required)")
	err = listCmd.MarkFlagRequired("projectId")
	if err != nil {
		panic(err)
	}

	listCmd.Flags().StringP("landscapeId", "l", "", "The ID of the landscape the stages belong to")

	stageCmd.AddCommand(listCmd)
	return stageCmd
}
