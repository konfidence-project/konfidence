package vector

import (
	"fmt"

	"github.com/konfidence-project/konfidence/internal/kden/validation"
	"github.com/spf13/cobra"
)

const (
	VectorFilesPathFlag   = "files"
	DefaultVectorFileName = "vector"
	VectorIdentifier      = "vector"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a vector against a predefined JSON schema.",
	Long:  ``,
	RunE:  runValidateCmd,
}

func runValidateCmd(cmd *cobra.Command, _ []string) error {
	filePaths, err := cmd.Flags().GetStringSlice(VectorIdentifier)
	if err != nil {
		return fmt.Errorf("failed to get %s files flag: %w", VectorIdentifier, err)
	}
	return validation.RunValidate(filePaths, validation.ValidateConfig{
		CmdDisplayName:      cmd.DisplayName(),
		DefaultFile:         DefaultVectorFileName,
		ComponentIdentifier: VectorIdentifier,
		ValidateFn:          validation.ValidateVector,
	})
}

func newValidateCmd() *cobra.Command {
	filesUsage := "comma-separated list of vector path files to validate"
	validateCmd.Flags().StringSlice(VectorFilesPathFlag, nil, filesUsage)
	return validateCmd
}
