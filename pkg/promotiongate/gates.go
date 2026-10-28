package promotiongate

import (
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
)

// Declared reports whether the promotion declares a gate of the given type.
func Declared(p *konfidence.VectorPromotion, gateType string) bool {
	for _, gate := range p.Spec.Gates {
		if gate.Type == gateType {
			return true
		}
	}
	return false
}

// Find returns the status entry of the given gate type, or nil if none exists.
func Find(p *konfidence.VectorPromotion, gateType string) *konfidence.PromotionGateStatus {
	for i := range p.Status.Gates {
		if p.Status.Gates[i].Type == gateType {
			return &p.Status.Gates[i]
		}
	}
	return nil
}

// StateOf returns the state of the given gate type. A missing status entry
// counts as Pending.
func StateOf(p *konfidence.VectorPromotion, gateType string) konfidence.PromotionGateState {
	if entry := Find(p, gateType); entry != nil {
		return entry.State
	}
	return konfidence.PromotionGateStatePending
}

// GatesMutable reports whether a gate write can still change the promotion.
// Gates stay mutable while GatesSatisfied is not yet True and the promotion has
// not finished (Succeeded is neither True nor False); GatesSatisfied freezes
// them once every gate is satisfied.
func GatesMutable(p *konfidence.VectorPromotion) bool {
	return !meta.IsStatusConditionTrue(p.Status.Conditions, konfidence.ConditionTypeGatesSatisfied) &&
		!meta.IsStatusConditionTrue(p.Status.Conditions, konfidence.ConditionTypeSucceeded) &&
		!meta.IsStatusConditionFalse(p.Status.Conditions, konfidence.ConditionTypeSucceeded)
}
