package ocm

import (
	"context"
	"crypto"
	"errors"
	"testing"

	mocks2 "github.com/konfidence-project/konfidence/internal/vectorassembly/internal/ocm/mocks"
	"github.com/konfidence-project/konfidence/internal/vectorassembly/internal/vector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	norm "ocm.software/open-component-model/bindings/go/descriptor/normalisation/json/v4alpha1"
	"ocm.software/open-component-model/bindings/go/runtime"
)

func TestOcmAdapter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OCM Adapter Suite")
}

var _ = Describe("Adapter", func() {
	Describe("CreateVector", func() {
		var (
			ctrl         *gomock.Controller
			mockClient   *mocks2.MockClient
			mockSigner   *mocks2.MockSigner
			mockDigester *mocks2.MockDigester
			mockVerifier *mocks2.MockVerifier
			adapter      Adapter
			ctx          context.Context
			repoSpec     runtime.Typed
			v            vector.Vector
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			mockClient = mocks2.NewMockClient(ctrl)
			mockSigner = mocks2.NewMockSigner(ctrl)
			mockDigester = mocks2.NewMockDigester(ctrl)
			mockVerifier = mocks2.NewMockVerifier(ctrl)

			adapter = Adapter{
				ocmClient:    mockClient,
				vectorSigner: mockSigner,
				digester:     mockDigester,
				verifier:     mockVerifier,
			}

			ctx = context.Background()
			repoSpec = &runtime.Unstructured{
				Data: map[string]interface{}{
					"type":    "oci",
					"baseUrl": "http://localhost:5100",
				},
			}
			v = vector.Vector{
				Name:    "test-vector",
				Version: "1.0.0",
				Artifacts: []vector.Artifact{
					{
						Name:    "artifact1",
						Version: "2.0.0",
						Digest:  "sha256:abc123",
						SourceRepo: &runtime.Unstructured{
							Data: map[string]interface{}{
								"type":    "oci",
								"baseUrl": "http://localhost:5100",
							},
						},
					},
				},
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should successfully create a vector", func() {
			mockClient.EXPECT().Copy(ctx, gomock.Any(), repoSpec).Return(nil)
			mockDigester.EXPECT().GetHashAlgorithm().Return(crypto.SHA256.String()).AnyTimes()
			mockDigester.EXPECT().GetNormalisationAlgorithm().Return(norm.Algorithm).AnyTimes()
			mockSigner.EXPECT().Sign(ctx, gomock.Any()).Return(nil)
			mockClient.EXPECT().Save(ctx, repoSpec, gomock.Any()).Return(nil)

			err := adapter.CreateVector(ctx, repoSpec, v)

			Expect(err).ToNot(HaveOccurred())
		})

		It("should return error when Copy artifacts fails", func() {
			copyErr := errors.New("failed to copy artifacts")

			mockClient.EXPECT().Copy(ctx, gomock.Any(), repoSpec).Return(copyErr)

			err := adapter.CreateVector(ctx, repoSpec, v)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to copy artifact components"))
			Expect(errors.Is(err, copyErr)).To(BeTrue())
		})

		It("should return error when Sign fails", func() {
			signErr := errors.New("signing failed")

			mockClient.EXPECT().Copy(ctx, gomock.Any(), repoSpec).Return(nil)
			mockDigester.EXPECT().GetHashAlgorithm().Return(crypto.SHA256.String()).AnyTimes()
			mockDigester.EXPECT().GetNormalisationAlgorithm().Return(norm.Algorithm).AnyTimes()
			mockSigner.EXPECT().Sign(ctx, gomock.Any()).Return(signErr)

			err := adapter.CreateVector(ctx, repoSpec, v)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to Sign ocm descriptor"))
			Expect(errors.Is(err, signErr)).To(BeTrue())
		})

		It("should return error when Save fails with non-ErrComponentAlreadyExists error", func() {
			saveErr := errors.New("repository unavailable")

			mockClient.EXPECT().Copy(ctx, gomock.Any(), repoSpec).Return(nil)
			mockDigester.EXPECT().GetHashAlgorithm().Return(crypto.SHA256.String()).AnyTimes()
			mockDigester.EXPECT().GetNormalisationAlgorithm().Return(norm.Algorithm).AnyTimes()
			mockSigner.EXPECT().Sign(ctx, gomock.Any()).Return(nil)
			mockClient.EXPECT().Save(ctx, repoSpec, gomock.Any()).Return(saveErr)

			err := adapter.CreateVector(ctx, repoSpec, v)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to save ocm descriptor"))
			Expect(errors.Is(err, saveErr)).To(BeTrue())
		})
	})
})
