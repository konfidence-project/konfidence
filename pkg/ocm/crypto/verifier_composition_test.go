package crypto

import (
	"context"
	"log/slog"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ocm "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/signing"
)

// buildChain assembles the same decorator stack that VerifierBuilder.Build
// composes (preFlight → parallel → cache → inner), but around a caller-supplied
// inner Verifier so the test can observe crypto invocations. Passing size==0
// skips the cache layer; limiter==nil skips the parallel layer — mirroring the
// builder's opt-in composition.
func buildChain(inner Verifier, limiter Limiter, cacheSize int, cacheTTL time.Duration) Verifier {
	v := inner
	if cacheSize > 0 {
		v = newCachingVerifier(v, cacheSize, cacheTTL)
	}
	if limiter != nil {
		v = newParallelVerifier(v, limiter)
	}
	return newPreFlightVerifier(v)
}

// These tests target the way the decorator layers compose (preFlight → parallel
// → cache → ocm), not any single decorator in isolation — the individual
// decorators are covered in their own *_test.go files. Here we prove the
// composition is correct: the failure mode the isolated unit tests cannot see
// (a layer silently dropped or reordered).
var _ = Describe("Verifier composition", func() {
	var specs = []SignatureSpec{specFor("s1")}

	BeforeEach(func() {
		// Bare descriptors must pass the cache pre-check.
		verifyDigestMatchesDescriptor = func(_ context.Context, _ *ocm.Descriptor, _ ocm.Signature, _ *slog.Logger) error {
			return nil
		}
	})
	AfterEach(func() {
		verifyDigestMatchesDescriptor = signing.VerifyDigestMatchesDescriptor
	})

	Describe("cache layer", func() {
		It("with a cache, the same descriptor verified twice runs inner once", func() {
			stub := newStub(nil)
			v := buildChain(stub, nil, 16, time.Minute)
			desc := descWith(sig("s1", "abc"))

			Expect(v.Verify(context.Background(), nil, specs, []*ocm.Descriptor{desc})).To(Succeed())
			Expect(v.Verify(context.Background(), nil, specs, []*ocm.Descriptor{desc})).To(Succeed())
			Expect(stub.calls.Load()).To(Equal(int64(1)), "second Verify must hit the cache, not re-run inner")
		})

		It("without a cache, every Verify re-runs inner", func() {
			stub := newStub(nil)
			v := buildChain(stub, nil, 0, 0) // no cache
			desc := descWith(sig("s1", "abc"))

			Expect(v.Verify(context.Background(), nil, specs, []*ocm.Descriptor{desc})).To(Succeed())
			Expect(v.Verify(context.Background(), nil, specs, []*ocm.Descriptor{desc})).To(Succeed())
			Expect(stub.calls.Load()).To(Equal(int64(2)), "no cache layer → inner runs each call")
		})
	})

	Describe("layer ordering: parallel wraps cache", func() {
		It("the cache saves crypto but NOT the limiter slot — parallel sits outside cache", func() {
			// Stack order is parallel(cache(inner)). On a fan-out call the
			// limiter is acquired per cell BEFORE the cache is consulted, so a
			// cache hit still pays the (cheap) Acquire; what it saves is the
			// (expensive) inner crypto. This asserts that ordering directly.
			//
			// A 2-cell batch is required: with a single cell parallelVerifier
			// takes its fast path and never touches the limiter, which would
			// make this assertion vacuous.
			stub := newStub(nil)
			lim := newCountingLimiter(NewLimiter(4))
			v := buildChain(stub, lim, 16, time.Minute)
			descs := []*ocm.Descriptor{descWith(sig("s1", "a")), descWith(sig("s1", "b"))}

			// First call: both cells miss → inner runs twice, limiter acquired twice.
			Expect(v.Verify(context.Background(), nil, specs, descs)).To(Succeed())
			Expect(stub.calls.Load()).To(Equal(int64(2)))
			Expect(lim.acquires.Load()).To(Equal(int64(2)))

			// Second call: both cells hit the cache → inner does NOT run again,
			// but the limiter IS acquired again because parallel is outside cache.
			Expect(v.Verify(context.Background(), nil, specs, descs)).To(Succeed())
			Expect(stub.calls.Load()).To(Equal(int64(2)), "cache hit must save the inner crypto")
			Expect(lim.acquires.Load()).To(Equal(int64(4)), "limiter is acquired per cell even on hits — parallel wraps cache")
		})
	})

	Describe("preFlight is the outermost layer", func() {
		It("rejects duplicate spec names without invoking inner crypto", func() {
			stub := newStub(nil)
			v := buildChain(stub, NewLimiter(4), 16, time.Minute)
			desc := descWith(sig("dup", "abc"))
			dupSpecs := []SignatureSpec{specFor("dup"), specFor("dup")}

			err := v.Verify(context.Background(), nil, dupSpecs, []*ocm.Descriptor{desc})
			Expect(err).To(MatchError(ContainSubstring(`duplicate signature name detected: "dup"`)))
			Expect(stub.calls.Load()).To(Equal(int64(0)), "preFlight must short-circuit before inner crypto")
		})
	})

	Describe("VerifierBuilder.Build composes the expected top layer", func() {
		It("returns a preFlightVerifier for every option combination", func() {
			// Guards against Build forgetting the outermost validation layer.
			for _, cfg := range []func() (Verifier, error){
				func() (Verifier, error) { return NewVerifierBuilder().Build() },
				func() (Verifier, error) { return NewVerifierBuilder().WithParallelism(NewLimiter(1)).Build() },
				func() (Verifier, error) { return NewVerifierBuilder().WithCache(16, time.Minute).Build() },
				func() (Verifier, error) {
					return NewVerifierBuilder().WithParallelism(NewLimiter(1)).WithCache(16, time.Minute).Build()
				},
			} {
				v, err := cfg()
				Expect(err).NotTo(HaveOccurred())
				Expect(v).To(BeAssignableToTypeOf(&preFlightVerifier{}))
			}
		})
	})
})
