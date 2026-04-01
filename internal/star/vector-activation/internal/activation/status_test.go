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
						{
							Type:               landscape.ActivationSucceeded,
							Status:             metav1.ConditionTrue,
							Reason:             landscape.ActivationSucceeded,
							Message:            "",
							ObservedGeneration: 1,
							LastTransitionTime: metav1.Now(),
						},
					},
				},
			}
			Expect(InFinalStatusCondition(va)).To(BeTrue())
		})

		It("returns true if ActivationFailed", func() {
			va := &landscape.VectorActivation{
				Status: landscape.VectorActivationStatus{
					Conditions: []metav1.Condition{
						{
							Type:               landscape.ActivationFailed,
							Status:             metav1.ConditionTrue,
							Reason:             landscape.ActivationFailed,
							Message:            "",
							ObservedGeneration: 1,
							LastTransitionTime: metav1.Now(),
						},
					},
				},
			}
			Expect(InFinalStatusCondition(va)).To(BeTrue())
		})

		It("returns true if ActivationSkipped", func() {
			vectorActivation := &landscape.VectorActivation{
				Status: landscape.VectorActivationStatus{
					Conditions: []metav1.Condition{
						{
							Type:               landscape.ActivationSkipped,
							Status:             metav1.ConditionTrue,
							Reason:             landscape.ActivationSkipped,
							Message:            "",
							ObservedGeneration: 1,
							LastTransitionTime: metav1.Now(),
						},
					},
				},
			}
			Expect(InFinalStatusCondition(vectorActivation)).To(BeTrue())
		})
	})

	Context("UpdateVectorActivationStatus", func() {
		var (
			condition        metav1.Condition
			vectorActivation *landscape.VectorActivation
		)

		BeforeEach(func() {
			condition = metav1.Condition{
				Type:               "Ready",
				Status:             metav1.ConditionTrue,
				Reason:             "Test",
				Message:            "Testing",
				ObservedGeneration: 1,
				LastTransitionTime: metav1.Now(),
			}
			vectorActivation = &landscape.VectorActivation{}
		})

		It("updates status", func() {
			clientMock.EXPECT().Status().Return(statusWriterMock)
			statusWriterMock.EXPECT().Update(ctx, gomock.Any()).Return(nil)
			err := UpdateVectorActivationStatus(ctx, clientMock, vectorActivation, condition)
			Expect(err).ToNot(HaveOccurred())

			newCondition := metav1.Condition{
				Type:               "AnotherType",
				Status:             metav1.ConditionTrue,
				Reason:             "AnotherTest",
				Message:            "Testing",
				ObservedGeneration: 1,
				LastTransitionTime: metav1.Now(),
			}
			clientMock.EXPECT().Status().Return(statusWriterMock)
			statusWriterMock.EXPECT().Update(ctx, gomock.Any()).Return(nil)
			err = UpdateVectorActivationStatus(ctx, clientMock, vectorActivation, newCondition)
			Expect(err).ToNot(HaveOccurred())
			Expect(vectorActivation.Status.Conditions).To(HaveLen(1))
			Expect(vectorActivation.Status.Conditions[0].Type).To(Equal(newCondition.Type))
		})

		It("returns error if update fails", func() {
			clientMock.EXPECT().Status().Return(statusWriterMock)
			statusWriterMock.EXPECT().Update(ctx, gomock.Any()).Return(errors.New("update error"))
			err := UpdateVectorActivationStatus(ctx, clientMock, vectorActivation, condition)
			Expect(err).To(HaveOccurred())
		})
	})
})
