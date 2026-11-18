package controller

import (
	"context"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
	utils "github.com/konfidence-project/konfidence/pkg/controller"
	"github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func cachedExecutionTarget() (*konfidence.Landscape, *konfidence.Stage) {
	landscape := &konfidence.Landscape{
		ObjectMeta: metav1.ObjectMeta{Name: "landscape", Namespace: "project"},
		Status:     konfidence.LandscapeStatus{Namespace: "landscape-project"},
	}
	stage := &konfidence.Stage{ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: landscape.Status.Namespace}}
	return landscape, stage
}

func TestPromotionExecutionPhases(t *testing.T) {
	for _, mode := range []string{"approved", "gateless", "bypassed"} {
		t.Run(mode, func(t *testing.T) {
			g := gomega.NewWithT(t)
			ctx := context.Background()
			p := lifecyclePromotion("current", 1, false)
			switch mode {
			case "approved":
				p.Status.Gates[0].State = konfidence.PromotionGateStateSatisfied
			case "gateless":
				p.Spec.Gates = []konfidence.PromotionGate{}
				p.Status = konfidence.VectorPromotionStatus{}
			case "bypassed":
				// Readiness already exists in the override observed by Get.
				p.Status.Override = &konfidence.PromotionOverride{Type: konfidence.PromotionOverrideTypeBypassGates}
			}
			landscape, stage := cachedExecutionTarget()
			c := lifecycleClient(t, p, landscape, stage)
			r := &VectorPromotionReconciler{Client: c, Recorder: events.NewFakeRecorder(20)}
			request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(p)}
			if mode != "bypassed" {
				result, err := r.Reconcile(ctx, request)
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(result.RequeueAfter).To(gomega.Equal(promotionResyncInterval))
				g.Expect(c.Get(ctx, request.NamespacedName, p)).To(gomega.Succeed())
				g.Expect(promotion.AllGatesSatisfied(p)).To(gomega.BeTrue())
				g.Expect(p.Status.StartTime).To(gomega.BeNil())
				g.Expect(c.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
				g.Expect(stage.Spec.Vector).To(gomega.BeEmpty())
			}
			result, err := r.Reconcile(ctx, request)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(result.RequeueAfter).To(gomega.Equal(promotionResyncInterval))
			g.Expect(c.Get(ctx, request.NamespacedName, p)).To(gomega.Succeed())
			g.Expect(promotion.IsRunning(p)).To(gomega.BeTrue())
			started := p.Status.StartTime.DeepCopy()
			g.Expect(c.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
			g.Expect(stage.Spec.Vector).To(gomega.BeEmpty(), "recording execution start must not write the Stage")

			// As after a restart, no in-memory state is required to complete.
			r = &VectorPromotionReconciler{Client: c, Recorder: events.NewFakeRecorder(20)}
			_, err = r.Reconcile(ctx, request)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(c.Get(ctx, request.NamespacedName, p)).To(gomega.Succeed())
			g.Expect(promotion.IsSucceeded(p)).To(gomega.BeTrue())
			g.Expect(p.Status.StartTime).To(gomega.Equal(started))
			g.Expect(c.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
			g.Expect(stage.Spec.Vector).To(gomega.Equal(p.Spec.Vector))
			g.Expect(stage.Annotations).To(gomega.Equal(map[string]string{utils.PromotedByAnnotation: "project/current"}))
		})
	}
}

func TestExecutionTransitionsWaitForCachedObservation(t *testing.T) {
	for _, ready := range []bool{false, true} {
		name := "readiness"
		if ready {
			name = "execution start"
		}
		t.Run(name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			ctx := context.Background()
			p := lifecyclePromotion("current", 1, ready)
			p.Status.Gates[0].State = konfidence.PromotionGateStateSatisfied
			landscape, stage := cachedExecutionTarget()
			base := lifecycleClient(t, p, landscape, stage)
			g.Expect(base.Get(ctx, client.ObjectKeyFromObject(p), p)).To(gomega.Succeed())
			cached := p.DeepCopy()
			c := interceptor.NewClient(base, interceptor.Funcs{
				Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if target, ok := obj.(*konfidence.VectorPromotion); ok {
						*target = *cached.DeepCopy()
						return nil
					}
					return c.Get(ctx, key, obj, opts...)
				},
			})
			r := &VectorPromotionReconciler{Client: c, Recorder: events.NewFakeRecorder(20)}
			request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(p)}
			result, err := r.Reconcile(ctx, request)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(result.RequeueAfter).To(gomega.Equal(promotionResyncInterval))
			_, err = r.Reconcile(ctx, request)
			g.Expect(apierrors.IsConflict(err)).To(gomega.BeTrue(), "an unrelated requeue is not a cache barrier")
			g.Expect(base.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
			g.Expect(stage.Spec.Vector).To(gomega.BeEmpty())
		})
	}
}

func TestStageConflictRechecksSiblingEligibility(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()
	current := lifecyclePromotion("current", 1, true)
	now := metav1.Now()
	current.Status.StartTime = &now
	landscape, stage := cachedExecutionTarget()
	base := lifecycleClient(t, current, landscape, stage)
	g.Expect(base.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
	staleStage := stage.DeepCopy()
	stage.Spec.Vector = "concurrent-stage-edit"
	g.Expect(base.Update(ctx, stage)).To(gomega.Succeed())
	stale := true
	c := interceptor.NewClient(base, interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if target, ok := obj.(*konfidence.Stage); ok && stale {
				*target = *staleStage.DeepCopy()
				return nil
			}
			return c.Get(ctx, key, obj, opts...)
		},
	})
	r := &VectorPromotionReconciler{Client: c, Recorder: events.NewFakeRecorder(20)}
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(current)}
	_, err := r.Reconcile(ctx, request)
	g.Expect(apierrors.IsConflict(err)).To(gomega.BeTrue())
	g.Expect(base.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
	g.Expect(stage.Spec.Vector).To(gomega.Equal("concurrent-stage-edit"))

	// A newer promotion becomes ready before retry. Retrying the entire
	// reconciliation must supersede current instead of retrying its Stage patch.
	newer := lifecyclePromotion("newer", 2, true)
	g.Expect(base.Create(ctx, newer)).To(gomega.Succeed())
	stale = false
	_, err = r.Reconcile(ctx, request)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(base.Get(ctx, request.NamespacedName, current)).To(gomega.Succeed())
	g.Expect(promotion.IsTerminal(current)).To(gomega.BeTrue())
	g.Expect(promotion.IsSucceeded(current)).To(gomega.BeFalse())
	g.Expect(base.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
	g.Expect(stage.Spec.Vector).To(gomega.Equal("concurrent-stage-edit"))
}

func TestExecutionResumesAfterStageWriteWithoutRewritingStage(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()
	p := lifecyclePromotion("current", 1, true)
	now := metav1.Now()
	p.Status.StartTime = &now
	landscape, stage := cachedExecutionTarget()
	base := lifecycleClient(t, p, landscape, stage)
	stagePatches := 0
	failCompletion := true
	c := interceptor.NewClient(base, interceptor.Funcs{
		Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
			if _, ok := obj.(*konfidence.Stage); ok {
				stagePatches++
			}
			return c.Patch(ctx, obj, patch, opts...)
		},
		SubResourcePatch: func(
			ctx context.Context, c client.Client, subresource string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption,
		) error {
			if p, ok := obj.(*konfidence.VectorPromotion); ok && promotion.IsSucceeded(p) && failCompletion {
				return apierrors.NewConflict(konfidence.GroupVersion.WithResource("vectorpromotions").GroupResource(), p.Name, nil)
			}
			return c.SubResource(subresource).Patch(ctx, obj, patch, opts...)
		},
	})
	r := &VectorPromotionReconciler{Client: c, Recorder: events.NewFakeRecorder(20)}
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(p)}
	_, err := r.Reconcile(ctx, request)
	g.Expect(apierrors.IsConflict(err)).To(gomega.BeTrue())
	g.Expect(base.Get(ctx, request.NamespacedName, p)).To(gomega.Succeed())
	g.Expect(promotion.IsRunning(p)).To(gomega.BeTrue())
	g.Expect(base.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
	g.Expect(stage.Spec.Vector).To(gomega.Equal(p.Spec.Vector))
	failCompletion = false
	r = &VectorPromotionReconciler{Client: c, Recorder: events.NewFakeRecorder(20)}
	_, err = r.Reconcile(ctx, request)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(base.Get(ctx, request.NamespacedName, p)).To(gomega.Succeed())
	g.Expect(promotion.IsSucceeded(p)).To(gomega.BeTrue())
	g.Expect(stagePatches).To(gomega.Equal(1), "completion retry must not rewrite the already promoted Stage")
}
