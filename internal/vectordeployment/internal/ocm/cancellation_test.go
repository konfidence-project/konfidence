package ocm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/vectordeployment/internal/controller"
	"github.com/konfidence-project/konfidence/internal/vectordeployment/internal/ocm/mocks"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"go.uber.org/mock/gomock"
	ocmblob "ocm.software/open-component-model/bindings/go/blob"
	"ocm.software/open-component-model/bindings/go/credentials"
	descruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	ociblob "ocm.software/open-component-model/bindings/go/oci/blob"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ociv1 "ocm.software/open-component-model/bindings/go/oci/spec/repository/v1/oci"
)

type verifierFunc func(context.Context, credentials.Resolver, []crypto.SignatureSpec, []*descruntime.Descriptor) error

func (f verifierFunc) Verify(ctx context.Context, resolver credentials.Resolver, specs []crypto.SignatureSpec, descriptors []*descruntime.Descriptor) error {
	return f(ctx, resolver, specs, descriptors)
}

type countingReadCloser struct {
	io.Reader
	closeCalls atomic.Int32
}

func (r *countingReadCloser) Close() error {
	r.closeCalls.Add(1)
	return nil
}

type readStartedReadCloser struct {
	io.ReadCloser
	started chan<- struct{}
}

func (r *readStartedReadCloser) Read(p []byte) (int, error) {
	select {
	case r.started <- struct{}{}:
	default:
	}

	return r.ReadCloser.Read(p)
}

var _ = Describe("OCM adapter cancellation", func() {
	var (
		mockCtrl    *gomock.Controller
		mockClient  *mocks.MockClient
		artifactRef compref.Ref
		resource    descruntime.Resource
		descriptor  descruntime.Descriptor
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockClient(mockCtrl)

		artifactRef = compref.Ref{
			Repository: &ociv1.Repository{BaseUrl: "https://registry.example.com"},
			Component:  "example.com/artifact",
			Version:    "1.0.0",
		}

		resource = descruntime.Resource{
			ElementMeta: descruntime.ElementMeta{
				ObjectMeta: descruntime.ObjectMeta{Name: "artifact-manifest"},
			},
			Type: konfidenceResourceTypeArtifactManifest,
		}

		descriptor = descruntime.Descriptor{
			Component: descruntime.Component{
				ComponentMeta: descruntime.ComponentMeta{
					ObjectMeta: descruntime.ObjectMeta{
						Name:    artifactRef.Component,
						Version: artifactRef.Version,
					},
				},
				Resources: []descruntime.Resource{resource},
			},
		}
	})

	It("returns context cancellation instead of a permanent verification error", func() {
		ctx, cancel := context.WithCancel(context.Background())

		adapter := Adapter{
			client: mockClient,
			verifier: verifierFunc(func(context.Context, credentials.Resolver, []crypto.SignatureSpec, []*descruntime.Descriptor) error {
				cancel()
				return errors.New("verification interrupted")
			}),
		}

		mockClient.EXPECT().Get(gomock.Any(), artifactRef).Return(descriptor, nil)

		_, err := adapter.GetArtifactManifestByReference(ctx, artifactRef)

		Expect(errors.Is(err, context.Canceled)).To(BeTrue())

		var permanent *controller.PermanentError
		Expect(errors.As(err, &permanent)).To(BeFalse())
	})

	It("returns context cancellation while reading a descriptor blob", func() {
		const content = `{"type":"test.konfidence.cloud","allowReuse":true}`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))

			// Send only part of the blob so io.ReadAll waits for more data.
			_, _ = io.WriteString(w, content[:1])
			w.(http.Flusher).Flush()

			<-r.Context().Done()
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		Expect(err).NotTo(HaveOccurred())

		response, err := http.DefaultClient.Do(request)
		Expect(err).NotTo(HaveOccurred())

		readStarted := make(chan struct{}, 1)
		reader := &readStartedReadCloser{
			ReadCloser: response.Body,
			started:    readStarted,
		}

		descriptorBlob := ociblob.NewDescriptorBlob(
			ocmblob.ReadOnlyBlobFunc(func() (io.ReadCloser, error) {
				return reader, nil
			}),
			ocispec.Descriptor{
				Digest: digest.FromString(content),
				Size:   int64(len(content)),
			},
		)

		adapter := NewAdapterWithClient(mockClient)
		mockClient.EXPECT().Get(gomock.Any(), artifactRef).Return(descriptor, nil)
		mockClient.EXPECT().
			GetLocalResource(gomock.Any(), artifactRef, resource.ToIdentity()).
			Return(descriptorBlob, &resource, nil)

		result := make(chan error, 1)
		go func() {
			_, err := adapter.GetArtifactManifestByReference(ctx, artifactRef)
			result <- err
		}()

		// Ensure GetLocalResource has returned and io.ReadAll is reading the blob.
		Eventually(readStarted).Should(Receive())

		cancel()

		var resultErr error
		Eventually(result).Should(Receive(&resultErr))
		Expect(errors.Is(resultErr, context.Canceled)).To(BeTrue())
	})

	It("closes a normally consumed descriptor blob exactly once", func() {
		content := []byte(`{"type":"test.konfidence.cloud","allowReuse":true}`)
		reader := &countingReadCloser{
			Reader: bytes.NewReader(content),
		}

		descriptorBlob := ociblob.NewDescriptorBlob(
			ocmblob.ReadOnlyBlobFunc(func() (io.ReadCloser, error) {
				return reader, nil
			}),
			ocispec.Descriptor{
				Digest: digest.FromBytes(content),
				Size:   int64(len(content)),
			},
		)

		adapter := NewAdapterWithClient(mockClient)

		mockClient.EXPECT().Get(gomock.Any(), artifactRef).Return(descriptor, nil)
		mockClient.EXPECT().
			GetLocalResource(gomock.Any(), artifactRef, resource.ToIdentity()).
			Return(descriptorBlob, &resource, nil)

		manifest, err := adapter.GetArtifactManifestByReference(
			context.Background(),
			artifactRef,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(manifest.Type).To(Equal(
			konfidence.MustArtifactType("test.konfidence.cloud"),
		))
		Expect(reader.closeCalls.Load()).To(Equal(int32(1)))
	})
})
