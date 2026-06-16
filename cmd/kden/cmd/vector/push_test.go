package vector_test

import (
	"context"
	"fmt"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd/vector"
	"github.com/konfidence-project/konfidence/internal/kden/validation"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	ocmgenericspecv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
	ocmconstructorspecv1 "ocm.software/open-component-model/bindings/go/constructor/spec/v1"
)

var rootCmd *cobra.Command

var _ = BeforeEach(func() {
	vector.PushCmdMethodsImpl = vector.PushCmdMethods{
		ReadConstructorFromFile: func(filePath string) (*ocmconstructorspecv1.ComponentConstructor, error) {
			return &ocmconstructorspecv1.ComponentConstructor{}, nil
		},
		GetOcmConfiguration: func(cmd *cobra.Command) (*ocmgenericspecv1.Config, error) {
			return &ocmgenericspecv1.Config{}, nil
		},
		ValidateVector: func(filePaths []string, cfg validation.ValidateConfig) error {
			return nil
		},
		PushComponentConstructor: func(
			ocmConfiguration *ocmgenericspecv1.Config,
			ctx context.Context,
			registry string,
			constructor *ocmconstructorspecv1.ComponentConstructor,
		) error {
			return nil
		},
	}

	rootCmd = &cobra.Command{Use: "kden"}
	pushCmd, err := vector.NewPushCmd()
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
		})

		Context("with failing submethods", func() {
			It("when ReadConstructorFromFile returns an error", func() {
				vector.PushCmdMethodsImpl.ReadConstructorFromFile = func(filePath string) (*ocmconstructorspecv1.ComponentConstructor, error) {
					return nil, fmt.Errorf("some error")
				}
				rootCmd.SetArgs([]string{"push", "--file", "anyfile.yaml", "--registry", "docker.io/my-org/my-component"})
				err := rootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to read constructor from file"))
			})

			It("when GetOcmConfiguration returns an error", func() {
				vector.PushCmdMethodsImpl.GetOcmConfiguration = func(cmd *cobra.Command) (*ocmgenericspecv1.Config, error) {
					return nil, fmt.Errorf("some error")
				}
				rootCmd.SetArgs([]string{"push", "--file", "anyfile.yaml", "--registry", "docker.io/my-org/my-component"})
				err := rootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to get ocm config"))
			})

			It("when ValidateVector returns an error", func() {
				vector.PushCmdMethodsImpl.ValidateVector = func(filePaths []string, cfg validation.ValidateConfig) error {
					return fmt.Errorf("some error")
				}
				rootCmd.SetArgs([]string{"push", "--file", "anyfile.yaml", "--registry", "docker.io/my-org/my-component"})
				err := rootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("some error"))
			})

			It("when PushComponentConstructor returns an error", func() {
				vector.PushCmdMethodsImpl.PushComponentConstructor = func(
					ocmConfiguration *ocmgenericspecv1.Config,
					ctx context.Context,
					registry string,
					constructor *ocmconstructorspecv1.ComponentConstructor,
				) error {
					return fmt.Errorf("some error")
				}
				rootCmd.SetArgs([]string{"push", "--file", "anyfile.yaml", "--registry", "docker.io/my-org/my-component"})
				err := rootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("some error"))
			})
		})
	})
})
