package vectorpromotion

import (
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/output"
	"github.com/spf13/cobra"
)

var vectorPromotionCmd = &cobra.Command{
	Use:     "vector-promotion",
	Aliases: []string{"vp"},
	Short:   "Manage vector promotions",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		output.PrintMessage(cmd.UsageString())
	},
}

func NewVectorPromotionCmd(appConfig *cfg.AppConfig) *cobra.Command {
	listCmd, err := NewListCmd(appConfig)
	if err != nil {
		panic(err)
	}

	listCmd.Flags().StringP("projectId", "p", "", "The ID of the project the vector promotions belong to (required)")
	err = listCmd.MarkFlagRequired("projectId")
	if err != nil {
		panic(err)
	}

	getCmd, err := NewGetCmd(appConfig)
	if err != nil {
		panic(err)
	}

	getCmd.Flags().StringP("projectId", "p", "", "The ID of the project the vector promotions belong to (required)")
	err = getCmd.MarkFlagRequired("projectId")
	if err != nil {
		panic(err)
	}

	getCmd.Flags().StringP("vectorPromotionConfigId", "v", "", "The ID of the vector promotion configuration (required)")
	err = getCmd.MarkFlagRequired("vectorPromotionConfigId")
	if err != nil {
		panic(err)
	}

	vectorPromotionCmd.AddCommand(listCmd)
	vectorPromotionCmd.AddCommand(getCmd)
	return vectorPromotionCmd
}
