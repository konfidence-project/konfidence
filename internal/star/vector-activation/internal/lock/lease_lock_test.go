package lock

import (
	"context"
	"errors"
	"time"

	. "github.com/konfidence-project/landscape-vector-activation-controller/internal/lock/mocks"
	. "github.com/konfidence-project/landscape-vector-activation-controller/test/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ = Describe("lease lock", func() {
	const (
		ResourceId   = "1234567890"
		ResourceType = "test-type"
		Namespace    = "default"
		stageName    = "dev"
		LockName     = "test-type-dev-lock"
	)

	var (
		mockCtrl           *gomock.Controller
		clientMock         *MockInterface
		coordinationV1Mock *MockCoordinationV1Interface
		leasesMock         *MockLeaseInterface
		lease              *coordinationv1.Lease
		now                time.Time
		controllerId       = "controller-1"
		holderIdentity     = controllerId + "-" + ResourceId
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		clientMock = NewMockInterface(mockCtrl)
		coordinationV1Mock = NewMockCoordinationV1Interface(mockCtrl)
		leasesMock = NewMockLeaseInterface(mockCtrl)
		now = time.Now()

		lease = &coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      LockName,
				Namespace: Namespace,
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity:       &holderIdentity,
				LeaseDurationSeconds: pointer(int32(DefaultLeaseTTL.Seconds())),
				AcquireTime:          &metav1.MicroTime{Time: now},
				RenewTime:            &metav1.MicroTime{Time: now},
			},
		}

		clientMock.EXPECT().CoordinationV1().Return(coordinationV1Mock).AnyTimes()
		coordinationV1Mock.EXPECT().Leases(Namespace).Return(leasesMock).AnyTimes()

	})

	Context("lease lock library", func() {
		It("acquires a lease and updates it", func() {

			leasesMock.EXPECT().Get(gomock.Any(), LockName, gomock.Any()).Return(lease, nil)
			leasesMock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
			acquired, err := AcquireResourceLease(context.Background(), clientMock, ResourceId, Namespace, controllerId, ResourceType, stageName, metav1.OwnerReference{})

			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeTrue())
		})

		It("creates a lease if not found", func() {
			leasesMock.EXPECT().Get(gomock.Any(), LockName, gomock.Any()).Return(nil, apierrors.NewNotFound(schema.GroupResource{Group: "coordination.k8s.io", Resource: "leases"}, LockName))
			leasesMock.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(&coordinationv1.Lease{}, nil)

			acquired, err := AcquireResourceLease(context.Background(), clientMock, ResourceId, Namespace, controllerId, ResourceType, stageName, metav1.OwnerReference{})

			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeTrue())

			// does not create another lease for a second resource
			leasesMock.EXPECT().Get(gomock.Any(), LockName, gomock.Any()).Return(lease, nil)
			acquired, err = AcquireResourceLease(context.Background(), clientMock, "another-resource", Namespace, controllerId, ResourceType, stageName, metav1.OwnerReference{})

			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeFalse())

			// creates lease for different stage
			leasesMock.EXPECT().Get(gomock.Any(), "test-type-test-stage-lock", gomock.Any()).Return(nil, apierrors.NewNotFound(schema.GroupResource{Group: "coordination.k8s.io", Resource: "leases"}, LockName))
			leasesMock.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(&coordinationv1.Lease{}, nil)
			acquired, err = AcquireResourceLease(context.Background(), clientMock, ResourceId, Namespace, controllerId, ResourceType, "test-stage", metav1.OwnerReference{})
			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeTrue())
		})

		It("does not acquire a lease if held by another controller", func() {
			leasesMock.EXPECT().Get(gomock.Any(), LockName, gomock.Any()).Return(lease, nil)

			acquired, err := AcquireResourceLease(context.Background(), clientMock, ResourceId, Namespace, "another-controller", ResourceType, stageName, metav1.OwnerReference{})

			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeFalse())
		})

		It("acquires a lease if held by another controller but expired", func() {
			otherController := "another-controller"
			lease = &coordinationv1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      LockName,
					Namespace: Namespace,
				},
				Spec: coordinationv1.LeaseSpec{
					HolderIdentity:       &otherController,
					LeaseDurationSeconds: pointer(int32(DefaultLeaseTTL.Seconds())),
					AcquireTime:          &metav1.MicroTime{Time: now},
					RenewTime:            &metav1.MicroTime{Time: now.Add(-2 * DefaultLeaseTTL)},
				},
			}

			leasesMock.EXPECT().Get(gomock.Any(), LockName, gomock.Any()).Return(lease, nil)
			leasesMock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

			acquired, err := AcquireResourceLease(context.Background(), clientMock, ResourceId, Namespace, controllerId, ResourceType, stageName, metav1.OwnerReference{})

			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeTrue())

		})

		It("releases a lease if held by this controller", func() {
			leasesMock.EXPECT().Get(gomock.Any(), LockName, gomock.Any()).Return(lease, nil)
			leasesMock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any())

			err := ReleaseResourceLease(context.Background(), clientMock, ResourceId, Namespace, controllerId, ResourceType, stageName)
			Expect(err).ToNot(HaveOccurred())
		})

		It("release does nothing on lease not found error and throws other errors", func() {
			leasesMock.EXPECT().Get(gomock.Any(), LockName, gomock.Any()).Return(nil, apierrors.NewNotFound(schema.GroupResource{Group: "coordination.k8s.io", Resource: "leases"}, LockName))

			err := ReleaseResourceLease(context.Background(), clientMock, ResourceId, Namespace, controllerId, ResourceType, stageName)

			Expect(err).ToNot(HaveOccurred())

			leasesMock.EXPECT().Get(gomock.Any(), LockName, gomock.Any()).Return(nil, errors.New("some other error"))
			err = ReleaseResourceLease(context.Background(), clientMock, ResourceId, Namespace, controllerId, ResourceType, stageName)
			Expect(err).To(HaveOccurred())
		})

	})
})
