package conditions

import (
	"github.com/konfidence-project/pkg/conditions/internal/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Get", func() {
	var (
		ctrl          *gomock.Controller
		mockGetter    *mocks.MockGetter
		conditionType ConditionType
		conditions    []*metav1.Condition
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockGetter = mocks.NewMockGetter(ctrl)
		conditionType = ConditionType("TestType")
		conditions = []*metav1.Condition{
			{
				Type:   string(conditionType),
				Status: metav1.ConditionTrue,
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should return the correct condition", func() {
		mockGetter.EXPECT().GetConditions().Return(conditions).Times(1)
		result := Get(mockGetter, conditionType)
		Expect(result).ToNot(BeNil())
		Expect(result.Type).To(Equal(string(conditionType)))
		Expect(result.Status).To(Equal(metav1.ConditionTrue))
	})
})

var _ = Describe("Has", func() {
	var (
		ctrl          *gomock.Controller
		mockGetter    *mocks.MockGetter
		conditionType ConditionType
		conditions    []*metav1.Condition
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockGetter = mocks.NewMockGetter(ctrl)
		conditionType = ConditionType("TestType")
		conditions = []*metav1.Condition{
			{
				Type:   string(conditionType),
				Status: metav1.ConditionTrue,
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should return true if the condition exists", func() {
		mockGetter.EXPECT().GetConditions().Return(conditions).Times(1)
		result := Has(mockGetter, conditionType)
		Expect(result).To(BeTrue())
	})

	It("should return false if the condition does not exist", func() {
		mockGetter.EXPECT().GetConditions().Return([]*metav1.Condition{}).Times(1)
		result := Has(mockGetter, conditionType)
		Expect(result).To(BeFalse())
	})
})

var _ = Describe("HasAny", func() {
	var (
		ctrl           *gomock.Controller
		mockGetter     *mocks.MockGetter
		conditionTypes []ConditionType
		conditions     []*metav1.Condition
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockGetter = mocks.NewMockGetter(ctrl)
		conditionTypes = []ConditionType{"TestType1", "TestType2"}
		conditions = []*metav1.Condition{
			{
				Type:   string(conditionTypes[0]),
				Status: metav1.ConditionTrue,
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should return true if any of the conditions exist", func() {
		mockGetter.EXPECT().GetConditions().Return(conditions).Times(1)
		result := HasAny(mockGetter, conditionTypes)
		Expect(result).To(BeTrue())
	})

	It("should return false if none of the conditions exist", func() {
		mockGetter.EXPECT().GetConditions().Return([]*metav1.Condition{}).MinTimes(1)
		result := HasAny(mockGetter, conditionTypes)
		Expect(result).To(BeFalse())
	})
})

var _ = Describe("HasAll", func() {
	var (
		ctrl           *gomock.Controller
		mockGetter     *mocks.MockGetter
		conditionTypes []ConditionType
		conditions     []*metav1.Condition
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockGetter = mocks.NewMockGetter(ctrl)
		conditionTypes = []ConditionType{"TestType1", "TestType2"}
		conditions = []*metav1.Condition{
			{
				Type:   string(conditionTypes[0]),
				Status: metav1.ConditionTrue,
			},
			{
				Type:   string(conditionTypes[1]),
				Status: metav1.ConditionTrue,
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should return true if all of the conditions exist", func() {
		mockGetter.EXPECT().GetConditions().Return(conditions).Times(len(conditionTypes))
		result := HasAll(mockGetter, conditionTypes)
		Expect(result).To(BeTrue())
	})

	It("should return false if not all of the conditions exist", func() {
		mockGetter.EXPECT().GetConditions().Return(conditions[:1]).Times(len(conditionTypes))
		result := HasAll(mockGetter, conditionTypes)
		Expect(result).To(BeFalse())
	})
})
