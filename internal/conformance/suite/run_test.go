package suite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/konfidence-project/konfidence/internal/conformance/setup"
)

func TestEmptySuite(t *testing.T) {
	result, err := Run(context.Background(), &setup.Inputs{}, Options{ReportDir: t.TempDir()})
	if err != nil || result.Executed != 0 {
		t.Fatalf("unexpected empty suite result: %+v, %v", result, err)
	}
	var reports []Report
	readJSON(t, filepath.Join(result.ReportDir, "report.json"), &reports)
	if len(reports) != 1 || reports[0].SuiteDescription != "Deployer conformance" || !reports[0].SuiteSucceeded {
		t.Fatalf("unexpected framework report: %+v", reports)
	}
}

func TestCanceledSuite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Run(ctx, &setup.Inputs{}, Options{ReportDir: t.TempDir()}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want cancellation", err)
	}
}
