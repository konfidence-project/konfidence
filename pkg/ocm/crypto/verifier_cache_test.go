package crypto

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ocm "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/signing"
)

// stubVerifier is a minimal Verifier that counts calls and returns a
// preconfigured error (nil = success) per descriptor pointer.
type stubVerifier struct {
	calls   atomic.Int64
	results map[*ocm.Descriptor]error
}

func (s *stubVerifier) Verify(_ context.Context, descs ...*ocm.Descriptor) error {
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

var _ = Describe("CachingVerifier", func() {
	var (
		stub  *stubVerifier
		cache *CachingVerifier
	)

	BeforeEach(func() {
		stub = newStub(nil)
		cache = NewCachingVerifier(stub, DefaultVerifierCacheSize, DefaultVerifierCacheTTL)
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
		Expect(cache.Verify(context.Background(), desc)).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(1)))
	})

	It("does not call inner on cache hit", func() {
		desc := descWith(sig("s1", "abc"))
		Expect(cache.Verify(context.Background(), desc)).To(Succeed())
		Expect(cache.Verify(context.Background(), desc)).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(1)))
	})

	It("calls inner again for a different descriptor", func() {
		desc1 := descWith(sig("s1", "abc"))
		desc2 := descWith(sig("s1", "xyz"))
		Expect(cache.Verify(context.Background(), desc1)).To(Succeed())
		Expect(cache.Verify(context.Background(), desc2)).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(2)))
	})

	It("does not cache failures", func() {
		boom := fmt.Errorf("bad sig")
		desc := descWith(sig("s1", "abc"))
		stub = newStub(map[*ocm.Descriptor]error{desc: boom})
		cache = NewCachingVerifier(stub, DefaultVerifierCacheSize, DefaultVerifierCacheTTL)

		Expect(cache.Verify(context.Background(), desc)).To(MatchError(boom))
		Expect(cache.Verify(context.Background(), desc)).To(MatchError(boom))
		Expect(stub.calls.Load()).To(Equal(int64(2)))
	})

	It("caches successful descriptors even when a sibling fails", func() {
		good := descWith(sig("s1", "good"))
		bad := descWith(sig("s1", "bad"))
		boom := fmt.Errorf("bad sig")
		stub = newStub(map[*ocm.Descriptor]error{bad: boom})
		cache = NewCachingVerifier(stub, DefaultVerifierCacheSize, DefaultVerifierCacheTTL)

		err := cache.Verify(context.Background(), good, bad)
		Expect(err).To(MatchError(boom))

		// good was cached — inner must not be called again for it
		callsBefore := stub.calls.Load()
		Expect(cache.Verify(context.Background(), good)).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(callsBefore))
	})

	It("returns the first error but verifies remaining descriptors", func() {
		desc1 := descWith(sig("s1", "d1"))
		desc2 := descWith(sig("s1", "d2"))
		desc3 := descWith(sig("s1", "d3"))
		boom := fmt.Errorf("d2 failed")
		stub = newStub(map[*ocm.Descriptor]error{desc2: boom})
		cache = NewCachingVerifier(stub, DefaultVerifierCacheSize, DefaultVerifierCacheTTL)

		Expect(cache.Verify(context.Background(), desc1, desc2, desc3)).To(MatchError(boom))
		Expect(stub.calls.Load()).To(Equal(int64(3)))
	})

	It("evicts entries after TTL", func() {
		ttl := 50 * time.Millisecond
		cache = NewCachingVerifier(stub, DefaultVerifierCacheSize, ttl)
		desc := descWith(sig("s1", "abc"))

		Expect(cache.Verify(context.Background(), desc)).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(1)))

		time.Sleep(ttl + 20*time.Millisecond)

		Expect(cache.Verify(context.Background(), desc)).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(2)))
	})

	It("re-verifies all entries after Flush", func() {
		desc := descWith(sig("s1", "abc"))
		Expect(cache.Verify(context.Background(), desc)).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(1)))

		cache.Flush()

		Expect(cache.Verify(context.Background(), desc)).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(2)))
	})

	It("empty batch is a no-op", func() {
		Expect(cache.Verify(context.Background())).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(0)))
	})

	It("respects caller context cancellation", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		cache = NewCachingVerifier(&ctxCheckVerifier{}, DefaultVerifierCacheSize, DefaultVerifierCacheTTL)
		desc := descWith(sig("s1", "abc"))
		Expect(cache.Verify(ctx, desc)).To(MatchError(context.Canceled))
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

		Expect(cache.Verify(context.Background(), tampered)).To(
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
		Expect(cache.Verify(context.Background(), weakDesc)).To(
			MatchError(ContainSubstring("cache pre-check digest mismatch")))
		Expect(stub.calls.Load()).To(Equal(int64(0)))
	})
})

// ctxCheckVerifier returns ctx.Err() if the context is done, simulating how
// OCMVerifier propagates cancellation through Limiter.Acquire.
type ctxCheckVerifier struct{}

func (v *ctxCheckVerifier) Verify(ctx context.Context, _ ...*ocm.Descriptor) error {
	return ctx.Err()
}
