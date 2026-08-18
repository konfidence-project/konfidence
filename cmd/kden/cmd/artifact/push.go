package artifact

import (
	"fmt"

	"github.com/konfidence-project/konfidence/internal/kden/ocm"
	"github.com/konfidence-project/konfidence/internal/kden/validation"
	"github.com/spf13/cobra"
)

var (
	FileFlag          = "file"
	RegistryFlag      = "registry"
	FileFlagShort     = "f"
	RegistryFlagShort = "r"
)

var (
	ReadConstructorFromFile   = ocm.ReadConstructorFromFile
	GetOcmConstructorProvider = ocm.GetOcmConstructorProvider
	PushComponentConstructor  = ocm.PushComponentConstructor
	ValidateArtifact          = validation.RunValidate
)

func NewPushCmd() (*cobra.Command, error) {
	var push = &cobra.Command{
		Use: "push",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, err := cmd.Flags().GetString(FileFlag)
			if err != nil {
				return err
			}

			registry, err := cmd.Flags().GetString(RegistryFlag)
			if err != nil {
				return err
			}

			constructor, err := ReadConstructorFromFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read constructor from file %s: %w", filePath, err)
			}

			ocmConfiguration, err := GetOcmConfiguration(cmd)
			if err != nil {
				return fmt.Errorf("failed to get ocm config: %w", err)
			}

			err = ValidateArtifact([]string{filePath}, validation.ValidateConfig{
				CmdDisplayName:      cmd.DisplayName(),
				DefaultFile:         DefaultArtifactFileName,
				ComponentIdentifier: ArtifactIdentifier,
				ValidateFn:          validation.ValidateArtifact,
			})
			if err != nil {
				return fmt.Errorf("validation failed: %s", err.Error())
			}

			constructorOptionsProvider, err := GetOcmConstructorProvider(ocmConfiguration, cmd.Context(), registry)
			if err != nil {
				return err
			}

			return PushComponentConstructor(cmd.Context(), constructorOptionsProvider, constructor)
		},
	}

	push.Flags().StringP(RegistryFlag, RegistryFlagShort, "", "--registry=docker.io/<subpath>")
	push.Flags().StringP(FileFlag, FileFlagShort, "", "--file=<path>")
	err := push.MarkFlagRequired(FileFlag)
	if err != nil {
		return nil, err
	}
	err = push.MarkFlagRequired(RegistryFlag)
	if err != nil {
		return nil, err
	}

	return push, nil
}
