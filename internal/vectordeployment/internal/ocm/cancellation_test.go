package ocm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/vectordeployment/internal/controller"
	"github.com/konfidence-project/konfidence/internal/vectordeployment/internal/ocm/mocks"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"ocm.software/open-component-model/bindings/go/blob"
	"ocm.software/open-component-model/bindings/go/credentials"
	descruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ociv1 "ocm.software/open-component-model/bindings/go/oci/spec/repository/v1/oci"
)

type verifierFunc func(context.Context, credentials.Resolver, []crypto.SignatureSpec, []*descruntime.Descriptor) error

func (f verifierFunc) Verify(ctx context.Context, resolver credentials.Resolver, specs []crypto.SignatureSpec, descriptors []*descruntime.Descriptor) error {
	return f(ctx, resolver, specs, descriptors)
}

type readCloserBlob struct {
	reader io.ReadCloser
}

var _ blob.ReadOnlyBlob = readCloserBlob{}

func (b readCloserBlob) ReadCloser() (io.ReadCloser, error) {
	return b.reader, nil
}

type blockingReadCloser struct {
	started    chan struct{}
	closed     chan struct{}
	startOnce  sync.Once
	closeOnce  sync.Once
	closeCalls atomic.Int32
}

func newBlockingReadCloser() *blockingReadCloser {
	return &blockingReadCloser{
		started: make(chan struct{}),
		closed:  make(chan struct{}),
	}
}

func (r *blockingReadCloser) Read([]byte) (int, error) {
	r.startOnce.Do(func() { close(r.started) })
	<-r.closed
	return 0, errors.New("reader closed")
}

func (r *blockingReadCloser) Close() error {
	r.closeCalls.Add(1)
	r.closeOnce.Do(func() { close(r.closed) })
	return nil
}

type countingReadCloser struct {
	io.Reader
	closeCalls atomic.Int32
}

func (r *countingReadCloser) Close() error {
	r.closeCalls.Add(1)
	return nil
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

	It("closes a blocked blob reader exactly once on cancellation", func() {
		reader := newBlockingReadCloser()
		adapter := NewAdapterWithClient(mockClient)

		mockClient.EXPECT().Get(gomock.Any(), artifactRef).Return(descriptor, nil)
		mockClient.EXPECT().
			GetLocalResource(gomock.Any(), artifactRef, resource.ToIdentity()).
			Return(readCloserBlob{reader: reader}, &resource, nil)

		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)

		go func() {
			_, err := adapter.GetArtifactManifestByReference(ctx, artifactRef)
			result <- err
		}()

		Eventually(reader.started).Should(BeClosed())
		cancel()

		var err error
		Eventually(result).Should(Receive(&err))
		Expect(errors.Is(err, context.Canceled)).To(BeTrue())
		Expect(reader.closeCalls.Load()).To(Equal(int32(1)))
	})

	It("closes a normally consumed reader exactly once", func() {
		reader := &countingReadCloser{
			Reader: bytes.NewReader(
				[]byte(`{"type":"test.konfidence.cloud","allowReuse":true}`),
			),
		}
		adapter := NewAdapterWithClient(mockClient)

		mockClient.EXPECT().Get(gomock.Any(), artifactRef).Return(descriptor, nil)
		mockClient.EXPECT().
			GetLocalResource(gomock.Any(), artifactRef, resource.ToIdentity()).
			Return(readCloserBlob{reader: reader}, &resource, nil)

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
