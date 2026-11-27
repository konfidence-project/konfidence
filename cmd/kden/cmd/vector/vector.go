package vector

import (
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/output"
	"github.com/spf13/cobra"
)

var vectorCmd = &cobra.Command{
	Use:     "vector",
	Aliases: []string{"vc"},
	Short:   "Manage vectors",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		output.PrintMessage(cmd.UsageString())
	},
}

func NewVectorCmd(appConfig *cfg.AppConfig) *cobra.Command {
	vectorCmd.AddCommand(newValidateCmd())
	pushCmd, err := NewPushCmd()
	if err != nil {
		panic(err)
	}
	vectorCmd.AddCommand(pushCmd)
	vectorCmd.AddCommand(NewSignCmd())

	contentCmd, err := NewContentCmd(appConfig)
	if err != nil {
		panic(err)
	}
	contentCmd.Flags().StringP("projectId", "p", "", "The ID of the project the vectors belong to (required)")
	err = contentCmd.MarkFlagRequired("projectId")
	if err != nil {
		panic(err)
	}
	contentCmd.Flags().StringSlice("refs", nil, "The OCM refs of the vectors to retrieve content for (required)")
	err = contentCmd.MarkFlagRequired("refs")
	if err != nil {
		panic(err)
	}
	vectorCmd.AddCommand(contentCmd)

	diffCmd, err := NewDiffCmd(appConfig)
	if err != nil {
		panic(err)
	}
	diffCmd.Flags().StringP("projectId", "p", "", "The ID of the project the vectors belong to (required)")
	err = diffCmd.MarkFlagRequired("projectId")
	if err != nil {
		panic(err)
	}
	diffCmd.Flags().Bool("include-unchanged", false, "Include unchanged components and config entries in the diff")
	diffCmd.Flags().String("left", "", "The OCM ref of the left vector to compare (required)")
	err = diffCmd.MarkFlagRequired("left")
	if err != nil {
		panic(err)
	}
	diffCmd.Flags().String("right", "", "The OCM ref of the right vector to compare (required)")
	err = diffCmd.MarkFlagRequired("right")
	if err != nil {
		panic(err)
	}
	vectorCmd.AddCommand(diffCmd)
	return vectorCmd
}
