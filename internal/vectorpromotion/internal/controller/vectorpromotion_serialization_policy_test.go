package controller

import (
	"context"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSupersedesOlderPromotionsStates(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*konfidence.VectorPromotion)
		want   bool
	}{
		{name: "waiting"},
		{name: "gate error", change: func(p *konfidence.VectorPromotion) {
			p.Status.Gates[0].State = konfidence.PromotionGateStateError
		}},
		{name: "ready", want: true, change: func(p *konfidence.VectorPromotion) {
			setCondition(konfidence.ConditionTypeGatesSatisfied, metav1.ConditionTrue, "Test", "ready")(p)
		}},
		{name: "bypassed", want: true, change: func(p *konfidence.VectorPromotion) {
			p.Status.Override = &konfidence.PromotionOverride{Type: konfidence.PromotionOverrideTypeBypassGates}
		}},
		{name: "succeeded without gate summary", want: true, change: func(p *konfidence.VectorPromotion) {
			finish(metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, "done")(p)
		}},
		{name: "failed", want: true, change: func(p *konfidence.VectorPromotion) {
			finish(metav1.ConditionFalse, "ExecutionFailed", "failed")(p)
		}},
		{name: "superseded", want: true, change: func(p *konfidence.VectorPromotion) {
			finish(metav1.ConditionFalse, konfidence.ReasonPromotionSuperseded, "superseded")(p)
		}},
		{name: "outstanding limit", want: true, change: func(p *konfidence.VectorPromotion) {
			finish(metav1.ConditionFalse, konfidence.ReasonPromotionOutstandingLimitReached, "limit")(p)
		}},
		{name: "rejection pending", change: func(p *konfidence.VectorPromotion) {
			p.Status.Override = &konfidence.PromotionOverride{Type: konfidence.PromotionOverrideTypeReject}
		}},
		{name: "rejected", change: func(p *konfidence.VectorPromotion) {
			p.Status.Override = &konfidence.PromotionOverride{Type: konfidence.PromotionOverrideTypeReject}
			finish(metav1.ConditionFalse, konfidence.ReasonPromotionRejected, "rejected")(p)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			current := lifecyclePromotion("current", 2, false)
			sibling := lifecyclePromotion("sibling", 3, false)
			if test.change != nil {
				test.change(sibling)
			}
			g.Expect(supersedesOlderPromotions(sibling)).To(gomega.Equal(test.want))
			g.Expect(canSupersede(sibling, current)).To(gomega.Equal(test.want))
			for _, sequence := range []int64{1, 2} {
				sibling.Spec.Sequence = sequence
				g.Expect(canSupersede(sibling, current)).To(gomega.BeFalse(), "only strictly newer siblings supersede")
			}
		})
	}
}

func TestBypassedPromotionExecutesWithoutNewerEligibleSibling(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()
	p := lifecyclePromotion("bypassed", 1, false)
	p.Status.Override = &konfidence.PromotionOverride{Type: konfidence.PromotionOverrideTypeBypassGates}
	landscape, stage := cachedExecutionTarget()
	c := lifecycleClient(t, p, landscape, stage)
	r := &VectorPromotionReconciler{Client: c, Recorder: events.NewFakeRecorder(20)}
	key := client.ObjectKeyFromObject(p)
	_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(c.Get(ctx, key, p)).To(gomega.Succeed())
	g.Expect(promotion.IsSucceeded(p)).To(gomega.BeTrue())
	g.Expect(promotion.AllGatesSatisfied(p)).To(gomega.BeFalse(), "bypass must not claim the gates passed")
	g.Expect(c.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
	g.Expect(stage.Spec.Vector).To(gomega.Equal(p.Spec.Vector))
}
