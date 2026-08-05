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

func IsPending(p *konfidence.VectorPromotion) bool {
	return getSucceededCondition(p) == nil
}

func IsApproved(p *konfidence.VectorPromotion) bool {
	return meta.IsStatusConditionTrue(p.Status.Conditions, konfidence.ConditionTypeApproved)
}

func IsInProgress(p *konfidence.VectorPromotion) bool {
	cond := getSucceededCondition(p)
	return cond != nil && cond.Reason == konfidence.ReasonPromotionRunning
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
	switch cond.Status {
	case metav1.ConditionTrue:
		return true
	case metav1.ConditionUnknown:
		return true
	case metav1.ConditionFalse:
		return cond.Reason != konfidence.ReasonPromotionRunning
	}
	return false
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

// Newer reports whether p was created after than. Creation timestamps have
// second resolution; ties are broken by name, which is deterministic across
// reconciles but otherwise arbitrary.
func Newer(p, than *konfidence.VectorPromotion) bool {
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
	if p.Spec.TTLAfterFinished == nil {
		return false, 0
	}
	cond := getSucceededCondition(p)
	if cond == nil {
		return false, 0
	}
	expiresAt := cond.LastTransitionTime.Add(p.Spec.TTLAfterFinished.Duration)
	remaining = time.Until(expiresAt)
	return remaining <= 0, remaining
}
