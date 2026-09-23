package landscape

import (
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/output"
	"github.com/spf13/cobra"
)

var landscapeCmd = &cobra.Command{
	Use:     "landscape",
	Aliases: []string{"l"},
	Short:   "Manage landscapes",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		output.PrintMessage(cmd.UsageString())
	},
}

func NewLandscapeCmd(appConfig *cfg.AppConfig) *cobra.Command {
	listCmd, err := NewListCmd(appConfig)
	if err != nil {
		panic(err)
	}

	listCmd.Flags().StringP("projectId", "p", "", "The ID of the project the landscapes belong to (required)")
	err = listCmd.MarkFlagRequired("projectId")
	if err != nil {
		panic(err)
	}

	landscapeCmd.AddCommand(listCmd)
	return landscapeCmd
}
