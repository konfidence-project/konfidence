package crypto

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"ocm.software/open-component-model/bindings/go/credentials"
	runtime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
)

var _ = Describe("VerifierBuilder", func() {
	Describe("Build", func() {
		It("returns a preFlightVerifier wrapping OCMVerifier by default", func() {
			v, err := NewVerifierBuilder().Build()
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(BeAssignableToTypeOf(&preFlightVerifier{}))
		})

		It("does not require a resolver or specs — those are supplied per Verify call", func() {
			v, err := NewVerifierBuilder().
				WithParallelism(NewLimiter(1)).
				Build()
			Expect(err).NotTo(HaveOccurred())
			Expect(v).NotTo(BeNil())
		})
	})
})

var _ = Describe("SignerBuilder", func() {
	Describe("Build", func() {
		It("returns a no-op signer when specs are empty", func() {
			s, err := NewSignerBuilder().Build()
			Expect(err).NotTo(HaveOccurred())
			Expect(s).NotTo(BeNil())
			// No specs → Sign is a no-op: it must not error and must not touch
			// the descriptor's signatures.
			desc := &runtime.Descriptor{}
			Expect(s.Sign(context.Background(), desc)).To(Succeed())
			Expect(desc.Signatures).To(BeEmpty())
		})

		It("returns an error when specs are non-empty but resolver is nil", func() {
			_, err := NewSignerBuilder().
				WithSpecs([]SignatureSpec{DefaultSignatureSpec("sig", nil)}).
				Build()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("resolver is required"))
		})

		It("returns an error for duplicate spec names", func() {
			_, err := NewSignerBuilder().
				WithSpecs([]SignatureSpec{
					DefaultSignatureSpec("dup", nil),
					DefaultSignatureSpec("dup", nil),
				}).
				WithResolver(credentials.NewStaticCredentialsResolver(nil)).
				Build()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`duplicate signature name detected: "dup"`))
		})
	})
})
