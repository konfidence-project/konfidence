package conditions

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/pkg/conditions/internal/mocks"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Set", func() {
	var (
		ctrl       *gomock.Controller
		mockSetter *mocks.MockSetter
		condition  *metav1.Condition
		conditions []*metav1.Condition
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockSetter = mocks.NewMockSetter(ctrl)
		condition = &metav1.Condition{
			Type:               "TestType",
			Status:             metav1.ConditionTrue,
			Reason:             "TestReason",
			Message:            "TestMessage",
			ObservedGeneration: 1,
		}
		conditions = []*metav1.Condition{
			{
				Type:               "TestType",
				Status:             metav1.ConditionFalse,
				Reason:             "OldReason",
				Message:            "OldMessage",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.NewTime(time.Now().Add(-time.Hour)),
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should update LastTransitionTime and ObservedGeneration", func() {
		mockSetter.EXPECT().GetConditions().Return(conditions).Times(1)
		mockSetter.EXPECT().GetGeneration().Return(int64(2)).Times(1)
		mockSetter.EXPECT().SetConditions(gomock.Any()).Times(1)

		Set(mockSetter, condition)

		Expect(condition.LastTransitionTime.IsZero()).To(BeFalse())
		Expect(condition.ObservedGeneration).To(Equal(int64(2)))
	})

	It("should trim the message to the maximum length", func() {
		longMsg := make([]byte, messageMaxLength+10)
		for i := range longMsg {
			longMsg[i] = 'a'
		}
		condition.Message = string(longMsg)
		mockSetter.EXPECT().GetConditions().Return(conditions).Times(1)
		mockSetter.EXPECT().GetGeneration().Return(int64(2)).Times(1)
		mockSetter.EXPECT().SetConditions(gomock.Any()).Times(1)

		Set(mockSetter, condition)
		Expect(len(condition.Message)).To(BeNumerically("<=", messageMaxLength))
	})
})

var _ = Describe("Delete", func() {
	var (
		ctrl       *gomock.Controller
		mockSetter *mocks.MockSetter
		conditions []*metav1.Condition
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockSetter = mocks.NewMockSetter(ctrl)
		conditions = []*metav1.Condition{
			{
				Type:   "TestType",
				Status: metav1.ConditionTrue,
			},
			{
				Type:   "OtherType",
				Status: metav1.ConditionFalse,
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should delete the condition of the given type", func() {
		mockSetter.EXPECT().GetConditions().Return(conditions).Times(1)
		mockSetter.EXPECT().SetConditions(gomock.Any()).Times(1)

		Delete(mockSetter, "TestType")
	})
})

func setupMockSetter(ctrl *gomock.Controller) (*mocks.MockSetter, *metav1.Condition) {
	mockSetter := mocks.NewMockSetter(ctrl)
	condition := &metav1.Condition{
		Type: "TestType",
	}
	return mockSetter, condition
}

var _ = Describe("Condition Markers", func() {
	var (
		ctrl       *gomock.Controller
		mockSetter *mocks.MockSetter
		condition  *metav1.Condition
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockSetter, condition = setupMockSetter(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should mark the condition as true", func() {
		mockSetter.EXPECT().GetConditions().Return(nil).Times(1)
		mockSetter.EXPECT().GetGeneration().Return(int64(1)).Times(1)
		mockSetter.EXPECT().SetConditions(gomock.Any()).Times(1)

		MarkTrue(mockSetter, condition)

		Expect(condition.Status).To(Equal(metav1.ConditionTrue))
	})

	It("should mark the condition as false", func() {
		mockSetter.EXPECT().GetConditions().Return(nil).Times(1)
		mockSetter.EXPECT().GetGeneration().Return(int64(1)).Times(1)
		mockSetter.EXPECT().SetConditions(gomock.Any()).Times(1)

		MarkFalse(mockSetter, condition)

		Expect(condition.Status).To(Equal(metav1.ConditionFalse))
	})

	It("should mark the condition as unknown", func() {
		mockSetter.EXPECT().GetConditions().Return(nil).Times(1)
		mockSetter.EXPECT().GetGeneration().Return(int64(1)).Times(1)
		mockSetter.EXPECT().SetConditions(gomock.Any()).Times(1)

		MarkUnknown(mockSetter, condition)

		Expect(condition.Status).To(Equal(metav1.ConditionUnknown))
	})
})
