package suite

import (
	"context"
	"testing"

	"github.com/konfidence-project/konfidence/internal/conformance/setup"
)

func TestEmptySuite(t *testing.T) {
	result, err := Run(context.Background(), &setup.Inputs{})
	if err != nil || result.Executed != 0 {
		t.Fatalf("unexpected empty suite result: %+v, %v", result, err)
	}
}

func TestCanceledSuite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Run(ctx, &setup.Inputs{}); err != context.Canceled {
		t.Fatalf("got %v, want cancellation", err)
	}
}
