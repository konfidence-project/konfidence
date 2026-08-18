package crypto

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto/internal/mocks"
	"go.uber.org/mock/gomock"
	"ocm.software/open-component-model/bindings/go/credentials"
	norm "ocm.software/open-component-model/bindings/go/descriptor/normalisation/json/v4alpha1"
	"ocm.software/open-component-model/bindings/go/descriptor/runtime"
	rsav1alpha1 "ocm.software/open-component-model/bindings/go/rsa/signing/v1alpha1"
	rsacredentialsv1 "ocm.software/open-component-model/bindings/go/rsa/spec/credentials/v1"
	ocmruntime "ocm.software/open-component-model/bindings/go/runtime"
	"ocm.software/open-component-model/bindings/go/signing"
)

var _ = Describe("ocmSigner", func() {
	var (
		log = logr.Discard()

		creds = &rsacredentialsv1.RSACredentials{
			Type:          rsacredentialsv1.VersionedType,
			PrivateKeyPEM: "test-key",
		}
		identity = rsaIdentity("sig1", string(rsav1alpha1.AlgorithmRSASSAPSS))

		signerMock   *mocks.MockSigner
		resolverMock *mocks.MockResolver
		mockCtrl     *gomock.Controller

		stubDig = runtime.Digest{
			HashAlgorithm:          "SHA-256",
			NormalisationAlgorithm: norm.Algorithm,
			Value:                  "test-digest",
		}
		stubSig = runtime.SignatureInfo{
			Algorithm: string(rsav1alpha1.AlgorithmRSASSAPSS),
			Value:     "sig_data",
			MediaType: rsav1alpha1.MediaTypePEM,
		}
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		signerMock = mocks.NewMockSigner(mockCtrl)
		resolverMock = mocks.NewMockResolver(mockCtrl)
		generateDigest = func(_ context.Context, _ *runtime.Descriptor, _ *slog.Logger, normAlgo, hashAlgo string) (*runtime.Digest, error) {
			return &runtime.Digest{
				HashAlgorithm:          hashAlgo,
				NormalisationAlgorithm: normAlgo,
				Value:                  "test-digest",
			}, nil
		}
		isSafelyDigestible = func(_ *runtime.Component) error { return nil }
	})

	AfterEach(func() {
		mockCtrl.Finish()
		generateDigest = signing.GenerateDigest
		isSafelyDigestible = signing.IsSafelyDigestible
	})

	makeOCMSigner := func(resolver credentials.Resolver, specs []SignatureSpec) *ocmSigner {
		return &ocmSigner{
			log:       log,
			rsaSigner: signerMock,
			resolver:  resolver,
			specs:     specs,
			limiter:   NoopLimiter{},
		}
	}

	It("works with 1 signature", func() {
		spec := DefaultSignatureSpec("sig1", nil)
		desc := &runtime.Descriptor{}
		s := makeOCMSigner(resolverMock, []SignatureSpec{spec})

		signerMock.EXPECT().GetSigningCredentialConsumerIdentity(gomock.Any(), "sig1", stubDig, gomock.Any()).Return(identity, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity).Return(creds, nil)
		signerMock.EXPECT().Sign(gomock.Any(), stubDig, gomock.Any(), creds).Return(stubSig, nil)

		Expect(s.Sign(context.Background(), desc)).To(Succeed())
		Expect(desc.Signatures).To(HaveLen(1))
		Expect(desc.Signatures[0].Name).To(Equal("sig1"))
		Expect(desc.Signatures[0].Digest).To(Equal(stubDig))
		Expect(desc.Signatures[0].Signature).To(Equal(stubSig))
	})

	It("injects issuer into SignatureInfo when spec.Issuer is set", func() {
		issuer := "CN=my-ca,O=acme"
		spec := DefaultSignatureSpec("sig1", &issuer)
		desc := &runtime.Descriptor{}
		s := makeOCMSigner(resolverMock, []SignatureSpec{spec})

		signerMock.EXPECT().GetSigningCredentialConsumerIdentity(gomock.Any(), "sig1", stubDig, gomock.Any()).Return(identity, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity).Return(creds, nil)
		signerMock.EXPECT().Sign(gomock.Any(), stubDig, gomock.Any(), creds).Return(stubSig, nil)

		Expect(s.Sign(context.Background(), desc)).To(Succeed())
		Expect(desc.Signatures).To(HaveLen(1))
		Expect(desc.Signatures[0].Signature.Issuer).To(Equal(issuer))
	})

	It("leaves Issuer empty in SignatureInfo when spec.Issuer is nil", func() {
		spec := DefaultSignatureSpec("sig1", nil)
		desc := &runtime.Descriptor{}
		s := makeOCMSigner(resolverMock, []SignatureSpec{spec})

		signerMock.EXPECT().GetSigningCredentialConsumerIdentity(gomock.Any(), "sig1", stubDig, gomock.Any()).Return(identity, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity).Return(creds, nil)
		signerMock.EXPECT().Sign(gomock.Any(), stubDig, gomock.Any(), creds).Return(stubSig, nil)

		Expect(s.Sign(context.Background(), desc)).To(Succeed())
		Expect(desc.Signatures).To(HaveLen(1))
		Expect(desc.Signatures[0].Signature.Issuer).To(BeEmpty())
	})

	It("works with 3 signatures", func() {
		specs := []SignatureSpec{
			DefaultSignatureSpec("sig1", nil),
			DefaultSignatureSpec("sig2", nil),
			DefaultSignatureSpec("sig3", nil),
		}
		desc := &runtime.Descriptor{}
		s := makeOCMSigner(resolverMock, specs)

		signerMock.EXPECT().GetSigningCredentialConsumerIdentity(gomock.Any(), gomock.Any(), stubDig, gomock.Any()).
			Times(3).Return(identity, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity).Return(creds, nil).Times(3)
		signerMock.EXPECT().Sign(gomock.Any(), stubDig, gomock.Any(), creds).Times(3).Return(stubSig, nil)

		Expect(s.Sign(context.Background(), desc)).To(Succeed())
		Expect(desc.Signatures).To(HaveLen(3))
		for _, spec := range specs {
			found := false
			for _, sig := range desc.Signatures {
				if sig.Name == spec.Name {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "expected signature %q in descriptor", spec.Name)
		}
	})

	It("each spec uses its own rsaConfig", func() {
		spec1 := NewSignatureSpec("sig1", rsav1alpha1.AlgorithmRSASSAPSS, rsav1alpha1.MediaTypePEM, "SHA-256", norm.Algorithm, nil)
		spec2 := NewSignatureSpec("sig2", rsav1alpha1.AlgorithmRSASSAPKCS1V15, rsav1alpha1.MediaTypePEM, "SHA-256", norm.Algorithm, nil)
		desc := &runtime.Descriptor{}
		s := makeOCMSigner(resolverMock, []SignatureSpec{spec1, spec2})

		dig1 := runtime.Digest{HashAlgorithm: "SHA-256", NormalisationAlgorithm: norm.Algorithm, Value: "test-digest"}
		dig2 := dig1

		signerMock.EXPECT().GetSigningCredentialConsumerIdentity(gomock.Any(), "sig1", dig1, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ runtime.Digest, cfg *rsav1alpha1.Config) (ocmruntime.Identity, error) {
				Expect(cfg.SignatureAlgorithm).To(Equal(rsav1alpha1.AlgorithmRSASSAPSS))
				return identity, nil
			})
		signerMock.EXPECT().GetSigningCredentialConsumerIdentity(gomock.Any(), "sig2", dig2, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ runtime.Digest, cfg *rsav1alpha1.Config) (ocmruntime.Identity, error) {
				Expect(cfg.SignatureAlgorithm).To(Equal(rsav1alpha1.AlgorithmRSASSAPKCS1V15))
				return identity, nil
			})
		resolverMock.EXPECT().Resolve(gomock.Any(), identity).Return(creds, nil).Times(2)
		signerMock.EXPECT().Sign(gomock.Any(), gomock.Any(), gomock.Any(), creds).Times(2).Return(stubSig, nil)

		Expect(s.Sign(context.Background(), desc)).To(Succeed())
		Expect(desc.Signatures).To(HaveLen(2))
	})

	It("fails when signature already exists in descriptor", func() {
		spec := DefaultSignatureSpec("sig1", nil)
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{{Name: "sig1"}}}
		s := makeOCMSigner(resolverMock, []SignatureSpec{spec})

		Expect(s.Sign(context.Background(), desc)).To(MatchError(ContainSubstring(`signature with name "sig1" already exists`)))
		Expect(desc.Signatures).To(HaveLen(1))
	})

	It("fails when digest generation fails", func() {
		spec := DefaultSignatureSpec("sig1", nil)
		desc := &runtime.Descriptor{}
		s := makeOCMSigner(resolverMock, []SignatureSpec{spec})

		generateDigest = func(_ context.Context, _ *runtime.Descriptor, _ *slog.Logger, _, _ string) (*runtime.Digest, error) {
			return nil, fmt.Errorf("digest generation failed")
		}

		Expect(s.Sign(context.Background(), desc)).To(MatchError(ContainSubstring("digest generation failed")))
		Expect(desc.Signatures).To(BeEmpty())
	})

	It("fails when GetSigningCredentialConsumerIdentity errors", func() {
		spec := DefaultSignatureSpec("sig1", nil)
		desc := &runtime.Descriptor{}
		s := makeOCMSigner(resolverMock, []SignatureSpec{spec})

		signerMock.EXPECT().GetSigningCredentialConsumerIdentity(gomock.Any(), "sig1", stubDig, gomock.Any()).
			Return(nil, fmt.Errorf("identity build failed"))

		err := s.Sign(context.Background(), desc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("derive consumer identity"))
		Expect(err.Error()).To(ContainSubstring("identity build failed"))
		Expect(desc.Signatures).To(BeEmpty())
	})

	It("fails when resolver returns an error", func() {
		spec := DefaultSignatureSpec("sig1", nil)
		desc := &runtime.Descriptor{}
		s := makeOCMSigner(resolverMock, []SignatureSpec{spec})

		signerMock.EXPECT().GetSigningCredentialConsumerIdentity(gomock.Any(), "sig1", stubDig, gomock.Any()).Return(identity, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity).Return(nil, fmt.Errorf("resolver stopped"))

		err := s.Sign(context.Background(), desc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("resolve signing credentials"))
		Expect(err.Error()).To(ContainSubstring("resolver stopped"))
		Expect(desc.Signatures).To(BeEmpty())
	})

	It("ErrNotFound from resolver is a hard failure", func() {
		spec := DefaultSignatureSpec("sig1", nil)
		desc := &runtime.Descriptor{}
		s := makeOCMSigner(resolverMock, []SignatureSpec{spec})

		signerMock.EXPECT().GetSigningCredentialConsumerIdentity(gomock.Any(), "sig1", stubDig, gomock.Any()).Return(identity, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity).Return(nil, credentials.ErrNotFound)

		err := s.Sign(context.Background(), desc)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, credentials.ErrNotFound)).To(BeTrue())
		Expect(desc.Signatures).To(BeEmpty())
	})

	It("fails when Sign (crypto) fails", func() {
		spec := DefaultSignatureSpec("sig1", nil)
		desc := &runtime.Descriptor{}
		s := makeOCMSigner(resolverMock, []SignatureSpec{spec})

		signerMock.EXPECT().GetSigningCredentialConsumerIdentity(gomock.Any(), "sig1", stubDig, gomock.Any()).Return(identity, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity).Return(creds, nil)
		signerMock.EXPECT().Sign(gomock.Any(), stubDig, gomock.Any(), creds).Return(runtime.SignatureInfo{}, fmt.Errorf("signing failed"))

		Expect(s.Sign(context.Background(), desc)).To(MatchError(ContainSubstring("signing failed")))
		Expect(desc.Signatures).To(BeEmpty())
	})

	It("appends to existing signatures in descriptor", func() {
		spec := DefaultSignatureSpec("new-sig", nil)
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{{Name: "existing-sig"}}}
		s := makeOCMSigner(resolverMock, []SignatureSpec{spec})

		signerMock.EXPECT().GetSigningCredentialConsumerIdentity(gomock.Any(), "new-sig", stubDig, gomock.Any()).Return(identity, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity).Return(creds, nil)
		signerMock.EXPECT().Sign(gomock.Any(), stubDig, gomock.Any(), creds).Return(stubSig, nil)

		Expect(s.Sign(context.Background(), desc)).To(Succeed())
		Expect(desc.Signatures).To(HaveLen(2))
		Expect(desc.Signatures[0].Name).To(Equal("existing-sig"))
		Expect(desc.Signatures[1].Name).To(Equal("new-sig"))
	})

	It("newOCMSigner errors with nil resolver", func() {
		s, err := newOCMSigner(nil, []SignatureSpec{DefaultSignatureSpec("sig1", nil)})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("credentials resolver is required"))
		Expect(s).To(BeNil())
	})

	It("newOCMSigner with empty specs slice builds a no-op signer", func() {
		s, err := newOCMSigner(resolverMock, []SignatureSpec{})
		Expect(err).NotTo(HaveOccurred())
		Expect(s).NotTo(BeNil())
		// No specs → Sign is a no-op: no error, descriptor untouched, resolver unused.
		desc := &runtime.Descriptor{}
		Expect(s.Sign(context.Background(), desc)).To(Succeed())
		Expect(desc.Signatures).To(BeEmpty())
	})

	It("newOCMSigner with nil specs slice builds a no-op signer", func() {
		s, err := newOCMSigner(resolverMock, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(s).NotTo(BeNil())
		desc := &runtime.Descriptor{}
		Expect(s.Sign(context.Background(), desc)).To(Succeed())
		Expect(desc.Signatures).To(BeEmpty())
	})

	It("newOCMSigner errors with duplicate spec names", func() {
		s, err := newOCMSigner(resolverMock, []SignatureSpec{
			DefaultSignatureSpec("sig1", nil),
			DefaultSignatureSpec("sig1", nil),
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`duplicate signature name detected: "sig1"`))
		Expect(s).To(BeNil())
	})

	It("newOCMSigner errors with empty spec name", func() {
		s, err := newOCMSigner(resolverMock, []SignatureSpec{DefaultSignatureSpec("  ", nil)})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("signature names cannot be empty or whitespace"))
		Expect(s).To(BeNil())
	})

	It("newOCMSigner applies withSignerLogger option", func() {
		s, err := newOCMSigner(resolverMock, []SignatureSpec{DefaultSignatureSpec("sig1", nil)}, withSignerLogger(logr.Discard().WithName("custom")))
		Expect(err).ToNot(HaveOccurred())
		Expect(s).ToNot(BeNil())
	})
})
