package crypto

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"ocm.software/open-component-model/bindings/go/credentials"
)

var _ = Describe("VerifierBuilder", func() {
	Describe("Build", func() {
		It("returns NoopVerifier when specs are empty", func() {
			v, err := NewVerifierBuilder().Build()
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(BeAssignableToTypeOf(NoopVerifier{}))
		})

		It("returns a CachingVerifier for non-empty specs", func() {
			v, err := NewVerifierBuilder().
				WithSpecs([]SignatureSpec{DefaultSignatureSpec("sig", nil)}).
				WithLimiter(NewLimiter(1)).
				Build()
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(BeAssignableToTypeOf(&CachingVerifier{}))
		})

		It("accepts a nil resolver with non-empty specs", func() {
			v, err := NewVerifierBuilder().
				WithSpecs([]SignatureSpec{DefaultSignatureSpec("sig", nil)}).
				Build()
			Expect(err).NotTo(HaveOccurred())
			Expect(v).NotTo(BeNil())
		})

		It("returns an error for duplicate spec names", func() {
			_, err := NewVerifierBuilder().
				WithSpecs([]SignatureSpec{
					DefaultSignatureSpec("dup", nil),
					DefaultSignatureSpec("dup", nil),
				}).
				Build()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`duplicate signature name detected: "dup"`))
		})
	})
})

var _ = Describe("SignerBuilder", func() {
	Describe("Build", func() {
		It("returns NoopSigner when specs are empty", func() {
			s, err := NewSignerBuilder().Build()
			Expect(err).NotTo(HaveOccurred())
			Expect(s).To(BeAssignableToTypeOf(NoopSigner{}))
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
