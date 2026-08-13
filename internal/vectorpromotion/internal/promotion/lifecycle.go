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

// Cleared reports whether every gate has passed: the approval gate is either
// granted or absent. Promotions without gates carry no Approved condition at
// all; a gate that does not exist leaves no record.
func Cleared(p *konfidence.VectorPromotion) bool {
	return IsApproved(p) || !p.Spec.RequireApproval
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

func IsSuperseded(p *konfidence.VectorPromotion) bool {
	cond := getSucceededCondition(p)
	return cond != nil && cond.Status == metav1.ConditionFalse &&
		cond.Reason == konfidence.ReasonPromotionSuperseded
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

// NewestCleared returns the highest-sequence promotion whose gates have all
// passed and that is not terminal. Returns nil if no promotion qualifies.
func NewestCleared(promotions []konfidence.VectorPromotion) *konfidence.VectorPromotion {
	var newest *konfidence.VectorPromotion
	for i := range promotions {
		p := &promotions[i]
		if !Cleared(p) || IsTerminal(p) {
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

// deriveApprovalState projects the gate view: Ready means every gate has
// passed (approval granted, or none required), Waiting means at least one
// gate is still open.
func deriveApprovalState(p *konfidence.VectorPromotion) konfidence.VectorPromotionState {
	if Cleared(p) {
		return konfidence.PromotionStateReady
	}
	return konfidence.PromotionStateWaiting
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
