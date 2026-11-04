package controller

import (
	"context"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestReconcileWithCachedGateSummary(t *testing.T) {
	for _, test := range []struct {
		name            string
		approval        bool
		stale           bool
		emptyStatus     bool
		siblingSequence int64
		siblingCleared  bool
		siblingRunning  bool
		wantSuperseded  bool
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
			scheme := runtime.NewScheme()
			g.Expect(konfidence.AddToScheme(scheme)).To(gomega.Succeed())
			promotion := &konfidence.VectorPromotion{
				ObjectMeta: metav1.ObjectMeta{Name: "promotion", Namespace: "project-test", UID: "promotion-uid", Generation: 1},
				Spec: konfidence.VectorPromotionSpec{
					VectorPromotionConfigName: "config", Sequence: 2,
					Source: templateSource("template"), Target: stageTargetInLandscape("prod", "landscape"),
					Vector: testVector, Gates: []konfidence.PromotionGate{},
				},
				Status: konfidence.VectorPromotionStatus{Conditions: []metav1.Condition{
					{Type: konfidence.ConditionTypeGatesSatisfied, Status: metav1.ConditionFalse},
					{Type: konfidence.ConditionTypeSucceeded, Status: metav1.ConditionUnknown},
				}},
			}
			if test.emptyStatus {
				promotion.Status = konfidence.VectorPromotionStatus{}
			}
			if test.approval {
				promotion.Spec.Gates = approvalGate()
				promotion.Status.Gates = []konfidence.PromotionGateStatus{
					{Type: konfidence.GateTypeExplicitApproval, State: konfidence.PromotionGateStateSatisfied},
				}
			}
			cached := promotion.DeepCopy()
			landscape := &konfidence.Landscape{
				ObjectMeta: metav1.ObjectMeta{Name: "landscape", Namespace: promotion.Namespace},
				Status:     konfidence.LandscapeStatus{Namespace: "landscape-test"},
			}
			stage := &konfidence.Stage{ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: landscape.Status.Namespace}}
			objects := []client.Object{promotion, landscape, stage}
			var sibling *konfidence.VectorPromotion
			if test.siblingSequence != 0 {
				sibling = promotion.DeepCopy()
				sibling.Name = "sibling"
				sibling.UID = "sibling-uid"
				sibling.Spec.Sequence = test.siblingSequence
				sibling.Spec.Vector = "registry.example//konfidence.io/promo/app:2.0.0"
				if test.siblingCleared {
					setCondition(konfidence.ConditionTypeGatesSatisfied, metav1.ConditionTrue,
						konfidence.ReasonAllGatesSatisfied, "sibling gates satisfied")(sibling)
				}
				if test.siblingRunning {
					now := metav1.Now()
					sibling.Status.StartTime = &now
				}
				objects = append(objects, sibling)
			}
			base := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).
				WithStatusSubresource(&konfidence.VectorPromotion{}).
				WithIndex(&konfidence.VectorPromotion{}, promotionConfigNameField, func(obj client.Object) []string {
					return []string{obj.(*konfidence.VectorPromotion).Spec.VectorPromotionConfigName}
				}).Build()
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
			r := VectorPromotionReconciler{Client: c, Recorder: events.NewFakeRecorder(20)}
			key := types.NamespacedName{Name: promotion.Name, Namespace: promotion.Namespace}
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
			g.Expect(err).NotTo(gomega.HaveOccurred())
			actual := &konfidence.VectorPromotion{}
			if test.siblingRunning {
				g.Expect(result.RequeueAfter).To(gomega.Equal(siblingBlockedRequeueInterval))
				g.Expect(base.Get(ctx, key, actual)).To(gomega.Succeed())
				g.Expect(meta.IsStatusConditionTrue(actual.Status.Conditions, konfidence.ConditionTypeGatesSatisfied)).To(gomega.BeTrue())
				g.Expect(actual.Status.StartTime).To(gomega.BeNil())
				g.Expect(base.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
				g.Expect(stage.Spec.Vector).To(gomega.BeEmpty(), "promotion must not write the stage while a sibling is running")
				g.Expect(base.Get(ctx, client.ObjectKeyFromObject(sibling), sibling)).To(gomega.Succeed())
				finish(metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, "sibling finished")(sibling)
				g.Expect(base.Status().Update(ctx, sibling)).To(gomega.Succeed())
				_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
				g.Expect(err).NotTo(gomega.HaveOccurred())
			}
			g.Expect(base.Get(ctx, key, actual)).To(gomega.Succeed())
			condition := meta.FindStatusCondition(actual.Status.Conditions, konfidence.ConditionTypeSucceeded)
			g.Expect(condition).NotTo(gomega.BeNil())
			if test.wantSuperseded {
				g.Expect(condition.Status).To(gomega.Equal(metav1.ConditionFalse))
				g.Expect(condition.Reason).To(gomega.Equal(konfidence.ReasonPromotionSuperseded))
			} else {
				g.Expect(condition.Status).To(gomega.Equal(metav1.ConditionTrue))
			}
			g.Expect(base.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
			wantVector := promotion.Spec.Vector
			if test.wantSuperseded {
				wantVector = ""
			}
			g.Expect(stage.Spec.Vector).To(gomega.Equal(wantVector))
			if sibling != nil {
				g.Expect(base.Get(ctx, client.ObjectKeyFromObject(sibling), sibling)).To(gomega.Succeed())
				wantStatus := metav1.ConditionUnknown
				if test.siblingRunning {
					wantStatus = metav1.ConditionTrue
				} else if test.siblingSequence < promotion.Spec.Sequence {
					wantStatus = metav1.ConditionFalse
				}
				condition := meta.FindStatusCondition(sibling.Status.Conditions, konfidence.ConditionTypeSucceeded)
				g.Expect(condition).NotTo(gomega.BeNil())
				g.Expect(condition.Status).To(gomega.Equal(wantStatus))
			}
		})
	}
}
