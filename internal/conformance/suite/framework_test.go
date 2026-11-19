package suite

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/reporters"
	"github.com/onsi/ginkgo/v2/types"
	"github.com/onsi/gomega"
)

const (
	fixtureFailure        = "failure"
	fixturePanic          = "panic"
	fixtureSetupFailure   = "setup-failure"
	fixtureCleanupFailure = "cleanup-failure"
	fixtureTimeout        = "timeout"
)

// TestFrameworkFixture runs in a separate process because Ginkgo owns a single
// suite tree per process. Fixtures exercise the real runner, not mocked reports.
func TestFrameworkFixture(t *testing.T) {
	fixture := os.Getenv("KDEN_CONFORMANCE_FRAMEWORK_FIXTURE")
	if fixture == "" {
		t.Skip("subprocess fixture")
	}
	directory := os.Getenv("KDEN_CONFORMANCE_FIXTURE_REPORT_DIR")
	result, runErr := runFramework(&Environment{Context: context.Background()}, Options{ReportDir: directory}, func(*Environment) {
		if fixture == fixtureSetupFailure {
			ginkgo.BeforeSuite(func() {
				gomega.Expect(false).To(gomega.BeTrue(), "fixture setup failed")
			})
		}
		if fixture == fixtureCleanupFailure {
			ginkgo.AfterSuite(func() {
				gomega.Expect(false).To(gomega.BeTrue(), "fixture cleanup failed")
			})
		}
		specTimeout := 5 * time.Second
		if fixture == fixtureTimeout {
			specTimeout = 50 * time.Millisecond
		}
		ginkgo.Describe("Fixture deployer", func() {
			ginkgo.It("checks the contract", func(ctx ginkgo.SpecContext) {
				switch fixture {
				case fixtureFailure:
					gomega.Expect("actual").To(gomega.Equal("expected"))
				case fixturePanic:
					panic("fixture panic")
				case "skipped":
					ginkgo.Skip("fixture is not applicable")
				case fixtureTimeout:
					<-ctx.Done()
				default:
					gomega.Expect(1 + 1).To(gomega.Equal(2))
				}
			}, ginkgo.SpecTimeout(specTimeout))
		})
	})
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(result.ReportDir, "result.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if runErr != nil {
		t.Fatal(runErr)
	}
}

func TestNativeReportWriteFailure(t *testing.T) {
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "junit.xml"), 0700); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(os.Args[0], "-test.run=^TestFrameworkFixture$")
	command.Env = append(os.Environ(), "KDEN_CONFORMANCE_FRAMEWORK_FIXTURE=success",
		"KDEN_CONFORMANCE_FIXTURE_REPORT_DIR="+directory)
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "Failed to generate JUnit report") {
		t.Fatalf("expected Ginkgo's native report-generation failure: %v\n%s", err, output)
	}
}

func TestFrameworkReports(t *testing.T) {
	for _, fixture := range []string{"success", fixtureFailure, fixturePanic, "skipped", fixtureSetupFailure, fixtureCleanupFailure, fixtureTimeout} {
		t.Run(fixture, func(t *testing.T) {
			directory := t.TempDir()
			command := exec.Command(os.Args[0], "-test.run=^TestFrameworkFixture$")
			command.Env = append(os.Environ(),
				"KDEN_CONFORMANCE_FRAMEWORK_FIXTURE="+fixture,
				"KDEN_CONFORMANCE_FIXTURE_REPORT_DIR="+directory,
			)
			output, err := command.CombinedOutput()
			shouldFail := fixture == fixtureFailure || fixture == fixturePanic ||
				fixture == fixtureSetupFailure || fixture == fixtureCleanupFailure || fixture == fixtureTimeout
			if (err != nil) != shouldFail {
				t.Fatalf("unexpected subprocess outcome: %v\n%s", err, output)
			}
			var reports []Report
			readJSON(t, filepath.Join(directory, "report.json"), &reports)
			if len(reports) != 1 || reports[0].SuiteSucceeded == shouldFail {
				t.Fatalf("unexpected suite status: %+v", reports)
			}
			var result Result
			readJSON(t, filepath.Join(directory, "result.json"), &result)
			switch fixture {
			case "success":
				if result.Executed != 1 || result.Passed != 1 || result.Failed != 0 {
					t.Fatalf("unexpected success counts: %+v", result)
				}
			case fixtureFailure, fixturePanic, fixtureTimeout:
				if result.Executed != 1 || result.Failed != 1 {
					t.Fatalf("unexpected failure counts: %+v", result)
				}
			case "skipped":
				if result.Executed != 0 || result.Skipped != 1 {
					t.Fatalf("unexpected skip counts: %+v", result)
				}
			case fixtureSetupFailure:
				if reports[0].SpecReports.WithLeafNodeType(types.NodeTypeBeforeSuite).
					CountWithState(types.SpecStateFailed) != 1 {
					t.Fatal("setup failure was not recorded")
				}
			case fixtureCleanupFailure:
				if reports[0].SpecReports.WithLeafNodeType(types.NodeTypeAfterSuite).
					CountWithState(types.SpecStateFailed) != 1 {
					t.Fatal("cleanup failure was not recorded by Ginkgo")
				}
			}
			data, err := os.ReadFile(filepath.Join(directory, "junit.xml"))
			if err != nil {
				t.Fatal(err)
			}
			var junit reporters.JUnitTestSuites
			if err := xml.Unmarshal(data, &junit); err != nil {
				t.Fatalf("invalid JUnit report: %v", err)
			}
			if len(junit.TestSuites) != 1 || (junit.Failures > 0 || junit.Errors > 0) != shouldFail {
				t.Fatalf("unexpected JUnit results: %+v", junit)
			}
		})
	}
}

func readJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, value); err != nil {
		t.Fatal(err)
	}
}
