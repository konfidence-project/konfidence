// Package run provides the explicit conformance suite execution command.
package run

import (
	"fmt"

	"github.com/konfidence-project/konfidence/internal/conformance/setup"
	"github.com/konfidence-project/konfidence/internal/conformance/suite"
	"github.com/spf13/cobra"
)

// NewRunCmd constructs the command that loads inputs and executes the suite.
func NewRunCmd() *cobra.Command {
	var options setup.Options
	var reportDirectory string
	command := &cobra.Command{
		Use:   "run",
		Short: "Execute the deployer conformance suite",
		Long:  "Execute the deployer conformance suite using an application artifact and deployment-target configuration.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			inputs, err := setup.Load(cmd.Context(), options)
			if err != nil {
				return fmt.Errorf("setup failed: %w", err)
			}
			result, err := suite.Run(cmd.Context(), inputs, suite.Options{ReportDir: reportDirectory})
			if result.ReportDir != "" {
				cmd.Printf("Reports: %s\n", result.ReportDir)
			}
			if err != nil {
				return fmt.Errorf("suite failed: %w", err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(),
				"Executed %d conformance scenarios: %d passed, %d failed, %d skipped.\n",
				result.Executed, result.Passed, result.Failed, result.Skipped)
			if result.Executed == 0 {
				cmd.Println("No conformance checks were executed.")
			}
			return err
		},
	}
	flags := command.Flags()
	flags.StringVar(&reportDirectory, "report-dir", "conformance-results", "Directory for JUnit XML and Ginkgo JSON results")
	flags.StringVar(&options.Kubeconfig, "kubeconfig", "", "Kubeconfig file (defaults to KUBECONFIG or ~/.kube/config)")
	flags.StringVar(&options.ArtifactReference, "artifact-reference", "", "OCI image reference containing the OCM application artifact")
	flags.StringVar(&options.DeploymentTargetSpec, "deployment-target-spec", "", "YAML or JSON file containing DeploymentTarget.spec only")
	flags.StringArrayVar(&options.TargetSetup, "target-setup", nil, "Namespaced setup manifest file (repeatable; multi-document YAML supported)")
	flags.StringVar(&options.OCMRegistryConfig, "ocm-registry-config", "",
		"Kubernetes Secret manifest file for optional OCM registry credentials")
	// These names are defined immediately above, so marking them cannot fail.
	if err := command.MarkFlagRequired("artifact-reference"); err != nil {
		panic(err)
	}
	if err := command.MarkFlagRequired("deployment-target-spec"); err != nil {
		panic(err)
	}
	return command
}
