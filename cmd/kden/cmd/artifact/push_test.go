package artifact_test

import (
	"context"
	"fmt"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd/artifact"
	"github.com/konfidence-project/konfidence/internal/kden/ocm"
	"github.com/konfidence-project/konfidence/internal/kden/validation"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	ocmgenericspecv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
	ocmconstructorspecv1 "ocm.software/open-component-model/bindings/go/constructor/spec/v1"
)

var rootCmd *cobra.Command

var _ = BeforeEach(func() {
	artifact.ReadConstructorFromFile = func(filePath string) (*ocmconstructorspecv1.ComponentConstructor, error) {
		return &ocmconstructorspecv1.ComponentConstructor{}, nil
	}
	artifact.GetOcmConfiguration = func(cmd *cobra.Command) (*ocmgenericspecv1.Config, error) {
		return &ocmgenericspecv1.Config{}, nil
	}
	artifact.ValidateArtifact = func(filePaths []string, cfg validation.ValidateConfig) error {
		return nil
	}
	artifact.GetOcmConstructorProvider = func(
		ocmConfiguration *ocmgenericspecv1.Config,
		ctx context.Context,
		registry string,
	) (*ocm.ConstructorProvider, error) {
		return &ocm.ConstructorProvider{}, nil
	}
	artifact.PushComponentConstructor = func(
		ctx context.Context,
		constructorOptionProvider *ocm.ConstructorProvider,
		constructor *ocmconstructorspecv1.ComponentConstructor,
	) error {
		return nil
	}
	artifact.AddComponentVersionAlias = func(
		ctx context.Context,
		component,
		versionOrAlias,
		alias string,
		constructorOptionsProvider *ocm.ConstructorProvider,
	) error {
		return nil
	}

	rootCmd = &cobra.Command{Use: "kden"}
	pushCmd, err := artifact.NewPushCmd()
	Expect(err).ToNot(HaveOccurred())
	rootCmd.AddCommand(pushCmd)
})

var _ = Describe("push", func() {
	Describe("Push", func() {
		Context("with valid input", func() {
			It("pushes the component version successfully", func() {
				rootCmd.SetArgs([]string{"push", "--file", "anyfile.yaml", "--registry", "docker.io/my-org/my-component"})
				err := rootCmd.Execute()
				Expect(err).ToNot(HaveOccurred())
			})
			It("pushes the component version successfully with an alias", func() {
				rootCmd.SetArgs([]string{"push", "--file", "anyfile.yaml", "--registry", "docker.io/my-org/my-component", "--alias", "latest"})
				err := rootCmd.Execute()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("with invalid flags", func() {
			It("with missing registry flag", func() {
				rootCmd.SetArgs([]string{"push", "--file", "anyfile.yaml"})
				err := rootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("required flag(s) \"registry\" not set"))
			})
			It("with missing file flag", func() {
				rootCmd.SetArgs([]string{"push", "--registry", "my-registry.com"})
				err := rootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("required flag(s) \"file\" not set"))
			})
			It("with missing file and registry flag", func() {
				rootCmd.SetArgs([]string{"push"})
				err := rootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("required flag(s) \"file\", \"registry\" not set"))
			})
			It("with missing value for file flag", func() {
				rootCmd.SetArgs([]string{"push", "--registry", "my-registry.com", "--file"})
				err := rootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("flag needs an argument: --file"))
			})
			It("with missing value for alias flag", func() {
				rootCmd.SetArgs([]string{"push", "--registry", "my-registry.com", "--file", "anyfile.yaml", "--alias"})
				err := rootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("flag needs an argument: --alias"))
			})
		})

		Context("with failing submethods", func() {
			It("when ReadConstructorFromFile returns an error", func() {
				artifact.ReadConstructorFromFile = func(filePath string) (*ocmconstructorspecv1.ComponentConstructor, error) {
					return nil, fmt.Errorf("some error")
				}
				rootCmd.SetArgs([]string{"push", "--file", "anyfile.yaml", "--registry", "docker.io/my-org/my-component"})
				err := rootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to read constructor from file"))
			})

			It("when GetOcmConfiguration returns an error", func() {
				artifact.GetOcmConfiguration = func(cmd *cobra.Command) (*ocmgenericspecv1.Config, error) {
					return nil, fmt.Errorf("some error")
				}
				rootCmd.SetArgs([]string{"push", "--file", "anyfile.yaml", "--registry", "docker.io/my-org/my-component"})
				err := rootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to get ocm config"))
			})

			It("when ValidateArtifact returns an error", func() {
				artifact.ValidateArtifact = func(filePaths []string, cfg validation.ValidateConfig) error {
					return fmt.Errorf("some error")
				}
				rootCmd.SetArgs([]string{"push", "--file", "anyfile.yaml", "--registry", "docker.io/my-org/my-component"})
				err := rootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("some error"))
				Expect(err.Error()).To(ContainSubstring("validation failed"))
			})

			It("when PushComponentConstructor returns an error", func() {
				artifact.PushComponentConstructor = func(
					ctx context.Context,
					constructorOptionProvider *ocm.ConstructorProvider,
					constructor *ocmconstructorspecv1.ComponentConstructor,
				) error {
					return fmt.Errorf("some error")
				}
				rootCmd.SetArgs([]string{"push", "--file", "anyfile.yaml", "--registry", "docker.io/my-org/my-component"})
				err := rootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("some error"))
			})

			It("when AddComponentVersionAlias returns an error", func() {
				artifact.ReadConstructorFromFile = func(filePath string) (*ocmconstructorspecv1.ComponentConstructor, error) {
					return &ocmconstructorspecv1.ComponentConstructor{
						Components: []ocmconstructorspecv1.Component{
							{
								ComponentMeta: ocmconstructorspecv1.ComponentMeta{
									ObjectMeta: ocmconstructorspecv1.ObjectMeta{
										Name:    "my-component",
										Version: "1.0.0",
									},
								},
							},
						},
					}, nil
				}
				artifact.AddComponentVersionAlias = func(
					ctx context.Context,
					component,
					versionOrAlias,
					alias string,
					constructorOptionsProvider *ocm.ConstructorProvider,
				) error {
					return fmt.Errorf("alias 1.0.0 uses semantic version format and cannot be used as an alias")
				}
				rootCmd.SetArgs([]string{"push", "--file", "anyfile.yaml", "--registry", "docker.io/my-org/my-component", "--alias", "1.0.0"})
				err := rootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("alias 1.0.0 uses semantic version format and cannot be used as an alias"))
			})

			It("when GetOcmConstructorProvider returns an error", func() {
				artifact.GetOcmConstructorProvider = func(
					ocmConfiguration *ocmgenericspecv1.Config,
					ctx context.Context,
					registry string,
				) (*ocm.ConstructorProvider, error) {
					return nil, fmt.Errorf("some error")
				}
				rootCmd.SetArgs([]string{"push", "--file", "anyfile.yaml", "--registry", "docker.io/my-org/my-component"})
				err := rootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("some error"))
			})
		})
	})
})
