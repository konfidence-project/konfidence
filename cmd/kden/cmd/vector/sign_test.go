package vector_test

import (
	"context"
	"fmt"
	"io"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd/vector"
	"github.com/konfidence-project/konfidence/internal/kden/ocm"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	ocmgenericspecv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
	ocmdescriptorruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
)

var _ = Describe("sign", func() {
	var signRootCmd *cobra.Command

	BeforeEach(func() {
		vector.GetOcmConfiguration = func(_ *cobra.Command) (*ocmgenericspecv1.Config, error) {
			return &ocmgenericspecv1.Config{}, nil
		}
		vector.Sign = func(_ context.Context, _ ocm.SigningProperties, _ *ocmgenericspecv1.Config) (*ocmdescriptorruntime.Signature, error) {
			return &ocmdescriptorruntime.Signature{}, nil
		}
		vector.PrintSignature = func(_ io.Writer, _ ocmdescriptorruntime.Signature) error {
			return nil
		}

		signRootCmd = &cobra.Command{Use: "kden"}
		signRootCmd.AddCommand(vector.NewSignCmd())
	})

	Describe("Sign", func() {
		Context("with valid input", func() {
			It("signs the vector successfully", func() {
				signRootCmd.SetArgs([]string{"sign", "docker.io/my-org/my-component:1.0.0"})
				err := signRootCmd.Execute()
				Expect(err).ToNot(HaveOccurred())
			})

			It("signs with explicit flags", func() {
				signRootCmd.SetArgs([]string{
					"sign",
					"docker.io/my-org/my-component:1.0.0",
					"--signer-spec", "/path/to/spec.yaml",
					"--signature-name", "my-sig",
					"--dry-run",
					"--overwrite-signatures",
				})
				err := signRootCmd.Execute()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("with invalid arguments", func() {
			It("fails when no component version argument is provided", func() {
				signRootCmd.SetArgs([]string{"sign"})
				err := signRootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("accepts 1 arg(s), received 0"))
			})

			It("fails when too many arguments are provided", func() {
				signRootCmd.SetArgs([]string{"sign", "arg1", "arg2"})
				err := signRootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("accepts 1 arg(s), received 2"))
			})
		})

		Context("with failing submethods", func() {
			It("when GetOcmConfiguration returns an error", func() {
				vector.GetOcmConfiguration = func(_ *cobra.Command) (*ocmgenericspecv1.Config, error) {
					return nil, fmt.Errorf("some ocm config error")
				}
				signRootCmd.SetArgs([]string{"sign", "docker.io/my-org/my-component:1.0.0"})
				err := signRootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("some ocm config error"))
			})

			It("when Sign returns an error", func() {
				vector.Sign = func(_ context.Context, _ ocm.SigningProperties, _ *ocmgenericspecv1.Config) (*ocmdescriptorruntime.Signature, error) {
					return nil, fmt.Errorf("some sign error")
				}
				signRootCmd.SetArgs([]string{"sign", "docker.io/my-org/my-component:1.0.0"})
				err := signRootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to sign vector"))
				Expect(err.Error()).To(ContainSubstring("some sign error"))
			})

			It("when PrintSignature returns an error", func() {
				vector.PrintSignature = func(_ io.Writer, _ ocmdescriptorruntime.Signature) error {
					return fmt.Errorf("some print error")
				}
				signRootCmd.SetArgs([]string{"sign", "docker.io/my-org/my-component:1.0.0"})
				err := signRootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to print signature"))
				Expect(err.Error()).To(ContainSubstring("some print error"))
			})
		})
	})
})
