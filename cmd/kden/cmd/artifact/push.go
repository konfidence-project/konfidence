package artifact

import (
	"context"
	"fmt"

	"github.com/konfidence-project/konfidence/internal/kden/ocm"
	"github.com/konfidence-project/konfidence/internal/kden/validation"
	"github.com/spf13/cobra"
	ocmgenericspecv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
	ocmconstructorspecv1 "ocm.software/open-component-model/bindings/go/constructor/spec/v1"
)

var (
	FileFlag          = "file"
	RegistryFlag      = "registry"
	FileFlagShort     = "f"
	RegistryFlagShort = "r"
)

type PushCmdMethods struct {
	ReadConstructorFromFile  func(filePath string) (*ocmconstructorspecv1.ComponentConstructor, error)
	GetOcmConfiguration      func(cmd *cobra.Command) (*ocmgenericspecv1.Config, error)
	ValidateArtifact         func(filePaths []string, cfg validation.ValidateConfig) error
	PushComponentConstructor func(
		ocmConfiguration *ocmgenericspecv1.Config,
		ctx context.Context,
		registry string,
		constructor *ocmconstructorspecv1.ComponentConstructor,
	) error
}

var PushCmdMethodsImpl = PushCmdMethods{
	ReadConstructorFromFile:  ocm.ReadConstructorFromFile,
	GetOcmConfiguration:      ocm.GetOcmConfiguration,
	ValidateArtifact:         validation.RunValidate,
	PushComponentConstructor: ocm.PushComponentConstructor,
}

func NewPushCmd() (*cobra.Command, error) {
	var push = &cobra.Command{
		Use: "push",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, err := cmd.Flags().GetString(FileFlag)
			if err != nil {
				return err
			}

			constructor, err := PushCmdMethodsImpl.ReadConstructorFromFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read constructor from file %s: %w", filePath, err)
			}

			ocmConfiguration, err := PushCmdMethodsImpl.GetOcmConfiguration(cmd)
			if err != nil {
				return fmt.Errorf("failed to get ocm config: %w", err)
			}

			err = PushCmdMethodsImpl.ValidateArtifact([]string{filePath}, validation.ValidateConfig{
				CmdDisplayName:      cmd.DisplayName(),
				DefaultFile:         DefaultArtifactFileName,
				ComponentIdentifier: ArtifactIdentifier,
				ValidateFn:          validation.ValidateArtifact,
			})
			if err != nil {
				return err
			}

			registry, err := cmd.Flags().GetString(RegistryFlag)
			if err != nil {
				return err
			}

			return PushCmdMethodsImpl.PushComponentConstructor(ocmConfiguration, cmd.Context(), registry, constructor)
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
