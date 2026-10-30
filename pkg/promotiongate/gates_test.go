package promotiongate_test

import (
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/pkg/promotiongate"
)

const (
	approval = konfidence.GateTypeExplicitApproval
	e2e      = "e2e-tests.example.com"
)

func promotionWithGates(types ...string) *konfidence.VectorPromotion {
	p := &konfidence.VectorPromotion{}
	for _, t := range types {
		p.Spec.Gates = append(p.Spec.Gates, konfidence.PromotionGate{Type: t})
	}
	return p
}

func withGateState(p *konfidence.VectorPromotion, gateType string, state konfidence.PromotionGateState) *konfidence.VectorPromotion {
	p.Status.Gates = append(p.Status.Gates, konfidence.PromotionGateStatus{Type: gateType, State: state})
	return p
}

var _ = Describe("gate lookups", func() {
	It("reports whether a gate is declared", func() {
		p := promotionWithGates(approval)
		Expect(promotiongate.Declared(p, approval)).To(BeTrue())
		Expect(promotiongate.Declared(p, e2e)).To(BeFalse())
	})

	It("finds the status entry of a gate, or nil when none exists", func() {
		p := withGateState(promotionWithGates(approval), approval, konfidence.PromotionGateStateSatisfied)
		Expect(promotiongate.Find(p, approval).State).To(Equal(konfidence.PromotionGateStateSatisfied))
		Expect(promotiongate.Find(p, e2e)).To(BeNil())
	})

	It("treats a missing status entry as Pending", func() {
		p := promotionWithGates(approval)
		Expect(promotiongate.StateOf(p, approval)).To(Equal(konfidence.PromotionGateStatePending))

		withGateState(p, approval, konfidence.PromotionGateStateSatisfied)
		Expect(promotiongate.StateOf(p, approval)).To(Equal(konfidence.PromotionGateStateSatisfied))
	})
})
