package suite

import "github.com/onsi/ginkgo/v2"

// registerScenarios defines the deployer contract checks using the baseline
// environment. Scenario definitions live in regular Go files so release binaries
// include them without requiring Go or the Ginkgo CLI at execution time.
func registerScenarios(environment *Environment) {
	ginkgo.Describe("Deployer conformance", func() {
		setup := registerScenarioSetup(environment)
		registerDeploymentTargetScenarios(environment, setup)
	})
}
