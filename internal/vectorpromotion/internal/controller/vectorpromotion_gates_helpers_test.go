package controller

import (
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/konfidence-project/konfidence/pkg/promotiongate"
)

func gatedPromotion(types ...string) *konfidence.VectorPromotion {
	p := &konfidence.VectorPromotion{}
	for _, t := range types {
		p.Spec.Gates = append(p.Spec.Gates, konfidence.PromotionGate{Type: t})
	}
	return p
}

func withState(p *konfidence.VectorPromotion, gateType string, state konfidence.PromotionGateState) *konfidence.VectorPromotion {
	p.Status.Gates = append(p.Status.Gates, konfidence.PromotionGateStatus{Type: gateType, State: state})
	return p
}

var _ = Describe("gate aggregation helpers", func() {
	const approval = konfidence.GateTypeExplicitApproval

	It("treats a promotion without gates as satisfied", func() {
		p := gatedPromotion()
		Expect(allGatesSatisfied(p)).To(BeTrue())
	})

	It("treats a missing status entry as a gate not satisfied yet", func() {
		p := gatedPromotion(approval)
		Expect(allGatesSatisfied(p)).To(BeFalse())
	})

	It("requires every declared gate to be Satisfied", func() {
		p := gatedPromotion(approval, e2eGate)
		withState(p, approval, konfidence.PromotionGateStateSatisfied)
		withState(p, e2eGate, konfidence.PromotionGateStateError)
		Expect(allGatesSatisfied(p)).To(BeFalse())
		Expect(anyGateInError(p)).To(BeTrue())

		p.Status.Gates[1].State = konfidence.PromotionGateStateSatisfied
		Expect(allGatesSatisfied(p)).To(BeTrue())
	})

	It("ignores status entries for undeclared gates", func() {
		p := withState(gatedPromotion(), e2eGate, konfidence.PromotionGateStatePending)
		Expect(allGatesSatisfied(p)).To(BeTrue())
		Expect(anyGateInError(p)).To(BeFalse())
	})

	It("seeds only missing entries and keeps existing ones", func() {
		p := gatedPromotion(approval, e2eGate)
		setAt := metav1.Now()
		p.Status.Gates = []konfidence.PromotionGateStatus{{
			Type: e2eGate, State: konfidence.PromotionGateStateSatisfied, SetBy: "e2e-controller", SetAt: &setAt,
		}}

		Expect(seedGates(p)).To(BeTrue())
		Expect(p.Status.Gates).To(HaveLen(2))
		Expect(promotiongate.Find(p, e2eGate).State).To(Equal(konfidence.PromotionGateStateSatisfied))
		Expect(promotiongate.Find(p, e2eGate).SetBy).To(Equal("e2e-controller"))
		Expect(promotiongate.Find(p, approval).State).To(Equal(konfidence.PromotionGateStatePending))

		Expect(seedGates(p)).To(BeFalse())
	})
})
