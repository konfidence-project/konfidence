package activation

import (
	"context"
	"errors"

	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	. "github.com/konfidence-project/landscape-vector-activation-controller/test/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("status tests", func() {
	var (
		ctx              context.Context
		mockCtrl         *gomock.Controller
		clientMock       *MockClient
		statusWriterMock *MockSubResourceWriter
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockCtrl = gomock.NewController(GinkgoT())
		clientMock = NewMockClient(mockCtrl)
		statusWriterMock = NewMockSubResourceWriter(mockCtrl)
	})

	Context("InFinalStatusCondition", func() {
		It("returns false if no conditions", func() {
			va := &landscape.VectorActivation{}
			Expect(InFinalStatusCondition(va)).To(BeFalse())
		})

		It("returns true if ActivationSucceeded", func() {
			va := &landscape.VectorActivation{
				Status: landscape.VectorActivationStatus{
					Conditions: []metav1.Condition{
						{Type: landscape.ActivationSucceeded, Status: metav1.ConditionTrue},
					},
				},
			}
			Expect(InFinalStatusCondition(va)).To(BeTrue())
		})

		It("returns true if ActivationFailed", func() {
			va := &landscape.VectorActivation{
				Status: landscape.VectorActivationStatus{
					Conditions: []metav1.Condition{
						{Type: landscape.ActivationFailed, Status: metav1.ConditionTrue},
					},
				},
			}
			Expect(InFinalStatusCondition(va)).To(BeTrue())
		})

		It("returns true if ActivationSkipped", func() {
			va := &landscape.VectorActivation{
				Status: landscape.VectorActivationStatus{
					Conditions: []metav1.Condition{
						{Type: landscape.ActivationSkipped, Status: metav1.ConditionTrue},
					},
				},
			}
			Expect(InFinalStatusCondition(va)).To(BeTrue())
		})
	})

	Context("PatchVectorActivationStatus", func() {
		var (
			namespacedName types.NamespacedName
			condition      metav1.Condition
			va             *landscape.VectorActivation
		)

		BeforeEach(func() {
			namespacedName = types.NamespacedName{Name: "va", Namespace: "default"}
			condition = metav1.Condition{
				Type:    "Ready",
				Status:  metav1.ConditionTrue,
				Reason:  "Test",
				Message: "Testing",
			}
			va = &landscape.VectorActivation{}
		})

		It("returns error if Get fails", func() {
			clientMock.EXPECT().Get(ctx, namespacedName, gomock.Any()).Return(errors.New("not found"))
			err := PatchVectorActivationStatus(ctx, clientMock, namespacedName, condition)
			Expect(err).To(HaveOccurred())
		})

		It("returns nil if last condition matches", func() {
			va.Status.Conditions = []metav1.Condition{condition}
			clientMock.EXPECT().Get(ctx, namespacedName, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ types.NamespacedName, obj interface{}, _ ...interface{}) error {
					*obj.(*landscape.VectorActivation) = *va
					return nil
				})
			err := PatchVectorActivationStatus(ctx, clientMock, namespacedName, condition)
			Expect(err).ToNot(HaveOccurred())
		})

		It("patches status if condition is new", func() {
			clientMock.EXPECT().Get(ctx, namespacedName, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ types.NamespacedName, obj interface{}, _ ...interface{}) error {
					*obj.(*landscape.VectorActivation) = *va
					return nil
				})
			clientMock.EXPECT().Status().Return(statusWriterMock)
			statusWriterMock.EXPECT().Patch(ctx, gomock.Any(), gomock.Any()).Return(nil)
			err := PatchVectorActivationStatus(ctx, clientMock, namespacedName, condition)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if Patch fails", func() {
			clientMock.EXPECT().Get(ctx, namespacedName, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ types.NamespacedName, obj interface{}, _ ...interface{}) error {
					*obj.(*landscape.VectorActivation) = *va
					return nil
				})
			clientMock.EXPECT().Status().Return(statusWriterMock)
			statusWriterMock.EXPECT().Patch(ctx, gomock.Any(), gomock.Any()).Return(errors.New("patch error"))
			err := PatchVectorActivationStatus(ctx, clientMock, namespacedName, condition)
			Expect(err).To(HaveOccurred())
		})
	})
})
