package lock

import (
	"context"
	"errors"
	"time"

	. "github.com/konfidence-project/landscape-vector-activation-controller/test/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		lease          *coordinationv1.Lease
		now            time.Time
		controllerId   = "controller-1"
		holderIdentity = controllerId + "-" + ResourceId
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockCtrl = gomock.NewController(GinkgoT())
		clientMock = NewMockClient(mockCtrl)
		now = time.Now()

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
			},
		}
	})

	Context("lease lock library", func() {
		It("acquires a lease and updates it", func() {

			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&coordinationv1.Lease{})).DoAndReturn(func(_ context.Context, namespacedName interface{}, obj interface{}, _ ...interface{}) error {
				l := obj.(*coordinationv1.Lease)
				*l = *lease
				return nil
			})

			clientMock.EXPECT().Update(ctx, gomock.Any())
			acquired, err := AcquireResourceLease(ctx, clientMock, ResourceId, Namespace, controllerId, ResourceType, stageName, metav1.OwnerReference{})

			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeTrue())
		})

		It("creates a lease if not found", func() {
			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&coordinationv1.Lease{})).Return(apierrors.NewNotFound(schema.GroupResource{}, LeaseName))
			clientMock.EXPECT().Create(ctx, gomock.Any()).Return(nil)

			acquired, err := AcquireResourceLease(ctx, clientMock, ResourceId, Namespace, controllerId, ResourceType, stageName, metav1.OwnerReference{})

			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeTrue())

			// does not create another lease for a second resource
			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&coordinationv1.Lease{})).DoAndReturn(func(_ context.Context, namespacedName interface{}, obj interface{}, _ ...interface{}) error {
				l := obj.(*coordinationv1.Lease)
				*l = *lease
				return nil
			})
			acquired, err = AcquireResourceLease(ctx, clientMock, "another-resource", Namespace, controllerId, ResourceType, stageName, metav1.OwnerReference{})

			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeFalse())

			// creates lease for different stage
			clientMock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, LeaseName))
			clientMock.EXPECT().Create(ctx, gomock.Any()).Return(nil)
			acquired, err = AcquireResourceLease(ctx, clientMock, ResourceId, Namespace, controllerId, ResourceType, "test-stage", metav1.OwnerReference{})
			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeTrue())
		})
		//
		It("does not acquire a lease if held by another controller", func() {
			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&coordinationv1.Lease{})).DoAndReturn(func(_ context.Context, namespacedName interface{}, obj interface{}, _ ...interface{}) error {
				l := obj.(*coordinationv1.Lease)
				*l = *lease
				return nil
			})

			acquired, err := AcquireResourceLease(context.Background(), clientMock, ResourceId, Namespace, "another-controller", ResourceType, stageName, metav1.OwnerReference{})

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
				},
			}

			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&coordinationv1.Lease{})).DoAndReturn(func(_ context.Context, namespacedName interface{}, obj interface{}, _ ...interface{}) error {
				l := obj.(*coordinationv1.Lease)
				*l = *lease
				return nil
			})
			clientMock.EXPECT().Update(ctx, gomock.Any())

			acquired, err := AcquireResourceLease(context.Background(), clientMock, ResourceId, Namespace, controllerId, ResourceType, stageName, metav1.OwnerReference{})

			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeTrue())

		})

		It("releases a lease if held by this controller", func() {
			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&coordinationv1.Lease{})).DoAndReturn(func(_ context.Context, namespacedName interface{}, obj interface{}, _ ...interface{}) error {
				l := obj.(*coordinationv1.Lease)
				*l = *lease
				return nil
			})
			clientMock.EXPECT().Update(ctx, gomock.Any())

			err := ReleaseResourceLease(context.Background(), clientMock, ResourceId, Namespace, controllerId, ResourceType, stageName)
			Expect(err).ToNot(HaveOccurred())
		})

		It("release does nothing on lease not found error and throws other errors", func() {
			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&coordinationv1.Lease{})).Return(apierrors.NewNotFound(schema.GroupResource{}, LeaseName))

			err := ReleaseResourceLease(context.Background(), clientMock, ResourceId, Namespace, controllerId, ResourceType, stageName)

			Expect(err).ToNot(HaveOccurred())

			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&coordinationv1.Lease{})).Return(errors.New("some other error"))
			err = ReleaseResourceLease(context.Background(), clientMock, ResourceId, Namespace, controllerId, ResourceType, stageName)
			Expect(err).To(HaveOccurred())
		})

	})
})
