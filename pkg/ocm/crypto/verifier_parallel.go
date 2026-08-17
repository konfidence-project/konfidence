package crypto

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
	"ocm.software/open-component-model/bindings/go/credentials"
	ocm "ocm.software/open-component-model/bindings/go/descriptor/runtime"
)

var _ Verifier = (*parallelVerifier)(nil)

// parallelVerifier is a Verifier decorator that fans out the spec × desc
// matrix concurrently, bounded by a process-wide Limiter.
//
// Each goroutine acquires the limiter before delegating to the inner Verifier.
// Sibling failures do not cancel each other — every cell that passes completes
// regardless of what others do (best-effort semantics). The caller's context
// still propagates, so external cancellation (reconcile timeout, controller
// shutdown) is respected.
type parallelVerifier struct {
	inner   Verifier
	limiter Limiter
}

func newParallelVerifier(inner Verifier, limiter Limiter) *parallelVerifier {
	if limiter == nil {
		limiter = NoopLimiter{}
	}
	return &parallelVerifier{
		inner:   inner,
		limiter: limiter,
	}
}

func (p *parallelVerifier) Verify(ctx context.Context, resolver credentials.Resolver, specs []SignatureSpec, descs []*ocm.Descriptor) error {
	if len(specs) == 0 || len(descs) == 0 {
		return nil
	}

	if len(specs) == 1 && len(descs) == 1 {
		return p.verifyCell(ctx, resolver, specs[0], descs[0])
	}
	var g errgroup.Group
	for _, desc := range descs {
		for _, spec := range specs {
			g.Go(func() error { return p.verifyCell(ctx, resolver, spec, desc) })
		}
	}
	return g.Wait()
}

// verifyCell acquires one limiter slot, then delegates a single (spec, desc)
// cell to the inner verifier. Shared by both the fast path and the fan-out path
// so every cell — regardless of batch shape — is bounded by the same budget.
func (p *parallelVerifier) verifyCell(ctx context.Context, resolver credentials.Resolver, spec SignatureSpec, desc *ocm.Descriptor) error {
	release, err := p.limiter.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("ocm descriptor verification failed: %w", err)
	}
	defer release()
	return p.inner.Verify(ctx, resolver, []SignatureSpec{spec}, []*ocm.Descriptor{desc})
}
