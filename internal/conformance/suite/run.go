// Package suite owns conformance execution, not CLI parsing or credential loading.
package suite

import (
	"context"

	"github.com/konfidence-project/konfidence/internal/conformance/setup"
)

// Result summarizes conformance scenario execution and its report artifacts.
type Result struct {
	Executed  int
	Passed    int
	Failed    int
	Skipped   int
	ReportDir string
}

// Options configures the output of a suite invocation.
type Options struct {
	ReportDir string
}

// Run prepares shared registry Secrets when supplied, then executes scenarios.
// Per-case target setup resources are separate from these run-scoped credentials.
// Ginkgo executes one suite per process; callers must start a new process for
// another invocation.
func Run(ctx context.Context, inputs *setup.Inputs, options Options) (Result, error) {
	return runFramework(&Environment{Context: ctx, Inputs: inputs}, options, func(environment *Environment) {
		registerLifecycle(environment)
		registerScenarios(environment)
	})
}
