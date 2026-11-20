package suite

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"

	"github.com/konfidence-project/konfidence/internal/conformance/setup"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"ocm.software/open-component-model/bindings/go/credentials"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Report is the framework's complete record of a suite invocation.
type Report = types.Report

var frameworkStarted atomic.Bool

var errConformanceFailed = errors.New("conformance suite failed")

// Environment provides the baseline inputs and shared credentials to scenarios.
// It contains secrets and must not be serialized into test reports.
type Environment struct {
	Context             context.Context
	Inputs              *setup.Inputs
	Client              client.Client
	SharedNamespace     *corev1.Namespace
	RegistryCredentials credentials.Resolver
}

// frameworkTestingT adapts Ginkgo's failure notification to the CLI error path.
// Ginkgo handles assertion failures and panics inside specs.
type frameworkTestingT struct{}

func (frameworkTestingT) Fail() {}

func runFramework(environment *Environment, options Options, register func(*Environment)) (Result, error) {
	if err := environment.Context.Err(); err != nil {
		return Result{}, err
	}
	if frameworkStarted.Swap(true) {
		return Result{}, errors.New("only one Ginkgo suite invocation is supported per process; start a new conformance process")
	}
	directory := options.ReportDir
	if directory == "" {
		directory = "conformance-results"
	}
	directory, err := filepath.Abs(directory)
	if err != nil {
		return Result{}, err
	}
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.BeforeEach(func() {
		gomega.Expect(environment.Context.Err()).To(gomega.Succeed(), "suite execution was canceled")
	})
	register(environment)
	var report Report
	ginkgo.ReportAfterSuite("Capture conformance results", func(completed ginkgo.Report) {
		report = completed
	})
	suiteConfig, reporterConfig := ginkgo.GinkgoConfiguration()
	reporterConfig.JUnitReport = filepath.Join(directory, "junit.xml")
	reporterConfig.JSONReport = filepath.Join(directory, "report.json")
	passed := ginkgo.RunSpecs(frameworkTestingT{}, "Deployer conformance", suiteConfig, reporterConfig)
	specs := report.SpecReports.WithLeafNodeType(types.NodeTypeIt)
	result := Result{
		Passed:    specs.CountWithState(types.SpecStatePassed),
		Failed:    specs.CountWithState(types.SpecStateFailureStates),
		Skipped:   specs.CountWithState(types.SpecStateSkipped | types.SpecStatePending),
		ReportDir: directory,
	}
	result.Executed = result.Passed + result.Failed
	if !passed {
		return result, errConformanceFailed
	}
	return result, environment.Context.Err()
}
