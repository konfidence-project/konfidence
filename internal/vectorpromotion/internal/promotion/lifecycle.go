// Package promotion holds the VectorPromotion lifecycle predicates. Following
// the Kubernetes API conventions, a promotion has no phase: each predicate
// reads observed facts — condition statuses, timestamps, and the operator
// override — and never a condition reason or the gates.
package promotion

import (
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
)

// IsTerminal reports whether the promotion's Succeeded condition is True or
// False. A finished promotion never changes again.
func IsTerminal(p *konfidence.VectorPromotion) bool {
	return meta.IsStatusConditionTrue(p.Status.Conditions, konfidence.ConditionTypeSucceeded) ||
		meta.IsStatusConditionFalse(p.Status.Conditions, konfidence.ConditionTypeSucceeded)
}

// IsSucceeded reports whether the promotion finished successfully.
func IsSucceeded(p *konfidence.VectorPromotion) bool {
	return meta.IsStatusConditionTrue(p.Status.Conditions, konfidence.ConditionTypeSucceeded)
}

// AllGatesSatisfied reports whether the GatesSatisfied condition is True.
func AllGatesSatisfied(p *konfidence.VectorPromotion) bool {
	return meta.IsStatusConditionTrue(p.Status.Conditions, konfidence.ConditionTypeGatesSatisfied)
}

// IsWaiting reports whether the promotion is not yet ready to execute and has
// not finished.
func IsWaiting(p *konfidence.VectorPromotion) bool {
	return !IsReadyForExecution(p) && !IsTerminal(p)
}

// HasOverrideSet reports whether the operator recorded an override of the given
// type on the promotion.
func HasOverrideSet(p *konfidence.VectorPromotion, overrideType konfidence.PromotionOverrideType) bool {
	return p.Status.Override != nil && p.Status.Override.Type == overrideType
}

// IsReadyForExecution reports whether the active promotion may execute now: its
// gates are satisfied or an operator bypassed them, and it has not finished.
func IsReadyForExecution(p *konfidence.VectorPromotion) bool {
	return (AllGatesSatisfied(p) || HasOverrideSet(p, konfidence.PromotionOverrideTypeBypassGates)) && !IsTerminal(p)
}

// IsRunning reports whether the promotion is currently executing. At most
// one promotion per config is running at a time.
func IsRunning(p *konfidence.VectorPromotion) bool {
	return p.Status.StartTime != nil && !IsTerminal(p)
}

// NewestCleared returns the highest-sequence promotion that is ready for
// execution and is not finished. Returns nil if no promotion qualifies.
func NewestCleared(promotions []konfidence.VectorPromotion) *konfidence.VectorPromotion {
	var newest *konfidence.VectorPromotion
	for i := range promotions {
		p := &promotions[i]
		if !IsReadyForExecution(p) {
			continue
		}
		if newest == nil || Newer(p, newest) {
			newest = p
		}
	}
	return newest
}

// Newest returns the newest promotion by Newer ordering, regardless of state.
// Returns nil for an empty slice.
func Newest(promotions []konfidence.VectorPromotion) *konfidence.VectorPromotion {
	var newest *konfidence.VectorPromotion
	for i := range promotions {
		if newest == nil || Newer(&promotions[i], newest) {
			newest = &promotions[i]
		}
	}
	return newest
}

// Newer reports whether p has a higher creator-assigned `spec.sequence` than
// the vector promotion passed in the `than` parameter. Equal sequences
// (possible only between hand-crafted promotions) compare as not newer, keeping
// the incumbent stable; timestamps are never consulted because their second
// resolution invites ties.
func Newer(p, than *konfidence.VectorPromotion) bool {
	return p.Spec.Sequence > than.Spec.Sequence
}

// TTLStatus returns whether the promotion should be deleted and the time
// remaining until expiry, measured from `status.completionTime` (or the
// Succeeded condition's transition time if it is missing). Returns (false, 0)
// if no TTL is configured or the promotion is not finished.
func TTLStatus(p *konfidence.VectorPromotion) (shouldDelete bool, remaining time.Duration) {
	if p.Spec.TTLAfterFinished == nil || !IsTerminal(p) {
		return false, 0
	}
	finishedAt := meta.FindStatusCondition(p.Status.Conditions, konfidence.ConditionTypeSucceeded).LastTransitionTime
	if p.Status.CompletionTime != nil {
		finishedAt = *p.Status.CompletionTime
	}
	remaining = time.Until(finishedAt.Add(p.Spec.TTLAfterFinished.Duration))
	return remaining <= 0, remaining
}
