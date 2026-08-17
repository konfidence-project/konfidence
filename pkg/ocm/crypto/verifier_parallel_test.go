package crypto

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ocm "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/signing"
)

// countingLimiter wraps an inner Limiter and records how many times Acquire was
// called and the peak number of simultaneously-held slots. It lets a test prove
// (a) that a code path actually goes through the limiter, and (b) that
// concurrency never exceeds the configured bound. Shared across the crypto
// test suite (package-level).
type countingLimiter struct {
	inner    Limiter
	acquires atomic.Int64
	mu       sync.Mutex
	held     int
	peak     int
}

func newCountingLimiter(inner Limiter) *countingLimiter {
	return &countingLimiter{inner: inner}
}

func (l *countingLimiter) Acquire(ctx context.Context) (func(), error) {
	release, err := l.inner.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	l.acquires.Add(1)
	l.mu.Lock()
	l.held++
	if l.held > l.peak {
		l.peak = l.held
	}
	l.mu.Unlock()
	return func() {
		l.mu.Lock()
		l.held--
		l.mu.Unlock()
		release()
	}, nil
}

func (l *countingLimiter) peakHeld() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.peak
}

var _ = Describe("parallelVerifier", func() {
	var specs = []SignatureSpec{specFor("s1")}

	BeforeEach(func() {
		verifyDigestMatchesDescriptor = func(_ context.Context, _ *ocm.Descriptor, _ ocm.Signature, _ *slog.Logger) error {
			return nil
		}
	})
	AfterEach(func() {
		verifyDigestMatchesDescriptor = signing.VerifyDigestMatchesDescriptor
	})

	It("verifies every cell of the matrix and bounds concurrency to the limiter", func() {
		// A tight limiter (1) over a many-descriptor batch: every cell must
		// still complete, and peak concurrency must never exceed the bound.
		stub := newStub(nil)
		lim := newCountingLimiter(NewLimiter(1))
		p := newParallelVerifier(stub, lim)

		descs := []*ocm.Descriptor{
			descWith(sig("s1", "a")),
			descWith(sig("s1", "b")),
			descWith(sig("s1", "c")),
			descWith(sig("s1", "d")),
		}
		Expect(p.Verify(context.Background(), nil, specs, descs)).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(4)), "every cell must be verified")
		Expect(lim.peakHeld()).To(Equal(1), "concurrency must not exceed the limiter bound")
		Expect(lim.acquires.Load()).To(Equal(int64(4)), "each cell acquires one slot")
	})

	It("returns the first error but still verifies remaining cells (best-effort)", func() {
		desc1 := descWith(sig("s1", "d1"))
		desc2 := descWith(sig("s1", "d2"))
		desc3 := descWith(sig("s1", "d3"))
		boom := fmt.Errorf("d2 failed")
		stub := newStub(map[*ocm.Descriptor]error{desc2: boom})
		p := newParallelVerifier(stub, NoopLimiter{})

		err := p.Verify(context.Background(), nil, specs, []*ocm.Descriptor{desc1, desc2, desc3})
		Expect(err).To(MatchError(boom))
		Expect(stub.calls.Load()).To(Equal(int64(3)), "sibling failure must not cancel remaining cells")
	})

	It("propagates limiter acquisition failure from a cancelled context", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		stub := newStub(nil)
		p := newParallelVerifier(stub, NewLimiter(1))
		descs := []*ocm.Descriptor{descWith(sig("s1", "a")), descWith(sig("s1", "b"))}

		Expect(p.Verify(ctx, nil, specs, descs)).To(MatchError(context.Canceled))
	})

	It("single-cell fast path still acquires the limiter (bounded, no errgroup)", func() {
		// The 1×1 fast path skips the errgroup allocation but must still draw
		// from the shared crypto budget — a single RSA verify is not exempt.
		// This is the invariant that prevents many concurrent single-descriptor
		// callers (e.g. vectorassembly's async per-CR GetVector goroutines) from
		// collectively exceeding GOMAXPROCS.
		stub := newStub(nil)
		lim := newCountingLimiter(NewLimiter(4))
		p := newParallelVerifier(stub, lim)
		desc := descWith(sig("s1", "a"))

		Expect(p.Verify(context.Background(), nil, specs, []*ocm.Descriptor{desc})).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(1)))
		Expect(lim.acquires.Load()).To(Equal(int64(1)), "single cell must acquire exactly one limiter slot")
	})

	It("single-cell fast path propagates limiter acquisition failure", func() {
		// With the fast path now bounded, a cancelled context must fail at
		// Acquire before any inner crypto runs.
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		stub := newStub(nil)
		p := newParallelVerifier(stub, NewLimiter(1))
		desc := descWith(sig("s1", "a"))

		Expect(p.Verify(ctx, nil, specs, []*ocm.Descriptor{desc})).To(MatchError(context.Canceled))
		Expect(stub.calls.Load()).To(Equal(int64(0)), "must fail at Acquire before inner crypto")
	})
})
