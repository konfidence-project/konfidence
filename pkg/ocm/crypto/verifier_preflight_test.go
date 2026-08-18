package crypto

import (
	"context"
	"fmt"
	"log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ocm "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/signing"
)

// These tests target preFlightVerifier in isolation: it validates the
// SignatureSpecs and, on any violation, must reject BEFORE delegating to the
// inner Verifier. Wrapping a stub inner lets us assert that short-circuit
// directly (inner call count stays 0) — something the chain/OCMVerifier tests
// cannot show.
var _ = Describe("preFlightVerifier", func() {
	var desc *ocm.Descriptor

	BeforeEach(func() {
		verifyDigestMatchesDescriptor = func(_ context.Context, _ *ocm.Descriptor, _ ocm.Signature, _ *slog.Logger) error {
			return nil
		}
		isSafelyDigestible = func(_ *ocm.Component) error { return nil }
		desc = descWith(sig("sig1", "abc"))
	})
	AfterEach(func() {
		verifyDigestMatchesDescriptor = signing.VerifyDigestMatchesDescriptor
		isSafelyDigestible = signing.IsSafelyDigestible
	})

	It("delegates to inner when specs are valid", func() {
		stub := newStub(nil)
		v := newPreFlightVerifier(stub)

		Expect(v.Verify(context.Background(), nil, []SignatureSpec{specFor("sig1")}, []*ocm.Descriptor{desc})).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(1)), "valid specs must reach the inner verifier")
	})

	It("rejects duplicate spec names without delegating to inner", func() {
		stub := newStub(nil)
		v := newPreFlightVerifier(stub)

		err := v.Verify(context.Background(), nil, []SignatureSpec{specFor("sig1"), specFor("sig1")}, []*ocm.Descriptor{desc})
		Expect(err).To(MatchError(ContainSubstring(`duplicate signature name detected: "sig1"`)))
		Expect(stub.calls.Load()).To(Equal(int64(0)), "must short-circuit before inner crypto")
	})

	It("rejects an empty/whitespace spec name without delegating to inner", func() {
		stub := newStub(nil)
		v := newPreFlightVerifier(stub)

		err := v.Verify(context.Background(), nil, []SignatureSpec{{Name: "  "}}, []*ocm.Descriptor{desc})
		Expect(err).To(MatchError(ContainSubstring("signature names cannot be empty or whitespace")))
		Expect(stub.calls.Load()).To(Equal(int64(0)))
	})

	It("rejects a non-nil empty issuer pin without delegating to inner", func() {
		stub := newStub(nil)
		v := newPreFlightVerifier(stub)
		empty := ""
		spec := DefaultSignatureSpec("sig1", &empty)

		err := v.Verify(context.Background(), nil, []SignatureSpec{spec}, []*ocm.Descriptor{desc})
		Expect(err).To(MatchError(ContainSubstring(`issuer pin for "sig1" must not be empty`)))
		Expect(stub.calls.Load()).To(Equal(int64(0)))
	})

	It("is a no-op for empty specs — inner is never called", func() {
		stub := newStub(nil)
		v := newPreFlightVerifier(stub)

		Expect(v.Verify(context.Background(), nil, nil, []*ocm.Descriptor{desc})).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(0)))
	})

	It("is a no-op for empty descriptors — inner is never called", func() {
		stub := newStub(nil)
		v := newPreFlightVerifier(stub)

		Expect(v.Verify(context.Background(), nil, []SignatureSpec{specFor("sig1")}, nil)).To(Succeed())
		Expect(stub.calls.Load()).To(Equal(int64(0)))
	})

	It("rejects a not-safely-digestible descriptor without delegating to inner", func() {
		// The safe-digestibility gate lives here, at the top of the chain — run
		// once per batch, before any fan-out, caching, or crypto.
		stub := newStub(nil)
		v := newPreFlightVerifier(stub)
		isSafelyDigestible = func(_ *ocm.Component) error { return fmt.Errorf("not safely digestible") }

		err := v.Verify(context.Background(), nil, []SignatureSpec{specFor("sig1")}, []*ocm.Descriptor{desc})
		Expect(err).To(MatchError(ContainSubstring("descriptor is not safely digestible: not safely digestible")))
		Expect(stub.calls.Load()).To(Equal(int64(0)), "must short-circuit before any inner layer")
	})

	It("rejects the whole batch if any descriptor is not safely digestible", func() {
		// The gate covers every descriptor in the batch, not just the first.
		stub := newStub(nil)
		v := newPreFlightVerifier(stub)
		good := descWith(sig("sig1", "good"))
		bad := descWith(sig("sig1", "bad"))
		isSafelyDigestible = func(c *ocm.Component) error {
			if c == &bad.Component {
				return fmt.Errorf("not safely digestible")
			}
			return nil
		}

		err := v.Verify(context.Background(), nil, []SignatureSpec{specFor("sig1")}, []*ocm.Descriptor{good, bad})
		Expect(err).To(MatchError(ContainSubstring("not safely digestible")))
		Expect(stub.calls.Load()).To(Equal(int64(0)), "a single bad descriptor rejects the batch before delegation")
	})
})
