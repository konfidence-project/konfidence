package ocm

import (
	"context"
	"crypto"
	"errors"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
	norm "ocm.software/open-component-model/bindings/go/descriptor/normalisation/json/v4alpha1"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"ocm.software/open-component-model/bindings/go/runtime"

	"github.com/konfidence-project/konfidence/internal/galaxy/vector-assembly/internal/controller/domain"
	"github.com/konfidence-project/konfidence/internal/galaxy/vector-assembly/pkg/ocm/mocks"
)

func TestOcmAdapter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OCM Adapter Suite")
}

var _ = Describe("Adapter", func() {
	Describe("CreateVector", func() {
		var (
			ctrl         *gomock.Controller
			mockClient   *mocks.MockClient
			mockSigner   *mocks.MockSigner
			mockDigester *mocks.MockDigester
			mockVerifier *mocks.MockVerifier
			adapter      Adapter
			ctx          context.Context
			repoSpec     runtime.Typed
			vector       domain.Vector
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			mockClient = mocks.NewMockClient(ctrl)
			mockSigner = mocks.NewMockSigner(ctrl)
			mockDigester = mocks.NewMockDigester(ctrl)
			mockVerifier = mocks.NewMockVerifier(ctrl)

			adapter = Adapter{
				ocmClient:        mockClient,
				vectorSigner:     mockSigner,
				digester:         mockDigester,
				vectorVerifier:   mockVerifier,
				artifactVerifier: mockVerifier,
			}

			ctx = context.Background()
			repoSpec = &runtime.Unstructured{
				Data: map[string]interface{}{
					"type":    "oci",
					"baseUrl": "http://localhost:5100",
				},
			}
			vector = domain.Vector{
				Name:    "test-vector",
				Version: "1.0.0",
				Artifacts: []domain.Artifact{
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

		It("should successfully create vector with alias", func() {
			alias := "latest"
			var capturedAliasRef compref.Ref
			var capturedAlias string

			// Setup expectations
			mockClient.EXPECT().Copy(ctx, gomock.Any(), repoSpec).Return(nil)
			mockDigester.EXPECT().GetHashAlgorithm().Return(crypto.SHA256).AnyTimes()
			mockDigester.EXPECT().GetNormalisationAlgorithm().Return(norm.Algorithm).AnyTimes()
			mockSigner.EXPECT().Sign(ctx, gomock.Any()).Return(nil)
			mockClient.EXPECT().Save(ctx, repoSpec, gomock.Any()).Return(nil)
			mockClient.EXPECT().
				AddAlias(ctx, gomock.Any(), alias).
				DoAndReturn(func(ctx context.Context, ref compref.Ref, a string) error {
					capturedAliasRef = ref
					capturedAlias = a
					return nil
				})

			err := adapter.CreateVector(ctx, repoSpec, vector, alias)

			Expect(err).ToNot(HaveOccurred())
			Expect(capturedAlias).To(Equal(alias))
			Expect(capturedAliasRef.Component).To(Equal(vector.Name))
			Expect(capturedAliasRef.Version).To(Equal(vector.Version))
			Expect(capturedAliasRef.Repository).To(Equal(repoSpec))
		})

		It("should return error when AddAlias fails", func() {
			alias := "edge"
			aliasErr := errors.New("alias creation failed")

			mockClient.EXPECT().Copy(ctx, gomock.Any(), repoSpec).Return(nil)
			mockDigester.EXPECT().GetHashAlgorithm().Return(crypto.SHA256).AnyTimes()
			mockDigester.EXPECT().GetNormalisationAlgorithm().Return(norm.Algorithm).AnyTimes()
			mockSigner.EXPECT().Sign(ctx, gomock.Any()).Return(nil)
			mockClient.EXPECT().Save(ctx, repoSpec, gomock.Any()).Return(nil)
			mockClient.EXPECT().AddAlias(ctx, gomock.Any(), alias).Return(aliasErr)

			err := adapter.CreateVector(ctx, repoSpec, vector, alias)

			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, aliasErr)).To(BeTrue())
		})

		It("should return error when Copy artifacts fails", func() {
			alias := "latest"
			copyErr := errors.New("failed to copy artifacts")

			mockClient.EXPECT().Copy(ctx, gomock.Any(), repoSpec).Return(copyErr)

			err := adapter.CreateVector(ctx, repoSpec, vector, alias)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to copy artifact components"))
			Expect(errors.Is(err, copyErr)).To(BeTrue())
		})

		It("should return error when Sign fails", func() {
			alias := "edge"
			signErr := errors.New("signing failed")

			mockClient.EXPECT().Copy(ctx, gomock.Any(), repoSpec).Return(nil)
			mockDigester.EXPECT().GetHashAlgorithm().Return(crypto.SHA256).AnyTimes()
			mockDigester.EXPECT().GetNormalisationAlgorithm().Return(norm.Algorithm).AnyTimes()
			mockSigner.EXPECT().Sign(ctx, gomock.Any()).Return(signErr)

			err := adapter.CreateVector(ctx, repoSpec, vector, alias)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to Sign ocm descriptor"))
			Expect(errors.Is(err, signErr)).To(BeTrue())
		})

		It("should return error when Save fails with non-ErrComponentAlreadyExists error", func() {
			alias := "prod"
			saveErr := errors.New("repository unavailable")

			mockClient.EXPECT().Copy(ctx, gomock.Any(), repoSpec).Return(nil)
			mockDigester.EXPECT().GetHashAlgorithm().Return(crypto.SHA256).AnyTimes()
			mockDigester.EXPECT().GetNormalisationAlgorithm().Return(norm.Algorithm).AnyTimes()
			mockSigner.EXPECT().Sign(ctx, gomock.Any()).Return(nil)
			mockClient.EXPECT().Save(ctx, repoSpec, gomock.Any()).Return(saveErr)

			err := adapter.CreateVector(ctx, repoSpec, vector, alias)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to save ocm descriptor"))
			Expect(errors.Is(err, saveErr)).To(BeTrue())
		})
	})
})
