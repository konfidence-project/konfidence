package controller

import (
	"context"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
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
			ctx := context.Background()
			scheme := runtime.NewScheme()
			if err := konfidence.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
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
			if err != nil {
				t.Fatal(err)
			}
			actual := &konfidence.VectorPromotion{}
			if test.siblingRunning {
				if result.RequeueAfter != siblingBlockedRequeueInterval {
					t.Fatalf("expected requeue while sibling runs, got %v", result)
				}
				if err := base.Get(ctx, key, actual); err != nil {
					t.Fatal(err)
				}
				if !meta.IsStatusConditionTrue(actual.Status.Conditions, konfidence.ConditionTypeGatesSatisfied) ||
					actual.Status.StartTime != nil {
					t.Fatalf("expected promotion to wait with satisfied gates, got %+v", actual.Status)
				}
				if err := base.Get(ctx, client.ObjectKeyFromObject(stage), stage); err != nil {
					t.Fatal(err)
				}
				if stage.Spec.Vector != "" {
					t.Fatalf("promotion wrote stage while sibling was running: %q", stage.Spec.Vector)
				}
				if err := base.Get(ctx, client.ObjectKeyFromObject(sibling), sibling); err != nil {
					t.Fatal(err)
				}
				finish(metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, "sibling finished")(sibling)
				if err := base.Status().Update(ctx, sibling); err != nil {
					t.Fatal(err)
				}
				if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key}); err != nil {
					t.Fatal(err)
				}
			}
			if err := base.Get(ctx, key, actual); err != nil {
				t.Fatal(err)
			}
			if test.wantSuperseded {
				condition := meta.FindStatusCondition(actual.Status.Conditions, konfidence.ConditionTypeSucceeded)
				if condition == nil || condition.Status != metav1.ConditionFalse || condition.Reason != konfidence.ReasonPromotionSuperseded {
					t.Fatalf("expected promotion to be superseded, got %v", actual.Status.Conditions)
				}
			} else if !meta.IsStatusConditionTrue(actual.Status.Conditions, konfidence.ConditionTypeSucceeded) {
				t.Fatalf("expected promotion to succeed, got %v", actual.Status.Conditions)
			}
			if err := base.Get(ctx, client.ObjectKeyFromObject(stage), stage); err != nil {
				t.Fatal(err)
			}
			wantVector := promotion.Spec.Vector
			if test.wantSuperseded {
				wantVector = ""
			}
			if stage.Spec.Vector != wantVector {
				t.Fatalf("expected stage vector %q, got %q", wantVector, stage.Spec.Vector)
			}
			if sibling != nil {
				if err := base.Get(ctx, client.ObjectKeyFromObject(sibling), sibling); err != nil {
					t.Fatal(err)
				}
				wantStatus := metav1.ConditionUnknown
				if test.siblingRunning {
					wantStatus = metav1.ConditionTrue
				} else if test.siblingSequence < promotion.Spec.Sequence {
					wantStatus = metav1.ConditionFalse
				}
				if !meta.IsStatusConditionPresentAndEqual(sibling.Status.Conditions, konfidence.ConditionTypeSucceeded, wantStatus) {
					t.Fatalf("expected sibling Succeeded=%s, got %v", wantStatus, sibling.Status.Conditions)
				}
			}
		})
	}
}
