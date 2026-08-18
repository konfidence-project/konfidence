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
// Best-effort by design: a failing cell does NOT cancel its siblings. Verify
// waits for every cell to finish, so each cell that passes reaches the inner
// cache and is memoized regardless of what others do; the FIRST error is
// returned only after the full fan-in. This is why it uses a zero-value
// errgroup.Group (which never cancels) and NOT errgroup.WithContext — do not
// "optimize" it to the latter, which would abort in-flight verifications on the
// first failure and lose their cache fills. External cancellation still
// propagates: the caller's ctx is passed straight to each cell's Limiter.Acquire
// and inner Verify, so a reconcile timeout or controller shutdown is respected.
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
