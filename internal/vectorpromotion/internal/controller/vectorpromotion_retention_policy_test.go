package controller

import (
	"context"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
	"github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func retentionTestPromotion(name string, sequence int64, state string) *konfidence.VectorPromotion {
	p := lifecyclePromotion(name, sequence, state == "ready" || state == "running")
	switch state {
	case "unobserved":
		p.Status = konfidence.VectorPromotionStatus{}
	case "running":
		now := metav1.Now()
		p.Status.StartTime = &now
	case "succeeded":
		finish(metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, "done")(p)
	case "failed":
		finish(metav1.ConditionFalse, "ExecutionFailed", "failed")(p)
	case "rejected":
		p.Status.Override = &konfidence.PromotionOverride{Type: konfidence.PromotionOverrideTypeReject}
		finish(metav1.ConditionFalse, konfidence.ReasonPromotionRejected, "rejected")(p)
	}
	return p
}

func TestRetentionCountsOnlyEligibleTerminalPromotions(t *testing.T) {
	for _, test := range []struct {
		name       string
		promotions []*konfidence.VectorPromotion
		keep       int32
		remaining  []string
	}{
		{
			name: "no active promotions applies count to all terminal states", keep: 1,
			promotions: []*konfidence.VectorPromotion{
				retentionTestPromotion("one", 1, "succeeded"),
				retentionTestPromotion("two", 2, "failed"),
				retentionTestPromotion("three", 3, "rejected"),
			},
			remaining: []string{"three"},
		},
		{
			name: "equal terminal sequences have deterministic retention ordering", keep: 1,
			promotions: []*konfidence.VectorPromotion{
				retentionTestPromotion("b", 1, "rejected"),
				retentionTestPromotion("a", 1, "succeeded"),
			},
			remaining: []string{"a"},
		},
		{
			name: "deferred terminal promotions do not consume the eligible count", keep: 1,
			promotions: []*konfidence.VectorPromotion{
				retentionTestPromotion("four", 4, "succeeded"),
				retentionTestPromotion("one", 1, "succeeded"),
				retentionTestPromotion("active", 3, "waiting"),
				retentionTestPromotion("two", 2, "failed"),
			},
			remaining: []string{"two", "active", "four"},
		},
		{
			name: "oldest of multiple active promotions defines the boundary", keep: 1,
			promotions: []*konfidence.VectorPromotion{
				retentionTestPromotion("four", 4, "succeeded"),
				retentionTestPromotion("newer-active", 5, "ready"),
				retentionTestPromotion("oldest-active", 3, "waiting"),
				retentionTestPromotion("one", 1, "succeeded"),
				retentionTestPromotion("two", 2, "succeeded"),
			},
			remaining: []string{"two", "oldest-active", "four", "newer-active"},
		},
		{
			name: "equal sequences are deferred", keep: 0,
			promotions: []*konfidence.VectorPromotion{
				retentionTestPromotion("equal", 3, "failed"),
				retentionTestPromotion("active", 3, "ready"),
				retentionTestPromotion("older", 2, "succeeded"),
			},
			remaining: []string{"active", "equal"},
		},
		{
			name: "zero sequence with unobserved status is active", keep: 0,
			promotions: []*konfidence.VectorPromotion{
				retentionTestPromotion("terminal", 1, "failed"),
				retentionTestPromotion("active", 0, "unobserved"),
			},
			remaining: []string{"active", "terminal"},
		},
		{
			name: "rejected terminal promotions follow the same sequence rule", keep: 0,
			promotions: []*konfidence.VectorPromotion{
				retentionTestPromotion("rejected", 2, "rejected"),
				retentionTestPromotion("active", 1, "running"),
			},
			remaining: []string{"active", "rejected"},
		},
		{
			name: "zero count deletes all eligible promotions", keep: 0,
			promotions: []*konfidence.VectorPromotion{
				retentionTestPromotion("newer", 4, "succeeded"),
				retentionTestPromotion("one", 1, "succeeded"),
				retentionTestPromotion("two", 2, "failed"),
				retentionTestPromotion("active", 3, "running"),
			},
			remaining: []string{"active", "newer"},
		},
		{
			name: "active promotion newer than every terminal does not defer cleanup", keep: 1,
			promotions: []*konfidence.VectorPromotion{
				retentionTestPromotion("two", 2, "succeeded"),
				retentionTestPromotion("one", 1, "failed"),
				retentionTestPromotion("active", 3, "waiting"),
			},
			remaining: []string{"active", "two"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			ctx := context.Background()
			config := &konfidence.VectorPromotionConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "config", Namespace: "project"},
				Spec:       konfidence.VectorPromotionConfigSpec{KeepLastPromotions: &test.keep},
			}
			objects := make([]client.Object, 0, 1+len(test.promotions))
			objects = append(objects, config)
			for _, p := range test.promotions {
				objects = append(objects, p)
			}
			c := lifecycleClient(t, objects...)
			r := &VectorPromotionReconciler{Client: c}
			for _, p := range test.promotions {
				if !promotion.IsTerminal(p) {
					continue
				}
				before := &konfidence.VectorPromotionList{}
				g.Expect(c.List(ctx, before)).To(gomega.Succeed())
				key := client.ObjectKeyFromObject(p)
				result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
				g.Expect(err).NotTo(gomega.HaveOccurred())
				current := &konfidence.VectorPromotion{}
				getErr := c.Get(ctx, key, current)
				if apierrors.IsNotFound(getErr) {
					g.Expect(result.RequeueAfter).To(gomega.BeZero())
				} else {
					g.Expect(getErr).NotTo(gomega.HaveOccurred())
					g.Expect(result.RequeueAfter).To(gomega.Equal(promotionResyncInterval))
				}
				for i := range before.Items {
					sibling := &before.Items[i]
					if client.ObjectKeyFromObject(sibling) == key {
						continue
					}
					unchanged := &konfidence.VectorPromotion{}
					g.Expect(c.Get(ctx, client.ObjectKeyFromObject(sibling), unchanged)).To(gomega.Succeed())
					g.Expect(unchanged).To(gomega.Equal(sibling), "each promotion may delete only itself")
				}
			}
			list := &konfidence.VectorPromotionList{}
			g.Expect(c.List(ctx, list)).To(gomega.Succeed())
			names := make([]string, 0, len(list.Items))
			for _, p := range list.Items {
				names = append(names, p.Name)
			}
			g.Expect(names).To(gomega.ConsistOf(test.remaining))
		})
	}
}

func TestRejectedPromotionDeletesItselfOnNextReconcile(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()
	p := lifecyclePromotion("rejected", 1, false)
	p.Status.Override = &konfidence.PromotionOverride{Type: konfidence.PromotionOverrideTypeReject}
	keep := int32(0)
	config := &konfidence.VectorPromotionConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "config", Namespace: "project"},
		Spec:       konfidence.VectorPromotionConfigSpec{KeepLastPromotions: &keep},
	}
	c := lifecycleClient(t, p, config)
	r := &VectorPromotionReconciler{Client: c, Recorder: events.NewFakeRecorder(20)}
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(p)}
	result, err := r.Reconcile(ctx, request)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(result.RequeueAfter).To(gomega.Equal(promotionResyncInterval))
	g.Expect(c.Get(ctx, request.NamespacedName, p)).To(gomega.Succeed(), "rejection must be persisted before retention")
	g.Expect(promotion.IsTerminal(p)).To(gomega.BeTrue())
	g.Expect(promotion.IsSucceeded(p)).To(gomega.BeFalse())
	_, err = r.Reconcile(ctx, request)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(apierrors.IsNotFound(c.Get(ctx, request.NamespacedName, p))).To(gomega.BeTrue())
}

func TestDeferredRetentionConvergesAfterActivePromotionFinishes(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()
	older := retentionTestPromotion("older", 1, "waiting")
	newer := retentionTestPromotion("newer", 2, "succeeded")
	keep := int32(1)
	config := &konfidence.VectorPromotionConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "config", Namespace: "project"},
		Spec:       konfidence.VectorPromotionConfigSpec{KeepLastPromotions: &keep},
	}
	c := lifecycleClient(t, older, newer, config)
	r := &VectorPromotionReconciler{Client: c}
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(newer)}
	result, err := r.Reconcile(ctx, request)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(result.RequeueAfter).To(gomega.Equal(promotionResyncInterval))
	g.Expect(c.Get(ctx, client.ObjectKeyFromObject(older), older)).To(gomega.Succeed())
	finish(metav1.ConditionFalse, konfidence.ReasonPromotionSuperseded, "superseded")(older)
	g.Expect(c.Status().Update(ctx, older)).To(gomega.Succeed())
	// A new reconciler, as after restart, applies the count to both terminals.
	r = &VectorPromotionReconciler{Client: c}
	result, err = r.Reconcile(ctx, request)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(result.RequeueAfter).To(gomega.Equal(promotionResyncInterval))
	// The newest terminal never deletes older siblings. Each cleans up itself.
	result, err = r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(older)})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(result.RequeueAfter).To(gomega.BeZero())
	list := &konfidence.VectorPromotionList{}
	g.Expect(c.List(ctx, list)).To(gomega.Succeed())
	g.Expect(list.Items).To(gomega.HaveLen(1))
	g.Expect(list.Items[0].Name).To(gomega.Equal(newer.Name))
}
