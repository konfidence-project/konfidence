package conditions

import (
	metav1 "k8s.io/apimachinery/pkg/api/meta/v1"
)

// Get returns a pointer to the metav1.Condition of the specified type from the object.
// If the condition is not present, it returns nil.
func Get(obj Getter, t ConditionType) *metav1.Condition {
	conds := obj.GetConditions()
	if conds == nil {
		return nil
	}

	for i := range conds {
		if ConditionType(conds[i].Type) == t {
			return conds[i]
		}
	}

	return nil
}

// Has checks if the condition of the given type is present and its status is ConditionTrue.
func Has(obj Getter, t ConditionType) bool {
	cond := Get(obj, t)
	return cond != nil && cond.Status == metav1.ConditionTrue
}

// HasAny checks if any of the conditions in the given slice are present and have the status ConditionTrue.
func HasAny(obj Getter, types []ConditionType) bool {
	for _, t := range types {
		if Has(obj, t) {
			return true
		}
	}
	return false
}

// HasAll checks if all of the conditions in the given slice are present and have the status ConditionTrue.
func HasAll(obj Getter, types []ConditionType) bool {
	for _, t := range types {
		if !Has(obj, t) {
			return false
		}
	}
	return true
}

// IsTrue returns true if the condition of the given type is present and its status is ConditionTrue.
func IsTrue(obj Getter, t ConditionType) bool {
	cond := Get(obj, t)
	return cond != nil && cond.Status == metav1.ConditionTrue
}

// IsFalse returns true if the condition of the given type is present and its status is ConditionFalse.
func IsFalse(obj Getter, t ConditionType) bool {
	cond := Get(obj, t)
	return cond != nil && cond.Status == metav1.ConditionFalse
}

// IsUnknown returns true if the condition of the given type is present and its status is ConditionUnknown.
func IsUnknown(obj Getter, t ConditionType) bool {
	cond := Get(obj, t)
	return cond != nil && cond.Status == metav1.ConditionUnknown
}

// IsSet returns true if the condition of the given type is present and its status is any of True, False, or Unknown.
func IsSet(obj Getter, t ConditionType) bool {
	cond := Get(obj, t)
	return cond != nil &&
		(cond.Status == metav1.ConditionTrue ||
			cond.Status == metav1.ConditionFalse ||
			cond.Status == metav1.ConditionUnknown)
}

// IsNotSet returns true if the condition of the given type is not present, or its status is False or Unknown.
func IsNotSet(obj Getter, t ConditionType) bool {
	cond := Get(obj, t)
	return cond == nil || cond.Status == metav1.ConditionFalse || cond.Status == metav1.ConditionUnknown
}

// IsReady returns true if the condition of the given type is present,
// its status is True, and its type is ConditionReady.
func IsReady(obj Getter, t ConditionType) bool {
	cond := Get(obj, t)
	return cond != nil && cond.Status == metav1.ConditionTrue && cond.Type == string(ConditionReady)
}

// GetReason returns the Reason field of the condition of the given type, or an empty string if not present.
func GetReason(obj Getter, t ConditionType) string {
	if cond := Get(obj, t); cond != nil {
		return cond.Reason
	}

	return ""
}

// GetMessage returns the Message field of the condition of the given type, or an empty string if not present.
func GetMessage(obj Getter, t ConditionType) string {
	if cond := Get(obj, t); cond != nil {
		return cond.Message
	}

	return ""
}

// GetLastTransitionTime returns a pointer to the LastTransitionTime of the condition
// of the given type, or nil if not present.
func GetLastTransitionTime(obj Getter, t ConditionType) *metav1.Time {
	if cond := Get(obj, t); cond != nil {
		return &cond.LastTransitionTime
	}

	return nil
}

// GetObservedGeneration returns the ObservedGeneration of the condition of the given type, or 0 if not present.
func GetObservedGeneration(obj Getter, t ConditionType) int64 {
	if cond := Get(obj, t); cond != nil {
		return cond.ObservedGeneration
	}

	return 0
}
