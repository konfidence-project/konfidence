package vector

import (
	"fmt"

	"github.com/konfidence-project/konfidence/internal/kden/ocm"
	"github.com/konfidence-project/konfidence/internal/kden/validation"
	"github.com/spf13/cobra"
)

var (
	FileFlag          = "file"
	RegistryFlag      = "registry"
	AliasFlag         = "alias"
	FileFlagShort     = "f"
	RegistryFlagShort = "r"
	AliasFlagShort    = "a"
)

var (
	ReadConstructorFromFile   = ocm.ReadConstructorFromFile
	GetOcmConstructorProvider = ocm.GetOcmConstructorProvider
	PushComponentConstructor  = ocm.PushComponentConstructor
	AddComponentVersionAlias  = ocm.AddComponentVersionAlias
	ValidateVector            = validation.RunValidate
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

			alias, err := cmd.Flags().GetString(AliasFlag)
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

			err = ValidateVector([]string{filePath}, validation.ValidateConfig{
				CmdDisplayName:      cmd.DisplayName(),
				DefaultFile:         DefaultVectorFileName,
				ComponentIdentifier: VectorIdentifier,
				ValidateFn:          validation.ValidateVector,
			})
			if err != nil {
				return fmt.Errorf("validation failed: %s", err.Error())
			}

			constructorOptionsProvider, err := GetOcmConstructorProvider(ocmConfiguration, cmd.Context(), registry)
			if err != nil {
				return err
			}

			err = PushComponentConstructor(cmd.Context(), constructorOptionsProvider, constructor)
			if err != nil {
				return err
			}

			if alias != "" {
				for _, component := range constructor.Components {
					if err := AddComponentVersionAlias(cmd.Context(), component.Name, component.Version, alias, constructorOptionsProvider); err != nil {
						return err
					}
				}
			}

			return nil
		},
	}

	push.Flags().StringP(RegistryFlag, RegistryFlagShort, "", "--registry=docker.io/<subpath>")
	push.Flags().StringP(FileFlag, FileFlagShort, "", "--file=<path>")
	push.Flags().StringP(AliasFlag, AliasFlagShort, "", "--alias=<alias-name>")
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
