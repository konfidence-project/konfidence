package vector

import (
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

func NewVectorCmd() *cobra.Command {
	vectorCmd.AddCommand(newValidateCmd())
	pushCmd, err := NewPushCmd()
	if err != nil {
		panic(err)
	}
	vectorCmd.AddCommand(pushCmd)
	vectorCmd.AddCommand(NewSignCmd())
	return vectorCmd
}
