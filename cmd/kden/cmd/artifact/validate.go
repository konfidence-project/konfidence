package artifact

import (
	"fmt"

	"github.com/konfidence-project/konfidence/internal/kden/validation"
	"github.com/spf13/cobra"
)

const (
	ArtifactFilesPathFlag   = "files"
	DefaultArtifactFileName = "artifact"
	ArtifactIdentifier      = "artifact"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate artifacts against predefined JSON schema.",
	Long:  ``,
	RunE:  runValidateCmd,
}

func runValidateCmd(cmd *cobra.Command, _ []string) error {
	filePaths, err := cmd.Flags().GetStringSlice(ArtifactFilesPathFlag)
	if err != nil {
		return fmt.Errorf("failed to get %s files flag: %w", ArtifactIdentifier, err)
	}
	return validation.RunValidate(filePaths, validation.ValidateConfig{
		CmdDisplayName:      cmd.DisplayName(),
		DefaultFile:         DefaultArtifactFileName,
		ComponentIdentifier: ArtifactIdentifier,
		ValidateFn:          validation.ValidateArtifact,
	})
}

func newValidateCmd() *cobra.Command {
	filesUsage := "comma-separated list of artifact path files to validate"
	validateCmd.Flags().StringSlice(ArtifactFilesPathFlag, nil, filesUsage)
	return validateCmd
}
