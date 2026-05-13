package conditions

import (
	metav1 "k8s.io/apimachinery/pkg/api/meta/v1"
)

var _ = Describe("NewCondition", func() {
	It("should create a new condition with correct fields", func() {
		cond := NewCondition("TestType", metav1.ConditionTrue, "TestReason", "TestMessage")
		Expect(cond.Type).To(Equal("TestType"))
		Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		Expect(cond.Reason).To(Equal("TestReason"))
		Expect(cond.Message).To(Equal("TestMessage"))
		Expect(cond.LastTransitionTime.IsZero()).To(BeFalse())
	})
})

var _ = Describe("NewTrueCondition", func() {
	It("should create a new true condition", func() {
		cond := NewTrueCondition("TestType", "TestReason", "TestMessage")
		Expect(cond.Type).To(Equal("TestType"))
		Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		Expect(cond.Reason).To(Equal("TestReason"))
		Expect(cond.Message).To(Equal("TestMessage"))
		Expect(cond.LastTransitionTime.IsZero()).To(BeFalse())
	})
})

var _ = Describe("NewFalseCondition", func() {
	It("should create a new false condition", func() {
		cond := NewFalseCondition("TestType", "TestReason", "TestMessage")
		Expect(cond.Type).To(Equal("TestType"))
		Expect(cond.Status).To(Equal(metav1.ConditionFalse))
		Expect(cond.Reason).To(Equal("TestReason"))
		Expect(cond.Message).To(Equal("TestMessage"))
		Expect(cond.LastTransitionTime.IsZero()).To(BeFalse())
	})
})

var _ = Describe("NewUnknownCondition", func() {
	It("should create a new unknown condition", func() {
		cond := NewUnknownCondition("TestType", "TestReason", "TestMessage")
		Expect(cond.Type).To(Equal("TestType"))
		Expect(cond.Status).To(Equal(metav1.ConditionUnknown))
		Expect(cond.Reason).To(Equal("TestReason"))
		Expect(cond.Message).To(Equal("TestMessage"))
		Expect(cond.LastTransitionTime.IsZero()).To(BeFalse())
	})
})
