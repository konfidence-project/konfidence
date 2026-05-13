package crypto

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-logr/logr"
	"github.com/konfidence-project/pkg/ocm/crypto/internal/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/signing"
)

var _ = Describe("RSAVerifier", func() {
	var (
		log = logr.Discard()
		// mock credentials to verify they are passed through
		creds        = map[string]string{"public_key_pem": "test-cert"}
		verifierMock *mocks.MockVerifier
		providerMock *mocks.MockRSACredentialProvider
		mockCtrl     *gomock.Controller
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		verifierMock = mocks.NewMockVerifier(mockCtrl)
		providerMock = mocks.NewMockRSACredentialProvider(mockCtrl)
	})
	AfterEach(func() {
		mockCtrl.Finish()
		isSafelyDigestible = signing.IsSafelyDigestible
		verifyDigestMatchesDescriptor = signing.VerifyDigestMatchesDescriptor
	})
	It("works with 1 signature", func() {
		desc := &runtime.Descriptor{
			Signatures: []runtime.Signature{{Name: "sig1"}},
		}
		verifier := &RSAVerifier{
			log:              log,
			rsaVerifier:      verifierMock,
			targetSignatures: []string{"sig1"},
			provider:         providerMock,
		}
		verifyDigestMatchesDescriptor = func(
			ctx context.Context,
			desc *runtime.Descriptor,
			sig runtime.Signature,
			log *slog.Logger) error {
			return nil
		}
		isSafelyDigestible = func(desc *runtime.Component) error { return nil }
		providerMock.EXPECT().Get(gomock.Any()).Return(creds, nil).Times(1)
		verifierMock.EXPECT().Verify(gomock.Any(), desc.Signatures[0], gomock.Nil(), creds).
			Return(nil)
		Expect(verifier.Verify(context.Background(), desc)).To(Succeed())
	})
	It("works with multiple signatures", func() {
		desc := &runtime.Descriptor{
			Signatures: []runtime.Signature{
				{Name: "sig1"},
				{Name: "sig2"},
			},
		}
		verifier := &RSAVerifier{
			log:              log,
			rsaVerifier:      verifierMock,
			targetSignatures: []string{"sig1", "sig2"},
			provider:         providerMock,
		}
		verifyDigestMatchesDescriptor = func(
			ctx context.Context,
			desc *runtime.Descriptor,
			sig runtime.Signature,
			log *slog.Logger) error {
			return nil
		}
		isSafelyDigestible = func(desc *runtime.Component) error { return nil }
		providerMock.EXPECT().Get(gomock.Any()).Return(creds, nil).Times(1)
		verifierMock.EXPECT().Verify(gomock.Any(), desc.Signatures[0], gomock.Nil(), creds).
			Return(nil)
		verifierMock.EXPECT().Verify(gomock.Any(), desc.Signatures[1], gomock.Nil(), creds).
			Return(nil)
		Expect(verifier.Verify(context.Background(), desc)).To(Succeed())
	})
	It("works with 2 descriptors and 1 signature each", func() {
		desc1 := &runtime.Descriptor{
			Signatures: []runtime.Signature{{Name: "sig1"}},
		}
		desc2 := &runtime.Descriptor{
			Signatures: []runtime.Signature{{Name: "sig1"}},
		}
		verifier := &RSAVerifier{
			log:              log,
			rsaVerifier:      verifierMock,
			targetSignatures: []string{"sig1"},
			provider:         providerMock,
		}
		verifyDigestMatchesDescriptor = func(
			ctx context.Context,
			desc *runtime.Descriptor,
			sig runtime.Signature,
			log *slog.Logger) error {
			return nil
		}
		isSafelyDigestible = func(desc *runtime.Component) error { return nil }
		providerMock.EXPECT().Get(gomock.Any()).Return(creds, nil).Times(1)
		verifierMock.EXPECT().Verify(gomock.Any(), desc1.Signatures[0], gomock.Nil(), creds).
			Return(nil)
		verifierMock.EXPECT().Verify(gomock.Any(), desc2.Signatures[0], gomock.Nil(), creds).
			Return(nil)
		Expect(verifier.Verify(context.Background(), desc1, desc2)).To(Succeed())
	})
	It("works with 2 descriptors and multiple signatures each", func() {
		desc1 := &runtime.Descriptor{
			Signatures: []runtime.Signature{
				{Name: "sig1"},
				{Name: "sig2"},
			},
		}
		desc2 := &runtime.Descriptor{
			Signatures: []runtime.Signature{
				{Name: "sig1"},
				{Name: "sig2"},
			},
		}
		verifier := &RSAVerifier{
			log:              log,
			rsaVerifier:      verifierMock,
			targetSignatures: []string{"sig1", "sig2"},
			provider:         providerMock,
		}
		verifyDigestMatchesDescriptor = func(
			ctx context.Context,
			desc *runtime.Descriptor,
			sig runtime.Signature,
			log *slog.Logger) error {
			return nil
		}
		isSafelyDigestible = func(desc *runtime.Component) error { return nil }
		providerMock.EXPECT().Get(gomock.Any()).Return(creds, nil).Times(1)
		for _, desc := range []*runtime.Descriptor{desc1, desc2} {
			for _, sig := range desc.Signatures {
				verifierMock.EXPECT().Verify(gomock.Any(), sig, gomock.Nil(), creds).
					Return(nil)
			}
		}
		Expect(verifier.Verify(context.Background(), desc1, desc2)).To(Succeed())
	})
	It("fails when signature is missing in descriptor", func() {
		desc := &runtime.Descriptor{
			Signatures: []runtime.Signature{{Name: "sig1"}},
		}
		verifier := &RSAVerifier{
			log:              log,
			rsaVerifier:      verifierMock,
			targetSignatures: []string{"sig1", "sig2"},
			provider:         providerMock,
		}
		providerMock.EXPECT().Get(gomock.Any()).Return(creds, nil).Times(1)
		Expect(verifier.Verify(context.Background(), desc)).To(MatchError(
			"ocm descriptor verification failed: signature with name \"sig2\" not found in descriptor"))
	})
	It("fails when digest verification fails", func() {
		desc1 := &runtime.Descriptor{
			Signatures: []runtime.Signature{{Name: "sig1"}},
		}
		desc2 := &runtime.Descriptor{
			Signatures: []runtime.Signature{{Name: "sig1"}},
		}
		verifier := &RSAVerifier{
			log:              log,
			rsaVerifier:      verifierMock,
			targetSignatures: []string{"sig1"},
			provider:         providerMock,
		}
		verifyDigestMatchesDescriptor = func(
			ctx context.Context,
			desc *runtime.Descriptor,
			sig runtime.Signature,
			log *slog.Logger) error {
			if desc == desc1 {
				return fmt.Errorf("digest does not match descriptor")
			}
			return nil
		}
		isSafelyDigestible = func(desc *runtime.Component) error { return nil }
		providerMock.EXPECT().Get(gomock.Any()).Return(creds, nil).Times(1)
		verifierMock.EXPECT().Verify(gomock.Any(), desc2.Signatures[0], gomock.Nil(), creds).
			Return(nil)
		Expect(verifier.Verify(context.Background(), desc1, desc2)).To(MatchError(
			"ocm descriptor verification failed: digest verification failed for signature with name \"sig1\": " +
				"digest does not match descriptor"))
	})
	It("fails when signature verification fails", func() {
		desc := &runtime.Descriptor{
			Signatures: []runtime.Signature{{Name: "sig1"}},
		}
		verifier := &RSAVerifier{
			log:              log,
			rsaVerifier:      verifierMock,
			targetSignatures: []string{"sig1"},
			provider:         providerMock,
		}
		verifyDigestMatchesDescriptor = func(
			ctx context.Context,
			desc *runtime.Descriptor,
			sig runtime.Signature,
			log *slog.Logger) error {
			return nil
		}
		isSafelyDigestible = func(desc *runtime.Component) error { return nil }
		providerMock.EXPECT().Get(gomock.Any()).Return(creds, nil).Times(1)
		verifierMock.EXPECT().Verify(gomock.Any(), desc.Signatures[0], gomock.Nil(), creds).
			Return(fmt.Errorf("signature verification failed"))
		Expect(verifier.Verify(context.Background(), desc)).
			To(MatchError(
				"ocm descriptor verification failed: signature verification failed for signature with name \"sig1\": " +
					"signature verification failed"))
	})
	It("NewRSAVerifier errors with nil signature slice", func() {
		verifier, err := NewRSAVerifier(nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("at least one target signature name must be provided"))
		Expect(verifier).To(BeNil())
	})
	It("NewRSAVerifier errors with empty target signatures", func() {
		var sigs []string
		verifier, err := NewRSAVerifier(sigs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("at least one target signature name must be provided"))
		Expect(verifier).To(BeNil())
	})
	It("NewRSAVerifier errors with empty string as target signature", func() {
		sigs := []string{" ", "", "sig3"}
		verifier, err := NewRSAVerifier(sigs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("signature names cannot be empty or whitespace"))
		Expect(verifier).To(BeNil())
	})
	It("NewRSAVerifier errors with duplicate target signatures", func() {
		sigs := []string{"sig1", "sig2", "sig1"}
		verifier, err := NewRSAVerifier(sigs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("duplicate signature name detected: \"sig1\""))
		Expect(verifier).To(BeNil())
	})
	It("fails when descriptor is not safely digestible", func() {
		desc := &runtime.Descriptor{
			Signatures: []runtime.Signature{{Name: "sig1"}},
		}
		verifier := &RSAVerifier{
			log:              log,
			rsaVerifier:      verifierMock,
			targetSignatures: []string{"sig1"},
			provider:         providerMock,
		}
		isSafelyDigestible = func(desc *runtime.Component) error {
			return fmt.Errorf("descriptor not safely digestible")
		}
		providerMock.EXPECT().Get(gomock.Any()).Return(creds, nil).Times(1)
		Expect(verifier.Verify(context.Background(), desc)).To(MatchError(
			"ocm descriptor verification failed: descriptor is not safely digestible: descriptor not safely digestible"))
	})
	It("one failed verification in multi-descriptor batch fails entire operation", func() {
		desc1 := &runtime.Descriptor{
			Signatures: []runtime.Signature{{Name: "sig1"}},
		}
		desc2 := &runtime.Descriptor{
			Signatures: []runtime.Signature{{Name: "sig1"}},
		}
		verifier := &RSAVerifier{
			log:              log,
			rsaVerifier:      verifierMock,
			targetSignatures: []string{"sig1"},
			provider:         providerMock,
		}
		verifyDigestMatchesDescriptor = func(
			ctx context.Context,
			desc *runtime.Descriptor,
			sig runtime.Signature,
			log *slog.Logger) error {
			return nil
		}
		isSafelyDigestible = func(desc *runtime.Component) error { return nil }
		providerMock.EXPECT().Get(gomock.Any()).Return(creds, nil).Times(1)
		verifierMock.EXPECT().Verify(gomock.Any(), desc1.Signatures[0], gomock.Nil(), creds).
			Return(nil)
		verifierMock.EXPECT().Verify(gomock.Any(), desc2.Signatures[0], gomock.Nil(), creds).
			Return(fmt.Errorf("signature verification failed"))
		err := verifier.Verify(context.Background(), desc1, desc2)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("signature verification failed"))
	})
	It("verification with multiple signatures where one fails", func() {
		desc := &runtime.Descriptor{
			Signatures: []runtime.Signature{
				{Name: "sig1"},
				{Name: "sig2"},
			},
		}
		verifier := &RSAVerifier{
			log:              log,
			rsaVerifier:      verifierMock,
			targetSignatures: []string{"sig1", "sig2"},
			provider:         providerMock,
		}
		verifyDigestMatchesDescriptor = func(
			ctx context.Context,
			desc *runtime.Descriptor,
			sig runtime.Signature,
			log *slog.Logger) error {
			return nil
		}
		isSafelyDigestible = func(desc *runtime.Component) error { return nil }
		providerMock.EXPECT().Get(gomock.Any()).Return(creds, nil).Times(1)
		verifierMock.EXPECT().Verify(gomock.Any(), desc.Signatures[0], gomock.Nil(), creds).
			Return(nil)
		verifierMock.EXPECT().Verify(gomock.Any(), desc.Signatures[1], gomock.Nil(), creds).
			Return(fmt.Errorf("signature verification failed"))
		err := verifier.Verify(context.Background(), desc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("signature verification failed"))
	})
	It("verifies only target signatures when descriptor has additional signatures", func() {
		desc := &runtime.Descriptor{
			Signatures: []runtime.Signature{
				{Name: "sig1"},
				{Name: "sig2"},
				{Name: "sig3"},
			},
		}
		verifier := &RSAVerifier{
			log:              log,
			rsaVerifier:      verifierMock,
			targetSignatures: []string{"sig1", "sig2"},
			provider:         providerMock,
		}
		verifyDigestMatchesDescriptor = func(
			ctx context.Context,
			desc *runtime.Descriptor,
			sig runtime.Signature,
			log *slog.Logger) error {
			return nil
		}
		isSafelyDigestible = func(desc *runtime.Component) error { return nil }
		providerMock.EXPECT().Get(gomock.Any()).Return(creds, nil).Times(1)
		verifierMock.EXPECT().Verify(gomock.Any(), desc.Signatures[0], gomock.Nil(), creds).
			Return(nil)
		verifierMock.EXPECT().Verify(gomock.Any(), desc.Signatures[1], gomock.Nil(), creds).
			Return(nil)
		Expect(verifier.Verify(context.Background(), desc)).To(Succeed())
	})
	It("passes nil credentials when provider returns nil", func() {
		desc := &runtime.Descriptor{
			Signatures: []runtime.Signature{{Name: "sig1"}},
		}
		verifier := &RSAVerifier{
			log:              log,
			rsaVerifier:      verifierMock,
			targetSignatures: []string{"sig1"},
			provider:         providerMock,
		}
		verifyDigestMatchesDescriptor = func(
			ctx context.Context,
			desc *runtime.Descriptor,
			sig runtime.Signature,
			log *slog.Logger) error {
			return nil
		}
		isSafelyDigestible = func(desc *runtime.Component) error { return nil }
		providerMock.EXPECT().Get(gomock.Any()).Return(nil, nil).Times(1)
		verifierMock.EXPECT().Verify(gomock.Any(), desc.Signatures[0], gomock.Nil(), gomock.Nil()).
			Return(nil)
		Expect(verifier.Verify(context.Background(), desc)).To(Succeed())
	})
	It("fails when provider returns an error", func() {
		desc := &runtime.Descriptor{
			Signatures: []runtime.Signature{{Name: "sig1"}},
		}
		verifier := &RSAVerifier{
			log:              log,
			rsaVerifier:      verifierMock,
			targetSignatures: []string{"sig1"},
			provider:         providerMock,
		}
		providerMock.EXPECT().Get(gomock.Any()).Return(nil, fmt.Errorf("provider stopped")).Times(1)
		err := verifier.Verify(context.Background(), desc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("get credentials from provider"))
		Expect(err.Error()).To(ContainSubstring("provider stopped"))
	})
})
