package conditions

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewCondition creates a new condition with the specified type, status, reason, and message.
func NewCondition(t ConditionType, status metav1.ConditionStatus, reason, message string) *metav1.Condition {
	return &metav1.Condition{
		Type:               string(t),
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// NewTrueCondition creates a new condition with the specified type, status set to ConditionTrue, reason, and message.
func NewTrueCondition(t ConditionType, reason, message string) *metav1.Condition {
	return NewCondition(t, metav1.ConditionTrue, reason, message)
}

// NewFalseCondition creates a new condition with the specified type, status set to ConditionFalse, reason, and message.
func NewFalseCondition(t ConditionType, reason, message string) *metav1.Condition {
	return NewCondition(t, metav1.ConditionFalse, reason, message)
}

// NewUnknownCondition creates a new condition with the specified type, status set to ConditionUnknown, reason, and message.
func NewUnknownCondition(t ConditionType, reason, message string) *metav1.Condition {
	return NewCondition(t, metav1.ConditionUnknown, reason, message)
}
