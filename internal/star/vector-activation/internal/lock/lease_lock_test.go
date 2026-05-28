package lock

import (
	"context"
	"errors"
	"time"

	landscape "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	. "github.com/konfidence-project/konfidence/internal/star/vector-activation/test/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ = Describe("lease lock", func() {
	const (
		ResourceId   = "1234567890"
		ResourceType = "test-type"
		Namespace    = "default"
		stageName    = "dev"
		LeaseName    = "test-type-dev-lock"
	)

	var (
		ctx            context.Context
		mockCtrl       *gomock.Controller
		clientMock     *MockClient
		stage          *landscape.Stage
		lease          *coordinationv1.Lease
		now            time.Time
		controllerId   = "controller-1"
		holderIdentity = controllerId + "-" + ResourceId
		scheme         *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockCtrl = gomock.NewController(GinkgoT())
		clientMock = NewMockClient(mockCtrl)

		scheme = runtime.NewScheme()
		_ = landscape.AddToScheme(scheme)

		now = time.Now()

		stage = &landscape.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      stageName,
				Namespace: Namespace,
			},
		}
		lease = &coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      LeaseName,
				Namespace: Namespace,
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity:       &holderIdentity,
				LeaseDurationSeconds: pointer(int32(DefaultLeaseTTL.Seconds())),
				AcquireTime:          &metav1.MicroTime{Time: now},
				RenewTime:            &metav1.MicroTime{Time: now},
				LeaseTransitions:     pointer(0),
			},
		}
	})

	Context("lease lock library", func() {
		It("acquires a lease and updates it", func() {

			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&coordinationv1.Lease{})).
				DoAndReturn(func(_ context.Context, namespacedName interface{}, obj interface{}, _ ...interface{}) error {
					l := obj.(*coordinationv1.Lease)
					*l = *lease
					return nil
				})

			clientMock.EXPECT().Update(ctx, gomock.Any())
			acquired, err := AcquireResourceLease(ctx, clientMock, ResourceId, Namespace, controllerId, ResourceType, stage)

			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeTrue())
		})

		It("creates a lease if not found", func() {
			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&coordinationv1.Lease{})).
				Return(apierrors.NewNotFound(schema.GroupResource{}, LeaseName))
			clientMock.EXPECT().Scheme().Return(scheme)
			clientMock.EXPECT().Create(ctx, gomock.Any()).Return(nil)

			acquired, err := AcquireResourceLease(ctx, clientMock, ResourceId, Namespace, controllerId, ResourceType, stage)

			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeTrue())

			// does not create another lease for a second resource
			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&coordinationv1.Lease{})).
				DoAndReturn(func(_ context.Context, namespacedName interface{}, obj interface{}, _ ...interface{}) error {
					l := obj.(*coordinationv1.Lease)
					*l = *lease
					return nil
				})
			acquired, err = AcquireResourceLease(ctx, clientMock, "another-resource", Namespace, controllerId, ResourceType, stage)

			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeFalse())

			// creates lease for different stage
			newStage := &landscape.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "different-stage",
					Namespace: Namespace,
				},
			}
			clientMock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(apierrors.NewNotFound(schema.GroupResource{}, LeaseName))
			clientMock.EXPECT().Scheme().Return(scheme)
			clientMock.EXPECT().Create(ctx, gomock.Any()).Return(nil)
			acquired, err = AcquireResourceLease(ctx, clientMock, ResourceId, Namespace, controllerId, ResourceType, newStage)
			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeTrue())
		})
		//
		It("does not acquire a lease if held by another controller", func() {
			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&coordinationv1.Lease{})).
				DoAndReturn(func(_ context.Context, namespacedName interface{}, obj interface{}, _ ...interface{}) error {
					l := obj.(*coordinationv1.Lease)
					*l = *lease
					return nil
				})

			acquired, err := AcquireResourceLease(context.Background(), clientMock, ResourceId, Namespace, "another-controller", ResourceType, stage)

			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeFalse())
		})
		//
		It("acquires a lease if held by another controller but expired", func() {
			otherController := "another-controller"
			lease = &coordinationv1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      LeaseName,
					Namespace: Namespace,
				},
				Spec: coordinationv1.LeaseSpec{
					HolderIdentity:       &otherController,
					LeaseDurationSeconds: pointer(int32(DefaultLeaseTTL.Seconds())),
					AcquireTime:          &metav1.MicroTime{Time: now},
					RenewTime:            &metav1.MicroTime{Time: now.Add(-2 * DefaultLeaseTTL)},
					LeaseTransitions:     pointer(0),
				},
			}

			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&coordinationv1.Lease{})).
				DoAndReturn(func(_ context.Context, namespacedName interface{}, obj interface{}, _ ...interface{}) error {
					l := obj.(*coordinationv1.Lease)
					*l = *lease
					return nil
				})
			clientMock.EXPECT().Update(ctx, gomock.Any())

			acquired, err := AcquireResourceLease(
				context.Background(), clientMock, ResourceId, Namespace, controllerId, ResourceType, stage,
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeTrue())

		})

		It("releases a lease if held by this controller", func() {
			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&coordinationv1.Lease{})).
				DoAndReturn(func(_ context.Context, namespacedName interface{}, obj interface{}, _ ...interface{}) error {
					l := obj.(*coordinationv1.Lease)
					*l = *lease
					return nil
				})
			clientMock.EXPECT().Update(ctx, gomock.Any())

			err := ReleaseResourceLease(context.Background(), clientMock, ResourceId, Namespace, controllerId, ResourceType, stage)
			Expect(err).ToNot(HaveOccurred())
		})

		It("release does nothing on lease not found error and throws other errors", func() {
			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&coordinationv1.Lease{})).
				Return(apierrors.NewNotFound(schema.GroupResource{}, LeaseName))

			err := ReleaseResourceLease(
				context.Background(), clientMock, ResourceId, Namespace, controllerId, ResourceType, stage,
			)

			Expect(err).ToNot(HaveOccurred())

			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&coordinationv1.Lease{})).
				Return(errors.New("some other error"))
			err = ReleaseResourceLease(context.Background(), clientMock, ResourceId, Namespace, controllerId, ResourceType, stage)
			Expect(err).To(HaveOccurred())
		})

	})
})
