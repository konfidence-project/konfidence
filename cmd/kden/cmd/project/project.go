package project

import (
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/output"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:     "project",
	Aliases: []string{"pr"},
	Short:   "Manage projects",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		output.PrintMessage(cmd.UsageString())
	},
}

func NewProjectCmd(appConfig *cfg.AppConfig) *cobra.Command {
	listCmd, err := NewListCmd(appConfig)
	if err != nil {
		panic(err)
	}

	projectCmd.AddCommand(listCmd)
	return projectCmd
}
