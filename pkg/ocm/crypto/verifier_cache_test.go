package crypto

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ocm.software/open-component-model/bindings/go/credentials"
	ocm "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/signing"
)

const (
	testCacheSize = 1024
	testCacheTTL  = 30 * time.Minute
)

// stubVerifier is a minimal Verifier that counts calls and returns a
// preconfigured error (nil = success) per descriptor pointer. It matches the
// current Verifier interface: (ctx, resolver, specs, descs).
type stubVerifier struct {
	calls   atomic.Int64
	results map[*ocm.Descriptor]error
}

func (s *stubVerifier) Verify(_ context.Context, _ credentials.Resolver, _ []SignatureSpec, descs []*ocm.Descriptor) error {
	s.calls.Add(1)
	for _, desc := range descs {
		if err, ok := s.results[desc]; ok {
			return err
		}
	}
	return nil
}

func newStub(results map[*ocm.Descriptor]error) *stubVerifier {
	return &stubVerifier{results: results}
}

// sig builds a minimal Signature with the given value — enough to produce a
// unique cache key without setting every field.
//
//nolint:unparam
func sig(name, value string) ocm.Signature {
	return ocm.Signature{
		Name: name,
		Digest: ocm.Digest{
			HashAlgorithm:          "SHA-256",
			NormalisationAlgorithm: "jsonNormalisation/v4alpha1",
			Value:                  value,
		},
		Signature: ocm.SignatureInfo{
			Algorithm: "RSASSA-PSS",
			MediaType: "application/x-pem-file",
			Value:     value + "-sig",
		},
	}
}

func descWith(sigs ...ocm.Signature) *ocm.Descriptor {
	return &ocm.Descriptor{Signatures: sigs}
}

// specFor returns a SignatureSpec whose fields line up with sig(name, _).
func specFor(name string) SignatureSpec { return DefaultSignatureSpec(name, nil) }

var _ = Describe("CachingVerifier", func() {
	var (
		stub  *stubVerifier
		cache *cachingVerifier
		specs = []SignatureSpec{specFor("s1")}
	)

	BeforeEach(func() {
		stub = newStub(nil)
		cache = newCachingVerifier(stub, testCacheSize, testCacheTTL)
		// stub out digest verification so bare test descriptors pass the pre-check
		verifyDigestMatchesDescriptor = func(_ context.Context, _ *ocm.Descriptor, _ ocm.Signature, _ *slog.Logger) error {
			return nil
		}
	})

	AfterEach(func() {
		verifyDigestMatchesDescriptor = signing.VerifyDigestMatchesDescriptor
	})

	It("calls inner on first verify", func() {
		desc := descWith(sig("s1", "abc"))
		Expect(cache.Verify(context.Background(), nil, specs, []*ocm.Descriptor{desc})).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(1)))
	})

	It("does not call inner on cache hit", func() {
		desc := descWith(sig("s1", "abc"))
		Expect(cache.Verify(context.Background(), nil, specs, []*ocm.Descriptor{desc})).To(Succeed())
		Expect(cache.Verify(context.Background(), nil, specs, []*ocm.Descriptor{desc})).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(1)))
	})

	It("calls inner again for a different descriptor", func() {
		desc1 := descWith(sig("s1", "abc"))
		desc2 := descWith(sig("s1", "xyz"))
		Expect(cache.Verify(context.Background(), nil, specs, []*ocm.Descriptor{desc1})).To(Succeed())
		Expect(cache.Verify(context.Background(), nil, specs, []*ocm.Descriptor{desc2})).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(2)))
	})

	It("does not cache failures", func() {
		boom := fmt.Errorf("bad sig")
		desc := descWith(sig("s1", "abc"))
		stub = newStub(map[*ocm.Descriptor]error{desc: boom})
		cache = newCachingVerifier(stub, testCacheSize, testCacheTTL)

		Expect(cache.Verify(context.Background(), nil, specs, []*ocm.Descriptor{desc})).To(MatchError(boom))
		Expect(cache.Verify(context.Background(), nil, specs, []*ocm.Descriptor{desc})).To(MatchError(boom))
		Expect(stub.calls.Load()).To(Equal(int64(2)))
	})

	It("caches successful cells even when a sibling fails", func() {
		good := descWith(sig("s1", "good"))
		bad := descWith(sig("s1", "bad"))
		boom := fmt.Errorf("bad sig")
		stub = newStub(map[*ocm.Descriptor]error{bad: boom})
		cache = newCachingVerifier(stub, testCacheSize, testCacheTTL)

		err := cache.Verify(context.Background(), nil, specs, []*ocm.Descriptor{good, bad})
		Expect(err).To(MatchError(boom))

		// good was cached — inner must not be called again for it
		callsBefore := stub.calls.Load()
		Expect(cache.Verify(context.Background(), nil, specs, []*ocm.Descriptor{good})).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(callsBefore))
	})

	It("returns the first error and stops", func() {
		desc1 := descWith(sig("s1", "d1"))
		desc2 := descWith(sig("s1", "d2"))
		desc3 := descWith(sig("s1", "d3"))
		boom := fmt.Errorf("d2 failed")
		stub = newStub(map[*ocm.Descriptor]error{desc2: boom})
		cache = newCachingVerifier(stub, testCacheSize, testCacheTTL)

		err := cache.Verify(context.Background(), nil, specs, []*ocm.Descriptor{desc1, desc2, desc3})
		Expect(err).To(MatchError(boom))
		Expect(stub.calls.Load()).To(Equal(int64(2)))
	})

	It("evicts entries after TTL", func() {
		ttl := 50 * time.Millisecond
		cache = newCachingVerifier(stub, testCacheSize, ttl)
		desc := descWith(sig("s1", "abc"))

		Expect(cache.Verify(context.Background(), nil, specs, []*ocm.Descriptor{desc})).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(1)))

		time.Sleep(ttl + 20*time.Millisecond)

		Expect(cache.Verify(context.Background(), nil, specs, []*ocm.Descriptor{desc})).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(2)))
	})

	It("re-verifies all entries after Flush", func() {
		desc := descWith(sig("s1", "abc"))
		Expect(cache.Verify(context.Background(), nil, specs, []*ocm.Descriptor{desc})).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(1)))

		cache.Flush()

		Expect(cache.Verify(context.Background(), nil, specs, []*ocm.Descriptor{desc})).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(2)))
	})

	It("empty specs is a no-op", func() {
		desc := descWith(sig("s1", "abc"))
		Expect(cache.Verify(context.Background(), nil, nil, []*ocm.Descriptor{desc})).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(0)))
	})

	It("empty descs is a no-op", func() {
		Expect(cache.Verify(context.Background(), nil, specs, nil)).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(0)))
	})

	It("respects caller context cancellation", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		cache = newCachingVerifier(&ctxCheckVerifier{}, testCacheSize, testCacheTTL)
		desc := descWith(sig("s1", "abc"))
		Expect(cache.Verify(ctx, nil, specs, []*ocm.Descriptor{desc})).To(MatchError(context.Canceled))
	})

	It("rejects replay: same signature struct on tampered descriptor fails pre-check", func() {
		// restore real verifyDigestMatchesDescriptor for this test
		verifyDigestMatchesDescriptor = signing.VerifyDigestMatchesDescriptor

		validSig := sig("s1", "abc")
		tampered := &ocm.Descriptor{
			Component: ocm.Component{
				ComponentMeta: ocm.ComponentMeta{
					ObjectMeta: ocm.ObjectMeta{Name: "evil", Version: "1.0.0"},
				},
				Provider: ocm.Provider{Name: "evil-provider"},
			},
			Signatures: []ocm.Signature{validSig},
		}

		Expect(cache.Verify(context.Background(), nil, specs, []*ocm.Descriptor{tampered})).To(
			MatchError(ContainSubstring("cache pre-check digest mismatch")))
		Expect(stub.calls.Load()).To(Equal(int64(0)))
	})

	It("rejects weak hash algorithm in pre-check", func() {
		verifyDigestMatchesDescriptor = signing.VerifyDigestMatchesDescriptor

		weakDesc := descWith(ocm.Signature{
			Name: "s1",
			Digest: ocm.Digest{
				HashAlgorithm:          "MD5",
				NormalisationAlgorithm: "jsonNormalisation/v4alpha1",
				Value:                  "deadbeef",
			},
			Signature: ocm.SignatureInfo{
				Algorithm: "RSASSA-PSS",
				MediaType: "application/x-pem-file",
				Value:     "deadbeef-sig",
			},
		})
		Expect(cache.Verify(context.Background(), nil, specs, []*ocm.Descriptor{weakDesc})).To(
			MatchError(ContainSubstring("cache pre-check digest mismatch")))
		Expect(stub.calls.Load()).To(Equal(int64(0)))
	})

	It("distinct spec fingerprints produce distinct cache entries", func() {
		// Same signature bytes on the same descriptor, but two specs differing
		// only in issuer pin, must yield two independent verifies. Otherwise a
		// verdict cached under a lax spec would leak to a caller who pins.
		desc := descWith(sig("s1", "same-bytes"))
		pinned := "CN=corp-ca"
		lax := DefaultSignatureSpec("s1", nil)
		strict := DefaultSignatureSpec("s1", &pinned)

		Expect(cache.Verify(context.Background(), nil, []SignatureSpec{lax}, []*ocm.Descriptor{desc})).To(Succeed())
		Expect(cache.Verify(context.Background(), nil, []SignatureSpec{strict}, []*ocm.Descriptor{desc})).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(2)))
	})

	It("hit is served regardless of resolver identity — resolver is not part of the key", func() {
		// Resolver identity is not part of the cache key: the cached crypto
		// verdict is a statement about descriptor bytes ↔ signature bytes; the
		// resolver is only the delivery channel for pubkey material. Two
		// different resolvers presenting the same signature bytes must hit.
		desc := descWith(sig("s1", "abc"))
		r1 := credentials.NewStaticCredentialsResolver(nil)
		r2 := credentials.NewStaticCredentialsResolver(nil)

		Expect(cache.Verify(context.Background(), r1, specs, []*ocm.Descriptor{desc})).To(Succeed())
		Expect(cache.Verify(context.Background(), r2, specs, []*ocm.Descriptor{desc})).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(1)))
	})
})

// ctxCheckVerifier returns ctx.Err() if the context is done, simulating how
// OCMVerifier propagates cancellation through Limiter.Acquire.
type ctxCheckVerifier struct{}

func (v *ctxCheckVerifier) Verify(ctx context.Context, _ credentials.Resolver, _ []SignatureSpec, _ []*ocm.Descriptor) error {
	return ctx.Err()
}
