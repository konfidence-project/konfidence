package crypto

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"

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
	identityv1 "ocm.software/open-component-model/bindings/go/rsa/spec/identity/v1"
	ocmruntime "ocm.software/open-component-model/bindings/go/runtime"
	"ocm.software/open-component-model/bindings/go/signing"
)

// rsaIdentity builds an OCM RSA consumer identity that mirrors what
// rsa/signing/handler emits — versioned RSA/v1 type plus algorithm and
// signature attributes — so test mocks accept the exact map a real handler
// would produce in production.
func rsaIdentity(name, algorithm string) ocmruntime.Identity {
	id := ocmruntime.Identity{
		identityv1.IdentityAttributeAlgorithm: algorithm,
		identityv1.IdentityAttributeSignature: name,
	}
	id.SetType(identityv1.VersionedType)
	return id
}

// matchingSig returns a Signature whose pin fields match the given SignatureSpec.
func matchingSig(name string, spec SignatureSpec) runtime.Signature {
	return runtime.Signature{
		Name: name,
		Digest: runtime.Digest{
			HashAlgorithm:          spec.HashAlgorithm,
			NormalisationAlgorithm: spec.NormalisationAlgorithm,
		},
		Signature: runtime.SignatureInfo{
			MediaType: spec.MediaType,
		},
	}
}

// defaultSpec returns a SignatureSpec with secure defaults and no issuer pin.
func defaultSpec(name string) SignatureSpec {
	return DefaultSignatureSpec(name, nil)
}

var _ = Describe("OCMVerifier", func() {
	var (
		log = logr.Discard()

		creds = &rsacredentialsv1.RSACredentials{
			Type:         rsacredentialsv1.VersionedType,
			PublicKeyPEM: "test-cert",
		}
		identity  = rsaIdentity("sig1", string(rsav1alpha1.AlgorithmRSASSAPSS))
		identity2 = rsaIdentity("sig2", string(rsav1alpha1.AlgorithmRSASSAPSS))

		verifierMock *mocks.MockVerifier
		resolverMock *mocks.MockResolver
		mockCtrl     *gomock.Controller
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		verifierMock = mocks.NewMockVerifier(mockCtrl)
		resolverMock = mocks.NewMockResolver(mockCtrl)
		verifyDigestMatchesDescriptor = func(_ context.Context, _ *runtime.Descriptor, _ runtime.Signature, _ *slog.Logger) error {
			return nil
		}
		isSafelyDigestible = func(_ *runtime.Component) error { return nil }
	})

	AfterEach(func() {
		mockCtrl.Finish()
		isSafelyDigestible = signing.IsSafelyDigestible
		verifyDigestMatchesDescriptor = signing.VerifyDigestMatchesDescriptor
	})

	// makeVerifier builds an OCMVerifier wired with test doubles. Under the new
	// interface, specs and resolver are per-call arguments, not construction-
	// time captures — makeVerifier only wires in the crypto handler and log.
	makeVerifier := func() *ocmVerifier {
		return &ocmVerifier{
			log:         log,
			rsaVerifier: verifierMock,
		}
	}

	It("works with 1 signature", func() {
		spec := defaultSpec("sig1")
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{matchingSig("sig1", spec)}}
		v := makeVerifier()

		verifierMock.EXPECT().GetVerifyingCredentialConsumerIdentity(gomock.Any(), desc.Signatures[0], gomock.Nil()).Return(identity, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity).Return(creds, nil)
		verifierMock.EXPECT().Verify(gomock.Any(), desc.Signatures[0], gomock.Nil(), creds).Return(nil)

		Expect(v.Verify(context.Background(), resolverMock, []SignatureSpec{spec}, []*runtime.Descriptor{desc})).To(Succeed())
	})

	It("works with multiple signatures, resolving creds per signature", func() {
		spec1, spec2 := defaultSpec("sig1"), defaultSpec("sig2")
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{
			matchingSig("sig1", spec1),
			matchingSig("sig2", spec2),
		}}
		v := makeVerifier()

		verifierMock.EXPECT().GetVerifyingCredentialConsumerIdentity(gomock.Any(), desc.Signatures[0], gomock.Nil()).Return(identity, nil)
		verifierMock.EXPECT().GetVerifyingCredentialConsumerIdentity(gomock.Any(), desc.Signatures[1], gomock.Nil()).Return(identity2, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity).Return(creds, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity2).Return(creds, nil)
		verifierMock.EXPECT().Verify(gomock.Any(), desc.Signatures[0], gomock.Nil(), creds).Return(nil)
		verifierMock.EXPECT().Verify(gomock.Any(), desc.Signatures[1], gomock.Nil(), creds).Return(nil)

		Expect(v.Verify(context.Background(), resolverMock, []SignatureSpec{spec1, spec2}, []*runtime.Descriptor{desc})).To(Succeed())
	})

	It("works with 2 descriptors and 1 signature each", func() {
		spec := defaultSpec("sig1")
		desc1 := &runtime.Descriptor{Signatures: []runtime.Signature{matchingSig("sig1", spec)}}
		desc2 := &runtime.Descriptor{Signatures: []runtime.Signature{matchingSig("sig1", spec)}}
		v := makeVerifier()

		verifierMock.EXPECT().GetVerifyingCredentialConsumerIdentity(gomock.Any(), gomock.Any(), gomock.Nil()).Times(2).Return(identity, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity).Return(creds, nil).Times(2)
		verifierMock.EXPECT().Verify(gomock.Any(), desc1.Signatures[0], gomock.Nil(), creds).Return(nil)
		verifierMock.EXPECT().Verify(gomock.Any(), desc2.Signatures[0], gomock.Nil(), creds).Return(nil)

		Expect(v.Verify(context.Background(), resolverMock, []SignatureSpec{spec}, []*runtime.Descriptor{desc1, desc2})).To(Succeed())
	})

	It("works with 2 descriptors and multiple signatures each (full matrix)", func() {
		spec1, spec2 := defaultSpec("sig1"), defaultSpec("sig2")
		desc1 := &runtime.Descriptor{Signatures: []runtime.Signature{matchingSig("sig1", spec1), matchingSig("sig2", spec2)}}
		desc2 := &runtime.Descriptor{Signatures: []runtime.Signature{matchingSig("sig1", spec1), matchingSig("sig2", spec2)}}
		v := makeVerifier()

		verifierMock.EXPECT().GetVerifyingCredentialConsumerIdentity(gomock.Any(), gomock.Any(), gomock.Nil()).Times(4).Return(identity, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity).Return(creds, nil).Times(4)
		for _, desc := range []*runtime.Descriptor{desc1, desc2} {
			for _, sig := range desc.Signatures {
				verifierMock.EXPECT().Verify(gomock.Any(), sig, gomock.Nil(), creds).Return(nil)
			}
		}

		Expect(v.Verify(context.Background(), resolverMock, []SignatureSpec{spec1, spec2}, []*runtime.Descriptor{desc1, desc2})).To(Succeed())
	})

	It("fails when signature is missing in descriptor", func() {
		spec1 := defaultSpec("sig1")
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{}}
		v := makeVerifier()

		Expect(v.Verify(context.Background(), nil, []SignatureSpec{spec1}, []*runtime.Descriptor{desc})).To(MatchError(
			ContainSubstring(`signature "sig1" not found in descriptor`)))
	})

	It("fails when digest verification fails", func() {
		spec := defaultSpec("sig1")
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{matchingSig("sig1", spec)}}
		v := makeVerifier()

		verifyDigestMatchesDescriptor = func(_ context.Context, _ *runtime.Descriptor, _ runtime.Signature, _ *slog.Logger) error {
			return fmt.Errorf("digest does not match descriptor")
		}

		Expect(v.Verify(context.Background(), resolverMock, []SignatureSpec{spec}, []*runtime.Descriptor{desc})).To(MatchError(
			ContainSubstring(`digest mismatch for "sig1": digest does not match descriptor`)))
	})

	It("fails when signature verification fails", func() {
		spec := defaultSpec("sig1")
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{matchingSig("sig1", spec)}}
		v := makeVerifier()

		verifierMock.EXPECT().GetVerifyingCredentialConsumerIdentity(gomock.Any(), desc.Signatures[0], gomock.Nil()).Return(identity, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity).Return(creds, nil)
		verifierMock.EXPECT().Verify(gomock.Any(), desc.Signatures[0], gomock.Nil(), creds).Return(fmt.Errorf("invalid signature"))

		Expect(v.Verify(context.Background(), resolverMock, []SignatureSpec{spec}, []*runtime.Descriptor{desc})).To(MatchError(
			ContainSubstring(`signature verification failed for "sig1": invalid signature`)))
	})

	It("empty specs is a no-op", func() {
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{matchingSig("sig1", defaultSpec("sig1"))}}
		v := makeVerifier()
		Expect(v.Verify(context.Background(), resolverMock, nil, []*runtime.Descriptor{desc})).To(Succeed())
	})

	It("empty descs is a no-op", func() {
		v := makeVerifier()
		Expect(v.Verify(context.Background(), resolverMock, []SignatureSpec{defaultSpec("sig1")}, nil)).To(Succeed())
	})

	It("one failed verification in multi-descriptor batch fails entire operation", func() {
		spec := defaultSpec("sig1")
		desc1 := &runtime.Descriptor{Signatures: []runtime.Signature{matchingSig("sig1", spec)}}
		desc2 := &runtime.Descriptor{Signatures: []runtime.Signature{matchingSig("sig1", spec)}}
		v := makeVerifier()

		verifierMock.EXPECT().GetVerifyingCredentialConsumerIdentity(gomock.Any(), gomock.Any(), gomock.Nil()).Times(2).Return(identity, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity).Return(creds, nil).Times(2)
		verifierMock.EXPECT().Verify(gomock.Any(), desc1.Signatures[0], gomock.Nil(), creds).Return(nil)
		verifierMock.EXPECT().Verify(gomock.Any(), desc2.Signatures[0], gomock.Nil(), creds).Return(fmt.Errorf("bad sig"))

		Expect(v.Verify(context.Background(), resolverMock, []SignatureSpec{spec}, []*runtime.Descriptor{desc1, desc2})).To(HaveOccurred())
	})

	It("verifies only target signatures when descriptor has additional signatures", func() {
		spec1, spec2 := defaultSpec("sig1"), defaultSpec("sig2")
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{
			matchingSig("sig1", spec1),
			matchingSig("sig2", spec2),
			matchingSig("sig3", defaultSpec("sig3")), // extra — must be ignored
		}}
		v := makeVerifier()

		verifierMock.EXPECT().GetVerifyingCredentialConsumerIdentity(gomock.Any(), desc.Signatures[0], gomock.Nil()).Return(identity, nil)
		verifierMock.EXPECT().GetVerifyingCredentialConsumerIdentity(gomock.Any(), desc.Signatures[1], gomock.Nil()).Return(identity2, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity).Return(creds, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity2).Return(creds, nil)
		verifierMock.EXPECT().Verify(gomock.Any(), desc.Signatures[0], gomock.Nil(), creds).Return(nil)
		verifierMock.EXPECT().Verify(gomock.Any(), desc.Signatures[1], gomock.Nil(), creds).Return(nil)

		Expect(v.Verify(context.Background(), resolverMock, []SignatureSpec{spec1, spec2}, []*runtime.Descriptor{desc})).To(Succeed())
	})

	It("treats ErrNotFound from the resolver as nil credentials", func() {
		spec := defaultSpec("sig1")
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{matchingSig("sig1", spec)}}
		v := makeVerifier()

		verifierMock.EXPECT().GetVerifyingCredentialConsumerIdentity(gomock.Any(), desc.Signatures[0], gomock.Nil()).Return(identity, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity).Return(nil, credentials.ErrNotFound)
		verifierMock.EXPECT().Verify(gomock.Any(), desc.Signatures[0], gomock.Nil(), gomock.Nil()).Return(nil)

		Expect(v.Verify(context.Background(), resolverMock, []SignatureSpec{spec}, []*runtime.Descriptor{desc})).To(Succeed())
	})

	It("passes nil credentials when nil resolver is supplied", func() {
		spec := defaultSpec("sig1")
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{matchingSig("sig1", spec)}}
		v := makeVerifier()

		verifierMock.EXPECT().GetVerifyingCredentialConsumerIdentity(gomock.Any(), desc.Signatures[0], gomock.Nil()).Return(identity, nil)
		verifierMock.EXPECT().Verify(gomock.Any(), desc.Signatures[0], gomock.Nil(), gomock.Nil()).Return(nil)

		// nil resolver → falls back to empty static resolver; every Resolve returns ErrNotFound → nil creds
		Expect(v.Verify(context.Background(), nil, []SignatureSpec{spec}, []*runtime.Descriptor{desc})).To(Succeed())
	})

	It("fails when resolver returns an error other than ErrNotFound", func() {
		spec := defaultSpec("sig1")
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{matchingSig("sig1", spec)}}
		v := makeVerifier()
		boom := fmt.Errorf("resolver stopped")

		verifierMock.EXPECT().GetVerifyingCredentialConsumerIdentity(gomock.Any(), desc.Signatures[0], gomock.Nil()).Return(identity, nil)
		resolverMock.EXPECT().Resolve(gomock.Any(), identity).Return(nil, boom)

		err := v.Verify(context.Background(), resolverMock, []SignatureSpec{spec}, []*runtime.Descriptor{desc})
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, boom)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("resolve credentials"))
	})

	It("fails when GetVerifyingCredentialConsumerIdentity errors", func() {
		spec := defaultSpec("sig1")
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{matchingSig("sig1", spec)}}
		v := makeVerifier()

		verifierMock.EXPECT().GetVerifyingCredentialConsumerIdentity(gomock.Any(), desc.Signatures[0], gomock.Nil()).
			Return(nil, fmt.Errorf("identity build failed"))

		err := v.Verify(context.Background(), resolverMock, []SignatureSpec{spec}, []*runtime.Descriptor{desc})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("derive consumer identity"))
		Expect(err.Error()).To(ContainSubstring("identity build failed"))
	})

	// ── Pin-field mismatch cases ────────────────────────────────────────────

	It("fails when MediaType in descriptor does not match spec", func() {
		spec := defaultSpec("sig1")
		s := matchingSig("sig1", spec)
		s.Signature.MediaType = "application/vnd.ocm.signature.rsa.pss" // wrong
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{s}}
		v := makeVerifier()

		Expect(v.Verify(context.Background(), nil, []SignatureSpec{spec}, []*runtime.Descriptor{desc})).To(MatchError(
			ContainSubstring("media type mismatch")))
	})

	It("fails when HashAlgorithm in descriptor does not match spec", func() {
		spec := defaultSpec("sig1")
		s := matchingSig("sig1", spec)
		s.Digest.HashAlgorithm = "SHA-512" // wrong
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{s}}
		v := makeVerifier()

		Expect(v.Verify(context.Background(), nil, []SignatureSpec{spec}, []*runtime.Descriptor{desc})).To(MatchError(
			ContainSubstring("hash algorithm mismatch")))
	})

	It("fails when NormalisationAlgorithm in descriptor does not match spec", func() {
		spec := defaultSpec("sig1")
		s := matchingSig("sig1", spec)
		s.Digest.NormalisationAlgorithm = "jsonNormalisation/v1" // wrong
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{s}}
		v := makeVerifier()

		Expect(v.Verify(context.Background(), nil, []SignatureSpec{spec}, []*runtime.Descriptor{desc})).To(MatchError(
			ContainSubstring("normalisation algorithm mismatch")))
	})

	// ── Issuer injection cases ───────────────────────────────────────────────

	It("verify signature with issuer pin injects the pin before crypto", func() {
		issuer := "CN=my-ca,O=Acme"
		spec := DefaultSignatureSpec("sig1", &issuer)
		s := matchingSig("sig1", spec)
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{s}}
		v := makeVerifier()

		verifierMock.EXPECT().GetVerifyingCredentialConsumerIdentity(gomock.Any(), gomock.Any(), gomock.Nil()).Return(identity, nil)
		verifierMock.EXPECT().Verify(gomock.Any(), gomock.AssignableToTypeOf(runtime.Signature{}), gomock.Nil(), gomock.Nil()).
			DoAndReturn(func(_ context.Context, gotSig runtime.Signature, _ ocmruntime.Typed, _ ocmruntime.Typed) error {
				Expect(gotSig.Signature.Issuer).To(Equal(issuer))
				return nil
			})

		Expect(v.Verify(context.Background(), nil, []SignatureSpec{spec}, []*runtime.Descriptor{desc})).To(Succeed())
	})

	It("does not inject issuer when spec.Issuer is nil", func() {
		spec := defaultSpec("sig1") // Issuer == nil
		s := matchingSig("sig1", spec)
		s.Signature.Issuer = "original-issuer"
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{s}}
		v := makeVerifier()

		verifierMock.EXPECT().GetVerifyingCredentialConsumerIdentity(gomock.Any(), gomock.Any(), gomock.Nil()).Return(identity, nil)
		verifierMock.EXPECT().Verify(gomock.Any(), gomock.AssignableToTypeOf(runtime.Signature{}), gomock.Nil(), gomock.Nil()).
			DoAndReturn(func(_ context.Context, gotSig runtime.Signature, _ ocmruntime.Typed, _ ocmruntime.Typed) error {
				Expect(gotSig.Signature.Issuer).To(Equal("original-issuer"))
				return nil
			})

		Expect(v.Verify(context.Background(), nil, []SignatureSpec{spec}, []*runtime.Descriptor{desc})).To(Succeed())
	})

	It("uses the call-argument resolver — verifier holds no captured resolver", func() {
		// Sanity: two Verify calls on the same OCMVerifier with different
		// resolvers must resolve creds through the resolver supplied to that
		// specific call, not one captured elsewhere.
		spec := defaultSpec("sig1")
		desc := &runtime.Descriptor{Signatures: []runtime.Signature{matchingSig("sig1", spec)}}
		v := makeVerifier()

		var callsOnR1, callsOnR2 atomic.Int32
		r1 := &countingResolver{onResolve: func() { callsOnR1.Add(1) }, creds: creds}
		r2 := &countingResolver{onResolve: func() { callsOnR2.Add(1) }, creds: creds}

		verifierMock.EXPECT().GetVerifyingCredentialConsumerIdentity(gomock.Any(), gomock.Any(), gomock.Nil()).Times(2).Return(identity, nil)
		verifierMock.EXPECT().Verify(gomock.Any(), gomock.Any(), gomock.Nil(), creds).Times(2).Return(nil)

		Expect(v.Verify(context.Background(), r1, []SignatureSpec{spec}, []*runtime.Descriptor{desc})).To(Succeed())
		Expect(v.Verify(context.Background(), r2, []SignatureSpec{spec}, []*runtime.Descriptor{desc})).To(Succeed())
		Expect(callsOnR1.Load()).To(Equal(int32(1)))
		Expect(callsOnR2.Load()).To(Equal(int32(1)))
	})

	It("newOCMVerifier constructs without specs or resolver", func() {
		v, err := newOCMVerifier()
		Expect(err).NotTo(HaveOccurred())
		Expect(v).NotTo(BeNil())
	})

	It("newOCMVerifier constructs and can be configured via options", func() {
		v, err := newOCMVerifier(withVerifierLogger(logr.Discard()))
		Expect(err).NotTo(HaveOccurred())
		Expect(v).NotTo(BeNil())
	})

	It("NewSignatureSpec constructs with an empty-string issuer — validation is deferred", func() {
		empty := ""
		spec := NewSignatureSpec(
			"sig1",
			rsav1alpha1.AlgorithmRSASSAPSS,
			rsav1alpha1.MediaTypePEM,
			crypto.SHA256.String(),
			norm.Algorithm,
			&empty,
		)
		// Construction itself does not validate — the empty pin is only rejected
		// at Verify time by the preFlight layer (see verifier_preflight_test.go).
		Expect(spec.Issuer).NotTo(BeNil())
		Expect(*spec.Issuer).To(Equal(""))
	})
})

// countingResolver is a test double that counts Resolve invocations and
// returns a fixed credential.
type countingResolver struct {
	onResolve func()
	creds     ocmruntime.Typed
}

func (r *countingResolver) Resolve(_ context.Context, _ ocmruntime.Identity) (ocmruntime.Typed, error) {
	r.onResolve()
	return r.creds, nil
}
