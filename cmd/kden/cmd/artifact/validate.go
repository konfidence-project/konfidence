package artifact

import (
	"errors"
	"fmt"

	"github.com/konfidence-project/konfidence/internal/kden/output"
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
	err = validation.RunValidate(filePaths, validation.ValidateConfig{
		CmdDisplayName:      cmd.DisplayName(),
		DefaultFile:         DefaultArtifactFileName,
		ComponentIdentifier: ArtifactIdentifier,
		ValidateFn:          validation.ValidateArtifact,
	})

	var schemaError *validation.SchemaValidationError
	if err != nil && errors.As(err, &schemaError) {
		output.PrintMessage(schemaError.ErrorMessage)
		return nil
	}
	return err
}

func newValidateCmd() *cobra.Command {
	filesUsage := "comma-separated list of artifact path files to validate"
	validateCmd.Flags().StringSlice(ArtifactFilesPathFlag, nil, filesUsage)
	return validateCmd
}
