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

// AllSatisfied reports whether every declared gate is Satisfied. Status
// entries for gates the promotion does not declare are ignored; a promotion
// without gates is trivially satisfied.
func AllSatisfied(p *konfidence.VectorPromotion) bool {
	for _, gate := range p.Spec.Gates {
		if StateOf(p, gate.Type) != konfidence.PromotionGateStateSatisfied {
			return false
		}
	}
	return true
}

// Open returns the declared gate types that are not Satisfied, in declaration order.
func Open(p *konfidence.VectorPromotion) []string {
	var open []string
	for _, gate := range p.Spec.Gates {
		if StateOf(p, gate.Type) != konfidence.PromotionGateStateSatisfied {
			open = append(open, gate.Type)
		}
	}
	return open
}

// Errored returns the status entries of declared gates in state Error, in
// declaration order.
func Errored(p *konfidence.VectorPromotion) []konfidence.PromotionGateStatus {
	var errored []konfidence.PromotionGateStatus
	for _, gate := range p.Spec.Gates {
		if entry := Find(p, gate.Type); entry != nil && entry.State == konfidence.PromotionGateStateError {
			errored = append(errored, *entry)
		}
	}
	return errored
}

// Seed adds a Pending status entry for every declared gate that has none yet.
// Existing entries are never touched, so a writer that set its gate before the
// promotion controller seeded keeps its state and provenance. It reports
// whether any entry was added.
func Seed(p *konfidence.VectorPromotion) bool {
	added := false
	for _, gate := range p.Spec.Gates {
		if Find(p, gate.Type) != nil {
			continue
		}
		p.Status.Gates = append(p.Status.Gates, konfidence.PromotionGateStatus{
			Type:  gate.Type,
			State: konfidence.PromotionGateStatePending,
		})
		added = true
	}
	return added
}

// Mutable reports whether the promotion's gates may still change: only while
// its GatesSatisfied condition is not True and it has not finished (Succeeded
// is not True or False). GatesSatisfied latches True once every gate is
// satisfied, which freezes the gates.
func Mutable(p *konfidence.VectorPromotion) bool {
	return !meta.IsStatusConditionTrue(p.Status.Conditions, konfidence.ConditionTypeGatesSatisfied) &&
		!meta.IsStatusConditionTrue(p.Status.Conditions, konfidence.ConditionTypeSucceeded) &&
		!meta.IsStatusConditionFalse(p.Status.Conditions, konfidence.ConditionTypeSucceeded)
}
