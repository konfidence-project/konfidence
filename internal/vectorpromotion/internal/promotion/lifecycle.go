// Package promotion holds the VectorPromotion lifecycle. The phase of a
// promotion is the reason of its Succeeded condition. One table pairs every
// reason with its condition status and its VectorPromotionState: writers look
// up the status with ConditionStatusFor, readers classify a promotion with
// DeriveState. Nothing here reads the gates.
package promotion

import (
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type phase struct {
	status metav1.ConditionStatus
	state  konfidence.VectorPromotionState
}

// phases is the single place a Succeeded reason is paired with its condition
// status and its state.
var phases = map[string]phase{
	konfidence.ReasonGatesPending:                     {metav1.ConditionUnknown, konfidence.PromotionStateWaiting},
	konfidence.ReasonGateError:                        {metav1.ConditionUnknown, konfidence.PromotionStateWaiting},
	konfidence.ReasonPromotionReady:                   {metav1.ConditionUnknown, konfidence.PromotionStateReady},
	konfidence.ReasonPromotionRunning:                 {metav1.ConditionUnknown, konfidence.PromotionStateInProgress},
	konfidence.ReasonPromotionTargetUnresolved:        {metav1.ConditionUnknown, konfidence.PromotionStateBlocked},
	konfidence.ReasonPromotionSucceeded:               {metav1.ConditionTrue, konfidence.PromotionStateSucceeded},
	konfidence.ReasonPromotionSuperseded:              {metav1.ConditionFalse, konfidence.PromotionStateSuperseded},
	konfidence.ReasonPromotionOutstandingLimitReached: {metav1.ConditionFalse, konfidence.PromotionStateSuperseded},
	konfidence.ReasonPromotionFailed:                  {metav1.ConditionFalse, konfidence.PromotionStateFailed},
}

func getSucceededCondition(p *konfidence.VectorPromotion) *metav1.Condition {
	return meta.FindStatusCondition(p.Status.Conditions, konfidence.ConditionTypeSucceeded)
}

// ConditionStatusFor returns the Succeeded condition status that belongs to
// the given reason, and false for a reason that is not a known phase.
func ConditionStatusFor(reason string) (metav1.ConditionStatus, bool) {
	ph, ok := phases[reason]
	return ph.status, ok
}

// DeriveState classifies the promotion from the reason of its Succeeded
// condition: Waiting before the first reconcile, the table state for a known
// reason, and "" for an unrecognized reason (non-terminal, never executed).
func DeriveState(p *konfidence.VectorPromotion) konfidence.VectorPromotionState {
	cond := getSucceededCondition(p)
	if cond == nil {
		return konfidence.PromotionStateWaiting
	}
	return phases[cond.Reason].state
}

// NewestCleared returns the highest-sequence live promotion that passed its
// gates (Ready, InProgress or Blocked). Returns nil if no promotion qualifies.
func NewestCleared(promotions []konfidence.VectorPromotion) *konfidence.VectorPromotion {
	var newest *konfidence.VectorPromotion
	for i := range promotions {
		p := &promotions[i]
		if !DeriveState(p).Cleared() {
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

// TTLStatus returns whether the promotion should be deleted and the time remaining until expiry.
// Returns (false, 0) if no TTL is configured or promotion is not terminal.
func TTLStatus(p *konfidence.VectorPromotion) (shouldDelete bool, remaining time.Duration) {
	if p.Spec.TTLAfterFinished == nil || !DeriveState(p).Terminal() {
		return false, 0
	}
	expiresAt := getSucceededCondition(p).LastTransitionTime.Add(p.Spec.TTLAfterFinished.Duration)
	remaining = time.Until(expiresAt)
	return remaining <= 0, remaining
}
