package crypto

import (
	"context"
	"fmt"
	"runtime"

	"golang.org/x/sync/semaphore"
)

// Limiter bounds concurrent CPU-bound cryptographic work — descriptor normalisation,
// hashing, and RSA sign/verify — to a fixed number of goroutines.
//
// One Limiter is intended to be constructed once per process and shared by every
// Signer and Verifier. Crypto is the only meaningful CPU-bound work the controllers
// do; the rest of the reconcile loop is K8s API calls and OCI registry I/O. Capping
// in-flight crypto operations at GOMAXPROCS prevents oversubscription when multiple
// controllers reconcile concurrently — without throttling unrelated reconcile work.
//
// Acquire returns a release function and an error. Callers MUST call release when
// the protected work completes (typically via defer). Acquire respects ctx
// cancellation and returns ctx.Err() if the context is cancelled before a slot
// is available.
type Limiter interface {
	Acquire(ctx context.Context) (release func(), err error)
}

// NewLimiter returns a Limiter that allows at most n concurrent acquisitions.
// If n <= 0, runtime.GOMAXPROCS(0) is used.
func NewLimiter(n int) Limiter {
	if n <= 0 {
		n = runtime.GOMAXPROCS(0)
	}
	return &weightedLimiter{sem: semaphore.NewWeighted(int64(n))}
}

type weightedLimiter struct {
	sem *semaphore.Weighted
}

func (l *weightedLimiter) Acquire(ctx context.Context) (func(), error) {
	if err := l.sem.Acquire(ctx, 1); err != nil {
		return nil, fmt.Errorf("acquire crypto limiter slot: %w", err)
	}
	return func() { l.sem.Release(1) }, nil
}

// NoopLimiter is the zero-value safe default: every Acquire succeeds immediately
// and the returned release is a no-op. Useful for tests, CLI tools, and any
// caller that opts out of bounding (single-threaded contexts where the limiter
// would only add overhead).
//
// To preserve substitutability with NewLimiter under the Limiter contract, a
// cancelled context still yields ctx.Err() — code paths that bail early on a
// dead context behave the same regardless of which limiter is in use.
type NoopLimiter struct{}

// Acquire returns immediately with a no-op release, unless ctx is already
// cancelled in which case it returns ctx.Err() and a nil release.
func (NoopLimiter) Acquire(ctx context.Context) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return func() {}, nil
}
