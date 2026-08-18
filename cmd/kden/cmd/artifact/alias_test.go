package artifact_test

import (
	"context"
	"fmt"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd/artifact"
	"github.com/konfidence-project/konfidence/internal/kden/ocm"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	ocmgenericspecv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
)

var _ = Describe("alias", func() {
	var aliasRootCmd *cobra.Command

	BeforeEach(func() {
		artifact.GetOcmConfiguration = func(_ *cobra.Command) (*ocmgenericspecv1.Config, error) {
			return &ocmgenericspecv1.Config{}, nil
		}
		artifact.Alias = func(_ context.Context, _ ocm.AliasProperties, _ *ocmgenericspecv1.Config) error {
			return nil
		}

		aliasRootCmd = &cobra.Command{Use: "kden"}
		aliasRootCmd.AddCommand(artifact.NewAliasCmd())
	})

	Describe("Alias", func() {
		Context("with valid input", func() {
			It("creates the alias successfully", func() {
				aliasRootCmd.SetArgs([]string{"alias", "registry.example.com//konfidence.io/payment-hub:1.0.0", "edge"})
				Expect(aliasRootCmd.Execute()).To(Succeed())
			})
		})

		Context("with invalid arguments", func() {
			It("fails when no arguments are provided", func() {
				aliasRootCmd.SetArgs([]string{"alias"})
				err := aliasRootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("accepts 2 arg(s), received 0"))
			})

			It("fails when only source ref is provided", func() {
				aliasRootCmd.SetArgs([]string{"alias", "registry.example.com//konfidence.io/payment-hub:1.0.0"})
				err := aliasRootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("accepts 2 arg(s), received 1"))
			})

			It("fails when too many arguments are provided", func() {
				aliasRootCmd.SetArgs([]string{"alias", "registry.example.com//konfidence.io/payment-hub:1.0.0", "edge", "extra"})
				err := aliasRootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("accepts 2 arg(s), received 3"))
			})
		})

		Context("with failing submethods", func() {
			It("when GetOcmConfiguration returns an error", func() {
				artifact.GetOcmConfiguration = func(_ *cobra.Command) (*ocmgenericspecv1.Config, error) {
					return nil, fmt.Errorf("some ocm config error")
				}
				aliasRootCmd.SetArgs([]string{"alias", "registry.example.com//konfidence.io/payment-hub:1.0.0", "edge"})
				err := aliasRootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("some ocm config error"))
			})

			It("when CreateAlias returns an error", func() {
				artifact.Alias = func(_ context.Context, _ ocm.AliasProperties, _ *ocmgenericspecv1.Config) error {
					return fmt.Errorf("some alias error")
				}
				aliasRootCmd.SetArgs([]string{"alias", "registry.example.com//konfidence.io/payment-hub:1.0.0", "edge"})
				err := aliasRootCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to create alias"))
				Expect(err.Error()).To(ContainSubstring("some alias error"))
			})
		})
	})
})
