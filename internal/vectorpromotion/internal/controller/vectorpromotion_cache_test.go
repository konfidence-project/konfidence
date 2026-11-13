package controller

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	utils "github.com/konfidence-project/konfidence/pkg/controller"
	"github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

func TestReconcileWithCachedGateSummary(t *testing.T) {
	for _, test := range []struct {
		name                                           string
		approval, stale, emptyStatus                   bool
		siblingSequence                                int64
		siblingCleared, siblingRunning, wantSuperseded bool
	}{
		{name: "approved/fresh", approval: true},
		{name: "approved/stale", approval: true, stale: true},
		{name: "gateless/fresh"},
		{name: "gateless/stale", stale: true},
		{name: "gateless/empty-status/stale", stale: true, emptyStatus: true},
		{name: "older-cleared/stale", approval: true, stale: true, siblingSequence: 1, siblingCleared: true},
		{name: "newer-cleared/stale", approval: true, stale: true, siblingSequence: 3, siblingCleared: true, wantSuperseded: true},
		{name: "newer-waiting/stale", approval: true, stale: true, siblingSequence: 3},
		{name: "running-sibling/requeue/stale", approval: true, stale: true, siblingSequence: 1, siblingCleared: true, siblingRunning: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			ctx := context.Background()
			current := lifecyclePromotion("current", 2, false)
			current.Spec.Gates = []konfidence.PromotionGate{}
			current.Status.Gates = nil
			if test.emptyStatus {
				current.Status = konfidence.VectorPromotionStatus{}
			}
			if test.approval {
				current.Spec.Gates = approvalGate()
				current.Status.Gates = []konfidence.PromotionGateStatus{{Type: konfidence.GateTypeExplicitApproval, State: konfidence.PromotionGateStateSatisfied}}
			}
			cached := current.DeepCopy()
			landscape, stage := cachedExecutionTarget()
			objects := []client.Object{current, landscape, stage}
			var sibling *konfidence.VectorPromotion
			if test.siblingSequence != 0 {
				sibling = lifecyclePromotion("sibling", test.siblingSequence, test.siblingCleared)
				if test.siblingRunning {
					now := metav1.Now()
					sibling.Status.StartTime = &now
				}
				objects = append(objects, sibling)
			}
			base := lifecycleClient(t, objects...)
			c := interceptor.NewClient(base, interceptor.Funcs{
				List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					if err := c.List(ctx, list, opts...); err != nil {
						return err
					}
					if promotions, ok := list.(*konfidence.VectorPromotionList); ok && test.stale {
						for i := range promotions.Items {
							if promotions.Items[i].UID == cached.UID {
								promotions.Items[i] = *cached.DeepCopy()
							}
						}
					}
					return nil
				},
			})
			r := &VectorPromotionReconciler{Client: c, Recorder: events.NewFakeRecorder(20)}
			key := client.ObjectKeyFromObject(current)
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
			g.Expect(err).NotTo(gomega.HaveOccurred())
			if test.siblingRunning {
				g.Expect(result.RequeueAfter).To(gomega.Equal(siblingBlockedRequeueInterval))
				g.Expect(base.Get(ctx, key, current)).To(gomega.Succeed())
				g.Expect(current.Status.StartTime).To(gomega.BeNil())
				g.Expect(base.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
				g.Expect(stage.Spec.Vector).To(gomega.BeEmpty())
				g.Expect(base.Get(ctx, client.ObjectKeyFromObject(sibling), sibling)).To(gomega.Succeed())
				finish(metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, "done")(sibling)
				g.Expect(base.Status().Update(ctx, sibling)).To(gomega.Succeed())
				_, err = r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
				g.Expect(err).NotTo(gomega.HaveOccurred())
			}
			g.Expect(base.Get(ctx, key, current)).To(gomega.Succeed())
			condition := meta.FindStatusCondition(current.Status.Conditions, konfidence.ConditionTypeSucceeded)
			g.Expect(condition).NotTo(gomega.BeNil())
			wantStatus := metav1.ConditionTrue
			if test.wantSuperseded {
				wantStatus = metav1.ConditionFalse
				g.Expect(condition.Reason).To(gomega.Equal(konfidence.ReasonPromotionSuperseded))
			}
			g.Expect(condition.Status).To(gomega.Equal(wantStatus))
			g.Expect(base.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
			wantVector := current.Spec.Vector
			if test.wantSuperseded {
				wantVector = ""
			}
			g.Expect(stage.Spec.Vector).To(gomega.Equal(wantVector))
			if sibling != nil {
				g.Expect(base.Get(ctx, client.ObjectKeyFromObject(sibling), sibling)).To(gomega.Succeed())
				wantSiblingStatus := metav1.ConditionUnknown
				if test.siblingRunning {
					wantSiblingStatus = metav1.ConditionTrue
				}
				g.Expect(meta.FindStatusCondition(sibling.Status.Conditions, konfidence.ConditionTypeSucceeded).Status).To(gomega.Equal(wantSiblingStatus))
			}
		})
	}
}

func TestStageFencePreventsExecutionFromStaleCache(t *testing.T) {
	t.Run("different vectors", func(t *testing.T) { testStageFencePreventsExecutionFromStaleCache(t, false) })
	t.Run("same vector", func(t *testing.T) { testStageFencePreventsExecutionFromStaleCache(t, true) })
}

func testStageFencePreventsExecutionFromStaleCache(t *testing.T, sameVector bool) {
	t.Helper()
	g := gomega.NewWithT(t)
	ctx := context.Background()
	older := lifecyclePromotion("older", 1, true)
	newer := lifecyclePromotion("newer", 2, true)
	if !sameVector {
		newer.Spec.Vector = "registry.example//konfidence.io/promo/app:2.0.0"
	}
	landscape, stage := cachedExecutionTarget()
	base := lifecycleClient(t, older, newer, landscape, stage)
	g.Expect(base.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
	cachedStage := stage.DeepCopy()
	c := interceptor.NewClient(base, interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if target, ok := obj.(*konfidence.Stage); ok {
				*target = *cachedStage.DeepCopy()
				return nil
			}
			return c.Get(ctx, key, obj, opts...)
		},
		List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			if err := c.List(ctx, list, opts...); err != nil {
				return err
			}
			if promotions, ok := list.(*konfidence.VectorPromotionList); ok {
				// The sibling cache has not observed the newer promotion at all.
				promotions.Items = excludeTestPromotion(promotions.Items, newer.UID)
			}
			return nil
		},
	})
	r := &VectorPromotionReconciler{Client: c, Recorder: events.NewFakeRecorder(20)}
	_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(newer)})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(base.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
	g.Expect(stage.Spec.Vector).To(gomega.Equal(newer.Spec.Vector))

	// The fence survives audit-record deletion and a reconciler restart.
	g.Expect(base.Delete(ctx, newer)).To(gomega.Succeed())
	r = &VectorPromotionReconciler{Client: c, Recorder: events.NewFakeRecorder(20)}
	_, err = r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(older)})
	g.Expect(apierrors.IsConflict(err)).To(gomega.BeTrue(), "a stale Stage must reject the rollback atomically")
	g.Expect(base.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
	g.Expect(stage.Spec.Vector).To(gomega.Equal(newer.Spec.Vector))

	cachedStage = stage.DeepCopy() // Only the Stage informer catches up.
	_, err = r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(older)})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(base.Get(ctx, client.ObjectKeyFromObject(older), older)).To(gomega.Succeed())
	condition := meta.FindStatusCondition(older.Status.Conditions, konfidence.ConditionTypeSucceeded)
	g.Expect(condition.Status).To(gomega.Equal(metav1.ConditionFalse))
	g.Expect(condition.Reason).To(gomega.Equal(konfidence.ReasonPromotionSuperseded))
	g.Expect(condition.Message).To(gomega.ContainSubstring("sequence 2"))
	g.Expect(base.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
	g.Expect(stage.Spec.Vector).To(gomega.Equal(newer.Spec.Vector))
}

func excludeTestPromotion(promotions []konfidence.VectorPromotion, uid types.UID) []konfidence.VectorPromotion {
	filtered := promotions[:0]
	for _, p := range promotions {
		if p.UID != uid {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func TestStageFenceKeepsSeparateConfigWatermarks(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()
	a := lifecyclePromotion("a", 2, true)
	b := lifecyclePromotion("b", 1, true)
	b.Spec.VectorPromotionConfigName = "other-config"
	b.Spec.Vector = "registry.example//konfidence.io/promo/app:other"
	_, stage := cachedExecutionTarget()
	c := lifecycleClient(t, stage)
	r := &VectorPromotionReconciler{Client: c}
	for _, p := range []*konfidence.VectorPromotion{a, b} {
		g.Expect(c.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
		g.Expect(r.promoteStage(ctx, stage, p)).To(gomega.Succeed())
	}
	a.Spec.Sequence = 1
	g.Expect(c.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
	g.Expect(errors.Is(r.promoteStage(ctx, stage, a), errNewerPromotionExecuted)).To(gomega.BeTrue())
	g.Expect(stage.Spec.Vector).To(gomega.Equal(b.Spec.Vector))
	sequences := map[string]int64{}
	g.Expect(json.Unmarshal([]byte(stage.Annotations[utils.PromotionSequencesAnnotation]), &sequences)).To(gomega.Succeed())
	g.Expect(sequences).To(gomega.Equal(map[string]int64{"project/config": 2, "project/other-config": 1}))
}

func TestStageFenceDistinguishesRecreatedConfigs(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()
	p := lifecyclePromotion("current", 20, true)
	controller := true
	p.OwnerReferences = []metav1.OwnerReference{{APIVersion: konfidence.GroupVersion.String(), Kind: konfidence.VectorPromotionConfigKind,
		Name: p.Spec.VectorPromotionConfigName, UID: "old-config", Controller: &controller}}
	_, stage := cachedExecutionTarget()
	c := lifecycleClient(t, stage)
	r := &VectorPromotionReconciler{Client: c}
	g.Expect(c.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
	g.Expect(r.promoteStage(ctx, stage, p)).To(gomega.Succeed())
	p.OwnerReferences[0].UID = "new-config"
	p.Spec.Sequence = 1
	g.Expect(c.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
	g.Expect(r.promoteStage(ctx, stage, p)).To(gomega.Succeed())
	sequences := map[string]int64{}
	g.Expect(json.Unmarshal([]byte(stage.Annotations[utils.PromotionSequencesAnnotation]), &sequences)).To(gomega.Succeed())
	g.Expect(sequences).To(gomega.Equal(map[string]int64{"project/config/old-config": 20, "project/config/new-config": 1}))
	before := stage.DeepCopy()
	g.Expect(r.promoteStage(ctx, stage, p)).To(gomega.Succeed())
	g.Expect(c.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
	g.Expect(stage).To(gomega.Equal(before), "retrying the same sequence is a no-op")
}

func TestInvalidStageFenceFailsClosed(t *testing.T) {
	for _, raw := range []string{"invalid-json", "null", `{"project/config":"not-a-sequence"}`, `{"project/config":0}`, `{"project/config":-1}`} {
		t.Run(raw, func(t *testing.T) {
			g := gomega.NewWithT(t)
			p := lifecyclePromotion("current", 1, true)
			_, stage := cachedExecutionTarget()
			stage.Annotations = map[string]string{utils.PromotionSequencesAnnotation: raw}
			c := lifecycleClient(t, stage)
			g.Expect(c.Get(context.Background(), client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
			before := stage.DeepCopy()
			r := &VectorPromotionReconciler{Client: c}
			g.Expect(r.promoteStage(context.Background(), stage, p)).To(gomega.MatchError(gomega.ContainSubstring("invalid promotion sequences")))
			g.Expect(c.Get(context.Background(), client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
			g.Expect(stage).To(gomega.Equal(before))
		})
	}
}
