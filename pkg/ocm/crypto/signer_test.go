package crypto

import (
	"context"
	"fmt"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto/internal/mocks"
	"go.uber.org/mock/gomock"
	credv1 "ocm.software/open-component-model/bindings/go/credentials/spec/config/v1"
	"ocm.software/open-component-model/bindings/go/descriptor/runtime"
	rsav1alpha1 "ocm.software/open-component-model/bindings/go/rsa/signing/v1alpha1"
)

var _ = Describe("RSASigner", func() {
	var (
		log          = logr.Discard()
		creds        = map[string]string{"public_key_pem": "test-cert", "private_key_pem": "test-key"}
		typedCreds   = &credv1.DirectCredentials{Properties: creds}
		digesterMock *mocks.MockDigester
		signerMock   *mocks.MockSigner
		providerMock *mocks.MockRSACredentialProvider
		mockCtrl     *gomock.Controller
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		digesterMock = mocks.NewMockDigester(mockCtrl)
		signerMock = mocks.NewMockSigner(mockCtrl)
		providerMock = mocks.NewMockRSACredentialProvider(mockCtrl)
	})
	AfterEach(func() {
		mockCtrl.Finish()
	})
	It("works with 1 signature", func() {
		sigs := []string{"sig1"}
		desc := &runtime.Descriptor{}
		cfg := &rsav1alpha1.Config{}
		dig := runtime.Digest{
			HashAlgorithm:          "SHA256",
			NormalisationAlgorithm: "json/v4alpha1",
			Value:                  "test-digest",
		}
		sig := runtime.SignatureInfo{
			Algorithm: string(rsav1alpha1.AlgorithmRSASSAPSS),
			Value:     "sig_data",
			MediaType: string(rsav1alpha1.SignatureEncodingPolicyPEM),
		}
		digesterMock.EXPECT().GenerateDigest(gomock.Any(), desc).
			Times(1).Return(&dig, nil)
		providerMock.EXPECT().Get(gomock.Any()).Return(creds, nil).Times(1)
		signerMock.EXPECT().Sign(gomock.Any(), dig, cfg, typedCreds).Times(1).Return(sig, nil)
		signer := &RSASigner{
			log:              log,
			rsaSigner:        signerMock,
			targetSignatures: sigs,
			provider:         providerMock,
			rsaConfig:        cfg,
			digester:         digesterMock,
		}
		err := signer.Sign(context.Background(), desc)
		Expect(err).ToNot(HaveOccurred())
		Expect(desc.Signatures).To(HaveLen(len(sigs)))
		Expect(desc.Signatures[0].Signature).To(Equal(sig))
		Expect(desc.Signatures[0].Digest).To(Equal(dig))
		Expect(desc.Signatures[0].Name).To(Equal(sigs[0]))
	})
	It("works with 3 signatures", func() {
		sigs := []string{"sig1", "sig2", "sig3"}
		desc := &runtime.Descriptor{}
		cfg := &rsav1alpha1.Config{}
		dig := runtime.Digest{
			HashAlgorithm:          "SHA256",
			NormalisationAlgorithm: "json/v4alpha1",
			Value:                  "test-digest",
		}
		sig := runtime.SignatureInfo{
			Algorithm: string(rsav1alpha1.AlgorithmRSASSAPSS),
			Value:     "sig_data",
			MediaType: string(rsav1alpha1.SignatureEncodingPolicyPEM),
		}
		digesterMock.EXPECT().GenerateDigest(gomock.Any(), desc).
			Times(1).Return(&dig, nil)
		providerMock.EXPECT().Get(gomock.Any()).Return(creds, nil).Times(1)
		signerMock.EXPECT().Sign(gomock.Any(), dig, cfg, typedCreds).Times(3).Return(sig, nil)
		signer := &RSASigner{
			log:              log,
			rsaSigner:        signerMock,
			targetSignatures: sigs,
			provider:         providerMock,
			rsaConfig:        cfg,
			digester:         digesterMock,
		}
		err := signer.Sign(context.Background(), desc)
		Expect(err).ToNot(HaveOccurred())
		Expect(desc.Signatures).To(HaveLen(len(sigs)))
		for i := 0; i < len(sigs); i++ {
			Expect(desc.Signatures[i].Signature).To(Equal(sig))
			Expect(desc.Signatures[i].Digest).To(Equal(dig))
			Expect(slices.ContainsFunc(desc.Signatures, func(s runtime.Signature) bool {
				return s.Name == sigs[i]
			})).To(BeTrue())
		}
	})
	It("will return an error if generating the digest fails", func() {
		desc := &runtime.Descriptor{}
		digesterMock.EXPECT().GenerateDigest(gomock.Any(), desc).Times(1).Return(nil, fmt.Errorf("digest generation failed"))
		signer := &RSASigner{
			log:              log,
			rsaSigner:        signerMock,
			targetSignatures: []string{"sig1", "sig"},
			provider:         providerMock,
			rsaConfig:        &rsav1alpha1.Config{},
			digester:         digesterMock,
		}
		err := signer.Sign(context.Background(), desc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("digest generation failed"))
		Expect(desc.Signatures).To(BeEmpty())
	})
	It("will return an error if signing fails", func() {
		desc := &runtime.Descriptor{}
		cfg := &rsav1alpha1.Config{}
		dig := runtime.Digest{
			HashAlgorithm:          "SHA256",
			NormalisationAlgorithm: "json/v4alpha1",
			Value:                  "test-digest",
		}
		digesterMock.EXPECT().GenerateDigest(gomock.Any(), desc).Times(1).Return(&dig, nil)
		providerMock.EXPECT().Get(gomock.Any()).Return(creds, nil).Times(1)
		signerMock.EXPECT().Sign(gomock.Any(), dig, cfg, typedCreds).
			Times(2).
			Return(runtime.SignatureInfo{}, fmt.Errorf("signing failed"))
		signer := &RSASigner{
			log:              log,
			rsaSigner:        signerMock,
			targetSignatures: []string{"sig1", "sig2"},
			provider:         providerMock,
			rsaConfig:        cfg,
			digester:         digesterMock,
		}
		err := signer.Sign(context.Background(), desc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("signing failed"))
		Expect(desc.Signatures).To(BeEmpty())
	})
	It("no signature will be added when signing fails with 1 signature", func() {
		desc := &runtime.Descriptor{}
		cfg := &rsav1alpha1.Config{}
		dig := runtime.Digest{
			HashAlgorithm:          "SHA256",
			NormalisationAlgorithm: "json/v4alpha1",
			Value:                  "test-digest",
		}
		digesterMock.EXPECT().GenerateDigest(gomock.Any(), desc).Times(1).Return(&dig, nil)
		providerMock.EXPECT().Get(gomock.Any()).Return(creds, nil).Times(1)
		signerMock.EXPECT().Sign(gomock.Any(), dig, cfg, typedCreds).
			Times(1).
			Return(runtime.SignatureInfo{}, fmt.Errorf("signing failed"))
		signer := &RSASigner{
			log:              log,
			rsaSigner:        signerMock,
			targetSignatures: []string{"sig1"},
			provider:         providerMock,
			rsaConfig:        cfg,
			digester:         digesterMock,
		}
		err := signer.Sign(context.Background(), desc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("signing failed"))
		Expect(desc.Signatures).To(BeEmpty())
	})
	It("will return an error if the signature already exists in the descriptor", func() {
		desc := &runtime.Descriptor{
			Signatures: []runtime.Signature{{Name: "sig1"}},
		}
		signer := &RSASigner{
			log:              log,
			rsaSigner:        signerMock,
			targetSignatures: []string{"sig1", "sig2"},
			provider:         providerMock,
			rsaConfig:        &rsav1alpha1.Config{},
			digester:         digesterMock,
		}
		err := signer.Sign(context.Background(), desc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("signature with name \"sig1\" already exists"))
		Expect(desc.Signatures).To(HaveLen(1))
		Expect(desc.Signatures[0].Name).To(Equal("sig1"))
	})
	It("one failed signature will cause the whole signing process - no signatures added", func() {
		desc := &runtime.Descriptor{}
		cfg := &rsav1alpha1.Config{}
		dig := runtime.Digest{
			HashAlgorithm:          "SHA256",
			NormalisationAlgorithm: "json/v4alpha1",
			Value:                  "test-digest",
		}
		digesterMock.EXPECT().GenerateDigest(gomock.Any(), desc).Times(1).Return(&dig, nil)
		providerMock.EXPECT().Get(gomock.Any()).Return(creds, nil).Times(1)
		signerMock.EXPECT().Sign(gomock.Any(), dig, cfg, typedCreds).Times(1).
			Return(runtime.SignatureInfo{
				Algorithm: string(rsav1alpha1.AlgorithmRSASSAPSS),
				Value:     "sig_data",
				MediaType: string(rsav1alpha1.SignatureEncodingPolicyPEM),
			}, nil)
		signerMock.EXPECT().Sign(gomock.Any(), dig, cfg, typedCreds).Times(1).
			Return(runtime.SignatureInfo{}, fmt.Errorf("signing failed"))
		signer := &RSASigner{
			log:              log,
			rsaSigner:        signerMock,
			targetSignatures: []string{"sig1", "sig2"},
			provider:         providerMock,
			rsaConfig:        cfg,
			digester:         digesterMock,
		}
		err := signer.Sign(context.Background(), desc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("signing failed"))
		Expect(desc.Signatures).To(BeEmpty())
	})
	It("NewRSASigner errors with nil provider", func() {
		signer, err := NewRSASigner(nil, []string{"sig1"})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("credential provider is required"))
		Expect(signer).To(BeNil())
	})
	It("NewRSASigner errors with nil signature slice", func() {
		signer, err := NewRSASigner(providerMock, nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("at least one target signature name must be provided"))
		Expect(signer).To(BeNil())
	})
	It("NewRSASigner errors with empty target signatures", func() {
		var sigs []string
		signer, err := NewRSASigner(providerMock, sigs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("at least one target signature name must be provided"))
		Expect(signer).To(BeNil())
	})
	It("NewRSASigner errors with empty string as target signature", func() {
		sigs := []string{" ", "", "sig3"}
		signer, err := NewRSASigner(providerMock, sigs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("signature names cannot be empty or whitespace"))
		Expect(signer).To(BeNil())
	})
	It("NewRSASigner errors with duplicate target signatures", func() {
		sigs := []string{"sig1", "sig2", "sig1"}
		signer, err := NewRSASigner(providerMock, sigs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("duplicate signature name detected: \"sig1\""))
		Expect(signer).To(BeNil())
	})
	It("fails when credentials are nil", func() {
		desc := &runtime.Descriptor{}
		dig := runtime.Digest{
			HashAlgorithm:          "SHA256",
			NormalisationAlgorithm: "json/v4alpha1",
			Value:                  "test-digest",
		}
		digesterMock.EXPECT().GenerateDigest(gomock.Any(), desc).Times(1).Return(&dig, nil)
		providerMock.EXPECT().Get(gomock.Any()).Return(nil, nil).Times(1)
		signer := &RSASigner{
			log:              log,
			rsaSigner:        signerMock,
			targetSignatures: []string{"sig1"},
			provider:         providerMock,
			rsaConfig:        &rsav1alpha1.Config{},
			digester:         digesterMock,
		}
		err := signer.Sign(context.Background(), desc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("signing credentials are not available"))
		Expect(desc.Signatures).To(BeEmpty())
	})
	It("fails when provider returns an error", func() {
		desc := &runtime.Descriptor{}
		dig := runtime.Digest{
			HashAlgorithm:          "SHA256",
			NormalisationAlgorithm: "json/v4alpha1",
			Value:                  "test-digest",
		}
		digesterMock.EXPECT().GenerateDigest(gomock.Any(), desc).Times(1).Return(&dig, nil)
		providerMock.EXPECT().Get(gomock.Any()).Return(nil, fmt.Errorf("provider stopped")).Times(1)
		signer := &RSASigner{
			log:              log,
			rsaSigner:        signerMock,
			targetSignatures: []string{"sig1"},
			provider:         providerMock,
			rsaConfig:        &rsav1alpha1.Config{},
			digester:         digesterMock,
		}
		err := signer.Sign(context.Background(), desc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("get credentials from provider"))
		Expect(err.Error()).To(ContainSubstring("provider stopped"))
		Expect(desc.Signatures).To(BeEmpty())
	})
	It("appends signatures to existing signatures in descriptor", func() {
		desc := &runtime.Descriptor{
			Signatures: []runtime.Signature{{Name: "existing-sig"}},
		}
		cfg := &rsav1alpha1.Config{}
		dig := runtime.Digest{
			HashAlgorithm:          "SHA256",
			NormalisationAlgorithm: "json/v4alpha1",
			Value:                  "test-digest",
		}
		sig := runtime.SignatureInfo{
			Algorithm: string(rsav1alpha1.AlgorithmRSASSAPSS),
			Value:     "sig_data",
			MediaType: string(rsav1alpha1.SignatureEncodingPolicyPEM),
		}
		digesterMock.EXPECT().GenerateDigest(gomock.Any(), desc).Times(1).Return(&dig, nil)
		providerMock.EXPECT().Get(gomock.Any()).Return(creds, nil).Times(1)
		signerMock.EXPECT().Sign(gomock.Any(), dig, cfg, typedCreds).Times(1).Return(sig, nil)
		signer := &RSASigner{
			log:              log,
			rsaSigner:        signerMock,
			targetSignatures: []string{"new-sig"},
			provider:         providerMock,
			rsaConfig:        cfg,
			digester:         digesterMock,
		}
		err := signer.Sign(context.Background(), desc)
		Expect(err).ToNot(HaveOccurred())
		Expect(desc.Signatures).To(HaveLen(2))
		Expect(desc.Signatures[0].Name).To(Equal("existing-sig"))
		Expect(desc.Signatures[1].Name).To(Equal("new-sig"))
		Expect(desc.Signatures[1].Signature).To(Equal(sig))
		Expect(desc.Signatures[1].Digest).To(Equal(dig))
	})
	It("works with 2 signatures using parallel path", func() {
		sigs := []string{"sig1", "sig2"}
		desc := &runtime.Descriptor{}
		cfg := &rsav1alpha1.Config{}
		dig := runtime.Digest{
			HashAlgorithm:          "SHA256",
			NormalisationAlgorithm: "json/v4alpha1",
			Value:                  "test-digest",
		}
		sig := runtime.SignatureInfo{
			Algorithm: string(rsav1alpha1.AlgorithmRSASSAPSS),
			Value:     "sig_data",
			MediaType: string(rsav1alpha1.SignatureEncodingPolicyPEM),
		}
		digesterMock.EXPECT().GenerateDigest(gomock.Any(), desc).Times(1).Return(&dig, nil)
		providerMock.EXPECT().Get(gomock.Any()).Return(creds, nil).Times(1)
		signerMock.EXPECT().Sign(gomock.Any(), dig, cfg, typedCreds).Times(2).Return(sig, nil)
		signer := &RSASigner{
			log:              log,
			rsaSigner:        signerMock,
			targetSignatures: sigs,
			provider:         providerMock,
			rsaConfig:        cfg,
			digester:         digesterMock,
		}
		err := signer.Sign(context.Background(), desc)
		Expect(err).ToNot(HaveOccurred())
		Expect(desc.Signatures).To(HaveLen(2))
		for i := 0; i < len(sigs); i++ {
			Expect(desc.Signatures[i].Signature).To(Equal(sig))
			Expect(desc.Signatures[i].Digest).To(Equal(dig))
			Expect(slices.ContainsFunc(desc.Signatures, func(s runtime.Signature) bool {
				return s.Name == sigs[i]
			})).To(BeTrue())
		}
	})
	It("NewRSASigner applies WithSignerLogger option", func() {
		customLog := logr.Discard().WithName("custom")
		signer, err := NewRSASigner(providerMock, []string{"sig1"}, WithSignerLogger(customLog))
		Expect(err).ToNot(HaveOccurred())
		Expect(signer).ToNot(BeNil())
	})
	It("NewRSASigner applies WithNamedSignerLogger option", func() {
		signer, err := NewRSASigner(providerMock, []string{"sig1"}, WithNamedSignerLogger(logr.Discard()))
		Expect(err).ToNot(HaveOccurred())
		Expect(signer).ToNot(BeNil())
	})
	It("NewRSASigner applies WithDigester option", func() {
		signer, err := NewRSASigner(providerMock, []string{"sig1"}, WithDigester(digesterMock))
		Expect(err).ToNot(HaveOccurred())
		Expect(signer).ToNot(BeNil())
		Expect(signer.digester).To(Equal(digesterMock))
	})
})
