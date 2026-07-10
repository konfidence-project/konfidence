package promotion

import (
	"time"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getSucceededCondition(p *galaxy.VectorPromotion) *metav1.Condition {
	return meta.FindStatusCondition(p.Status.Conditions, galaxy.ConditionTypeSucceeded)
}

func IsPending(p *galaxy.VectorPromotion) bool {
	return getSucceededCondition(p) == nil
}

func IsRunning(p *galaxy.VectorPromotion) bool {
	cond := getSucceededCondition(p)
	return cond != nil && cond.Reason == galaxy.ReasonPromotionRunning
}

func IsSucceeded(p *galaxy.VectorPromotion) bool {
	cond := getSucceededCondition(p)
	return cond != nil && cond.Status == metav1.ConditionTrue
}

func IsTerminal(p *galaxy.VectorPromotion) bool {
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
		return cond.Reason != galaxy.ReasonPromotionRunning
	}
	return false
}

// TTLStatus returns whether the promotion should be deleted and the time remaining until expiry.
// Returns (false, 0) if no TTL is configured or promotion is not terminal.
func TTLStatus(p *galaxy.VectorPromotion) (shouldDelete bool, remaining time.Duration) {
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
