package promotiongate_test

import (
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

var _ = Describe("gate helpers", func() {
	It("treats a promotion without gates as satisfied", func() {
		p := promotionWithGates()
		Expect(promotiongate.AllSatisfied(p)).To(BeTrue())
		Expect(promotiongate.Open(p)).To(BeEmpty())
	})

	It("treats a missing status entry as Pending", func() {
		p := promotionWithGates(approval)
		Expect(promotiongate.StateOf(p, approval)).To(Equal(konfidence.PromotionGateStatePending))
		Expect(promotiongate.AllSatisfied(p)).To(BeFalse())
		Expect(promotiongate.Open(p)).To(ConsistOf(approval))
	})

	It("requires every declared gate to be Satisfied", func() {
		p := promotionWithGates(approval, e2e)
		withGateState(p, approval, konfidence.PromotionGateStateSatisfied)
		withGateState(p, e2e, konfidence.PromotionGateStateError)
		Expect(promotiongate.AllSatisfied(p)).To(BeFalse())
		Expect(promotiongate.Open(p)).To(ConsistOf(e2e))
		Expect(promotiongate.Errored(p)).To(HaveLen(1))

		p.Status.Gates[1].State = konfidence.PromotionGateStateSatisfied
		Expect(promotiongate.AllSatisfied(p)).To(BeTrue())
	})

	It("ignores status entries for undeclared gates", func() {
		p := withGateState(promotionWithGates(), e2e, konfidence.PromotionGateStatePending)
		Expect(promotiongate.AllSatisfied(p)).To(BeTrue())
		Expect(promotiongate.Errored(p)).To(BeEmpty())
	})

	It("seeds only missing entries and keeps existing ones", func() {
		p := promotionWithGates(approval, e2e)
		setAt := metav1.Now()
		p.Status.Gates = []konfidence.PromotionGateStatus{{
			Type: e2e, State: konfidence.PromotionGateStateSatisfied, SetBy: "e2e-controller", SetAt: &setAt,
		}}

		Expect(promotiongate.Seed(p)).To(BeTrue())
		Expect(p.Status.Gates).To(HaveLen(2))
		Expect(promotiongate.Find(p, e2e).State).To(Equal(konfidence.PromotionGateStateSatisfied))
		Expect(promotiongate.Find(p, e2e).SetBy).To(Equal("e2e-controller"))
		Expect(promotiongate.Find(p, approval).State).To(Equal(konfidence.PromotionGateStatePending))

		Expect(promotiongate.Seed(p)).To(BeFalse())
	})

	DescribeTable("is mutable only while the promotion state is empty or Waiting",
		func(state konfidence.VectorPromotionState, mutable bool) {
			p := promotionWithGates(approval)
			p.Status.State = state
			Expect(promotiongate.Mutable(p)).To(Equal(mutable))
		},
		Entry("never reconciled", konfidence.VectorPromotionState(""), true),
		Entry("waiting", konfidence.PromotionStateWaiting, true),
		Entry("ready", konfidence.PromotionStateReady, false),
		Entry("in progress", konfidence.PromotionStateInProgress, false),
		Entry("blocked", konfidence.PromotionStateBlocked, false),
		Entry("superseded", konfidence.PromotionStateSuperseded, false),
	)
})

var _ = Describe("Apply", func() {
	now := metav1.Now()
	satisfy := promotiongate.Update{
		Type: approval, State: konfidence.PromotionGateStateSatisfied, Reason: "ApprovedByUser", SetBy: "alice",
	}

	It("writes state and provenance together", func() {
		p := promotionWithGates(approval)
		changed, err := promotiongate.Apply(p, satisfy, now)
		Expect(err).NotTo(HaveOccurred())
		Expect(changed).To(BeTrue())
		Expect(*promotiongate.Find(p, approval)).To(Equal(konfidence.PromotionGateStatus{
			Type: approval, State: konfidence.PromotionGateStateSatisfied, Reason: "ApprovedByUser", SetBy: "alice", SetAt: &now,
		}))
	})

	It("keeps the original provenance when the state does not change", func() {
		p := promotionWithGates(approval)
		_, err := promotiongate.Apply(p, satisfy, now)
		Expect(err).NotTo(HaveOccurred())

		again := satisfy
		again.SetBy = "bob"
		changed, err := promotiongate.Apply(p, again, metav1.Now())
		Expect(err).NotTo(HaveOccurred())
		Expect(changed).To(BeFalse())
		Expect(promotiongate.Find(p, approval).SetBy).To(Equal("alice"))
	})

	It("overwrites the provenance on a state change", func() {
		p := withGateState(promotionWithGates(e2e), e2e, konfidence.PromotionGateStateSatisfied)
		changed, err := promotiongate.Apply(p, promotiongate.Update{
			Type: e2e, State: konfidence.PromotionGateStateError, Reason: "TestsFailed", SetBy: "e2e-controller",
		}, now)
		Expect(err).NotTo(HaveOccurred())
		Expect(changed).To(BeTrue())
		Expect(p.Status.Gates).To(HaveLen(1))
		Expect(p.Status.Gates[0].State).To(Equal(konfidence.PromotionGateStateError))
		Expect(p.Status.Gates[0].Reason).To(Equal("TestsFailed"))
	})

	DescribeTable("rejects invalid updates",
		func(p *konfidence.VectorPromotion, u promotiongate.Update, expected error) {
			_, err := promotiongate.Apply(p, u, now)
			Expect(err).To(MatchError(expected))
		},
		Entry("undeclared gate", promotionWithGates(e2e), satisfy, promotiongate.ErrGateNotDeclared),
		Entry("missing setBy", promotionWithGates(approval),
			promotiongate.Update{Type: approval, State: konfidence.PromotionGateStateSatisfied}, promotiongate.ErrSetByMissing),
		Entry("missing type", promotionWithGates(approval),
			promotiongate.Update{State: konfidence.PromotionGateStateSatisfied, SetBy: "x"}, promotiongate.ErrTypeMissing),
		Entry("invalid state", promotionWithGates(approval),
			promotiongate.Update{Type: approval, State: "Open", SetBy: "x"}, promotiongate.ErrInvalidState),
		Entry("ready promotion", func() *konfidence.VectorPromotion {
			p := promotionWithGates(approval)
			p.Status.State = konfidence.PromotionStateReady
			return p
		}(), satisfy, promotiongate.ErrPromotionNotMutable),
	)
})
