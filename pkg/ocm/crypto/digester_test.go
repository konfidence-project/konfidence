package crypto

import (
	"context"
	"crypto"
	"fmt"
	"log/slog"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	norm "ocm.software/open-component-model/bindings/go/descriptor/normalisation/json/v4alpha1"
	"ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/signing"
)

var _ = Describe("ConfigurableDigester", func() {
	AfterEach(func() {
		isSafelyDigestible = signing.IsSafelyDigestible
		generateDigest = signing.GenerateDigest
	})
	It("Returns the correct hash algorithm", func() {
		digester := NewDefaultDigester(logr.Discard())
		Expect(digester.GetHashAlgorithm()).To(Equal(crypto.SHA256))
	})
	It("Returns the correct normalisation algorithm", func() {
		digester := NewDefaultDigester(logr.Discard())
		Expect(digester.GetNormalisationAlgorithm()).To(Equal(norm.Algorithm))
	})
	It("Returns an error if the descriptor is not safely digestible", func() {
		isSafelyDigestible = func(component *runtime.Component) error {
			return fmt.Errorf("reason why component is not safely digestible")
		}
		digester := NewDefaultDigester(logr.Discard())
		_, err := digester.GenerateDigest(context.Background(), &runtime.Descriptor{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("reason why component is not safely digestible"))
	})
	It("Returns an error if generating the digest fails", func() {
		isSafelyDigestible = func(component *runtime.Component) error {
			return nil
		}
		generateDigest = func(
			ctx context.Context,
			desc *runtime.Descriptor,
			log *slog.Logger,
			normalisationAlgorithm string,
			hashAlgorithm string) (*runtime.Digest, error) {
			return nil, fmt.Errorf("reason why generating the digest failed")
		}
		digester := NewDefaultDigester(logr.Discard())
		_, err := digester.GenerateDigest(context.Background(), &runtime.Descriptor{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("reason why generating the digest failed"))
	})
	// We do not test the underlying OCM default implementation,
	// but we check if the digest is returned in case of a successful digest generation.
	It("Returns the generated digest", func() {
		expectedDigest := &runtime.Digest{
			HashAlgorithm:          crypto.SHA256.String(),
			NormalisationAlgorithm: norm.Algorithm,
			Value:                  "sha256:1234567890abcdef",
		}
		isSafelyDigestible = func(component *runtime.Component) error {
			return nil
		}
		generateDigest = func(
			ctx context.Context,
			desc *runtime.Descriptor,
			log *slog.Logger,
			normalisationAlgorithm string,
			hashAlgorithm string) (*runtime.Digest, error) {
			return expectedDigest, nil
		}
		digester := NewDefaultDigester(logr.Discard())
		dig, err := digester.GenerateDigest(context.Background(), &runtime.Descriptor{})
		Expect(err).ToNot(HaveOccurred())
		Expect(dig).To(Equal(expectedDigest))
	})
})
