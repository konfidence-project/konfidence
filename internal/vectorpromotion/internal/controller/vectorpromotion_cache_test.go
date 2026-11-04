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
		name     string
		approval bool
		stale    bool
	}{
		{name: "approved/fresh", approval: true},
		{name: "approved/stale", approval: true, stale: true},
		{name: "gateless/fresh"},
		{name: "gateless/stale", stale: true},
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
					VectorPromotionConfigName: "config", Sequence: 1,
					Source: templateSource("template"), Target: stageTargetInLandscape("prod", "landscape"),
					Vector: testVector, Gates: []konfidence.PromotionGate{},
				},
				Status: konfidence.VectorPromotionStatus{Conditions: []metav1.Condition{
					{Type: konfidence.ConditionTypeGatesSatisfied, Status: metav1.ConditionFalse},
					{Type: konfidence.ConditionTypeSucceeded, Status: metav1.ConditionUnknown},
				}},
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
			base := fake.NewClientBuilder().WithScheme(scheme).WithObjects(promotion, landscape, stage).
				WithStatusSubresource(&konfidence.VectorPromotion{}).
				WithIndex(&konfidence.VectorPromotion{}, promotionConfigNameField, func(obj client.Object) []string {
					return []string{obj.(*konfidence.VectorPromotion).Spec.VectorPromotionConfigName}
				}).Build()
			c := interceptor.NewClient(base, interceptor.Funcs{
				List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					if promotions, ok := list.(*konfidence.VectorPromotionList); ok && test.stale {
						promotions.Items = []konfidence.VectorPromotion{*cached.DeepCopy()}
						return nil
					}
					return c.List(ctx, list, opts...)
				},
			})
			r := VectorPromotionReconciler{Client: c, Recorder: events.NewFakeRecorder(20)}
			key := types.NamespacedName{Name: promotion.Name, Namespace: promotion.Namespace}
			if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key}); err != nil {
				t.Fatal(err)
			}
			actual := &konfidence.VectorPromotion{}
			if err := base.Get(ctx, key, actual); err != nil {
				t.Fatal(err)
			}
			if !meta.IsStatusConditionTrue(actual.Status.Conditions, konfidence.ConditionTypeSucceeded) {
				t.Fatalf("expected promotion to succeed, got %v", actual.Status.Conditions)
			}
			if err := base.Get(ctx, client.ObjectKeyFromObject(stage), stage); err != nil {
				t.Fatal(err)
			}
			if stage.Spec.Vector != promotion.Spec.Vector {
				t.Fatalf("expected stage vector %q, got %q", promotion.Spec.Vector, stage.Spec.Vector)
			}
		})
	}
}
