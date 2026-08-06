package promotion

import (
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getSucceededCondition(p *konfidence.VectorPromotion) *metav1.Condition {
	return meta.FindStatusCondition(p.Status.Conditions, konfidence.ConditionTypeSucceeded)
}

func IsApproved(p *konfidence.VectorPromotion) bool {
	return meta.IsStatusConditionTrue(p.Status.Conditions, konfidence.ConditionTypeApproved)
}

func IsInProgress(p *konfidence.VectorPromotion) bool {
	cond := getSucceededCondition(p)
	return cond != nil && cond.Reason == konfidence.ReasonPromotionRunning
}

// InProgressLongerThan reports whether the promotion has claimed InProgress
// for longer than the given duration.
func InProgressLongerThan(p *konfidence.VectorPromotion, d time.Duration) bool {
	if !IsInProgress(p) {
		return false
	}
	return time.Since(getSucceededCondition(p).LastTransitionTime.Time) > d
}

func IsSucceeded(p *konfidence.VectorPromotion) bool {
	cond := getSucceededCondition(p)
	return cond != nil && cond.Status == metav1.ConditionTrue
}

func IsTerminal(p *konfidence.VectorPromotion) bool {
	cond := getSucceededCondition(p)
	if cond == nil {
		return false
	}
	if cond.Status == metav1.ConditionTrue {
		return true
	}
	// Unknown conventionally means "not yet known" and is never terminal.
	if cond.Status != metav1.ConditionFalse {
		return false
	}
	return cond.Reason != konfidence.ReasonPromotionRunning &&
		cond.Reason != konfidence.ReasonPromotionTargetUnresolved
}

// DeriveState summarizes the promotion conditions into a display state.
// Conditions stay the source of truth; callers refresh `status.state` with
// this value whenever they write conditions.
func DeriveState(p *konfidence.VectorPromotion) konfidence.VectorPromotionState {
	cond := getSucceededCondition(p)
	if cond == nil {
		return deriveApprovalState(p)
	}
	if cond.Status == metav1.ConditionTrue {
		return konfidence.PromotionStateSucceeded
	}
	switch cond.Reason {
	case konfidence.ReasonPromotionRunning:
		return konfidence.PromotionStateInProgress
	case konfidence.ReasonPromotionTargetUnresolved:
		return konfidence.PromotionStateBlocked
	case konfidence.ReasonPromotionSuperseded:
		return konfidence.PromotionStateSuperseded
	}
	return konfidence.PromotionStateFailed
}

// NewestApproved returns the most recently created promotion that is approved
// and not terminal. Returns nil if no promotion qualifies.
func NewestApproved(promotions []konfidence.VectorPromotion) *konfidence.VectorPromotion {
	var newest *konfidence.VectorPromotion
	for i := range promotions {
		p := &promotions[i]
		if !IsApproved(p) || IsTerminal(p) {
			continue
		}
		if newest == nil || Newer(p, newest) {
			newest = p
		}
	}
	return newest
}

// Newer reports whether p is newer than than. The creator-assigned
// `spec.sequence` wins when both carry one; creation timestamps only have
// second resolution, and their ties are broken by name, which is
// deterministic across reconciles but otherwise arbitrary.
func Newer(p, than *konfidence.VectorPromotion) bool {
	if p.Spec.Sequence != than.Spec.Sequence {
		return p.Spec.Sequence > than.Spec.Sequence
	}
	if p.CreationTimestamp.Equal(&than.CreationTimestamp) {
		return p.Name > than.Name
	}
	return p.CreationTimestamp.After(than.CreationTimestamp.Time)
}

func deriveApprovalState(p *konfidence.VectorPromotion) konfidence.VectorPromotionState {
	if meta.IsStatusConditionTrue(p.Status.Conditions, konfidence.ConditionTypeApproved) {
		return konfidence.PromotionStateApproved
	}
	if p.Spec.RequireApproval {
		return konfidence.PromotionStateWaitingForApproval
	}
	return konfidence.PromotionStatePending
}

// TTLStatus returns whether the promotion should be deleted and the time remaining until expiry.
// Returns (false, 0) if no TTL is configured or promotion is not terminal.
func TTLStatus(p *konfidence.VectorPromotion) (shouldDelete bool, remaining time.Duration) {
	if p.Spec.TTLAfterFinished == nil || !IsTerminal(p) {
		return false, 0
	}
	expiresAt := getSucceededCondition(p).LastTransitionTime.Add(p.Spec.TTLAfterFinished.Duration)
	remaining = time.Until(expiresAt)
	return remaining <= 0, remaining
}
