// Package promotion holds the VectorPromotion lifecycle predicates. Following
// the Kubernetes API conventions, a promotion has no phase: each predicate
// reads observed facts — condition statuses and timestamps — and never a
// condition reason or the gates.
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

// GatesCleared reports whether the GatesSatisfied condition latched True.
func GatesCleared(p *konfidence.VectorPromotion) bool {
	return meta.IsStatusConditionTrue(p.Status.Conditions, konfidence.ConditionTypeGatesSatisfied)
}

// IsWaiting reports whether the active promotion still waits on its gates.
func IsWaiting(p *konfidence.VectorPromotion) bool {
	return !GatesCleared(p) && !IsTerminal(p)
}

// IsCleared reports whether the active promotion passed its gates and
// competes for execution.
func IsCleared(p *konfidence.VectorPromotion) bool {
	return GatesCleared(p) && !IsTerminal(p)
}

// IsRunning reports whether the promotion is currently executing. At most
// one promotion per config is running at a time.
func IsRunning(p *konfidence.VectorPromotion) bool {
	return p.Status.StartTime != nil && !IsTerminal(p)
}

// NewestCleared returns the highest-sequence promotion that passed its gates
// and is not finished. Returns nil if no promotion qualifies.
func NewestCleared(promotions []konfidence.VectorPromotion) *konfidence.VectorPromotion {
	var newest *konfidence.VectorPromotion
	for i := range promotions {
		p := &promotions[i]
		if !IsCleared(p) {
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
