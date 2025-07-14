package conditions

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

const (

	// messageMaxLength is the maximum length of a condition message.
	messageMaxLength = 32768
)

// Set sets the given condition.
//
// If a condition already exists, the LastTransitionTime is updated only if a change is detected in any of the
// following fields: Status, Reason, and Message. The ObservedGeneration is always updated.
func Set(obj Setter, condition *metav1.Condition) {
	if obj == nil || condition == nil {
		return
	}

	// Always set the observed generation on the condition.
	condition.ObservedGeneration = obj.GetGeneration()

	// Trim the message to the maximum accepted length.
	condition.Message = trimConditionMessage(condition.Message)

	conditions := obj.GetConditions()
	exists := false
	for i := range conditions {
		existingCondition := conditions[i]
		if existingCondition.Type == condition.Type {
			exists = true
			if !isSameState(existingCondition, condition) {
				condition.LastTransitionTime = metav1.NewTime(time.Now().UTC().Truncate(time.Second))
				conditions[i] = condition
				break
			}
			condition.LastTransitionTime = existingCondition.LastTransitionTime
			if existingCondition.ObservedGeneration != condition.ObservedGeneration {
				conditions[i] = condition
			}
			break
		}
	}

	if !exists {
		if condition.LastTransitionTime.IsZero() {
			condition.LastTransitionTime = metav1.NewTime(time.Now().UTC().Truncate(time.Second))
		}
		conditions = append(conditions, condition)
	}

	obj.SetConditions(conditions)
}

// Delete removes the condition of the specified type from the object.
// If the object is nil, the function returns without making changes.
func Delete(to Setter, t string) {
	if to == nil {
		return
	}

	conditions := to.GetConditions()
	newConditions := make([]*metav1.Condition, 0, len(conditions))
	for _, condition := range conditions {
		if condition.Type != t {
			newConditions = append(newConditions, condition)
		}
	}
	to.SetConditions(newConditions)
}

// MarkTrue sets the status of the given condition to ConditionTrue and updates it on the object.
// If the condition is nil, the function returns without making changes.
func MarkTrue(obj Setter, cond *metav1.Condition) {
	if cond == nil {
		return
	}

	cond.Status = metav1.ConditionTrue
	Set(obj, cond)
}

// MarkFalse sets the status of the given condition to ConditionFalse and updates it on the object.
// If the condition is nil, the function returns without making changes.
func MarkFalse(obj Setter, cond *metav1.Condition) {
	if cond == nil {
		return
	}

	cond.Status = metav1.ConditionFalse
	Set(obj, cond)
}

// MarkUnknown sets the status of the given condition to ConditionUnknown and updates it on the object.
// If the condition is nil, the function returns without making changes.
func MarkUnknown(obj Setter, cond *metav1.Condition) {
	if cond == nil {
		return
	}

	cond.Status = metav1.ConditionUnknown
	Set(obj, cond)
}

// trimConditionMessage trims the message to the maximum allowed length for a condition message.
// If the message exceeds the maximum length, it is truncated.
func trimConditionMessage(msg string) string {
	if len(msg) > messageMaxLength {
		return msg[:messageMaxLength]
	}
	return msg
}

// isSameState checks if the status, reason, and message fields are the same between two conditions.
// Returns true if all three fields match, otherwise false.
func isSameState(a, b *metav1.Condition) bool {
	return a.Status == b.Status && a.Reason == b.Reason && a.Message == b.Message
}
