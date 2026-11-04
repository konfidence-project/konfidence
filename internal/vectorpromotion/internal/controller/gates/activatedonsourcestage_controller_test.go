package gates

import (
	"context"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/pkg/promotiongate"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	projectNamespace   = "project"
	landscapeNamespace = "landscape-dev"
	testLandscape      = "dev"
	testStage          = "dev-stage"
	testVector         = "registry.example//example/vector:1.0.0"
)

func TestActivatedOnSourceStageSatisfied(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	promotion := stagePromotion("promotion")
	ready := readyStageVersion()
	c := gateClient(t, promotion, landscape(), stage(), ready)
	r := &ActivatedOnSourceStageReconciler{Client: c}

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(promotion)})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(result.RequeueAfter).To(gomega.BeZero())

	updated := &konfidence.VectorPromotion{}
	g.Expect(c.Get(context.Background(), client.ObjectKeyFromObject(promotion), updated)).To(gomega.Succeed())
	gate := promotiongate.Find(updated, konfidence.GateTypeActivatedOnSourceStage)
	g.Expect(gate).NotTo(gomega.BeNil())
	g.Expect(gate.State).To(gomega.Equal(konfidence.PromotionGateStateSatisfied))
	g.Expect(gate.Reason).To(gomega.Equal(ReasonActivatedOnSourceStage))
	g.Expect(gate.Message).To(gomega.Equal("Vector activated on source stage"))
	g.Expect(gate.SetBy).To(gomega.Equal(activatedOnSourceStageSetBy))
	g.Expect(gate.SetAt).NotTo(gomega.BeNil())
}

func TestActivatedOnSourceStageRejectsNonStageSource(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	promotion := stagePromotion("promotion")
	promotion.Spec.Source = konfidence.PromotionSourceReference{
		Kind: konfidence.VectorTemplateKind,
		Name: "source-template",
	}
	c := gateClient(t, promotion)
	r := &ActivatedOnSourceStageReconciler{Client: c}

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(promotion)})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(result.RequeueAfter).To(gomega.BeZero())

	updated := &konfidence.VectorPromotion{}
	g.Expect(c.Get(context.Background(), client.ObjectKeyFromObject(promotion), updated)).To(gomega.Succeed())
	gate := promotiongate.Find(updated, konfidence.GateTypeActivatedOnSourceStage)
	g.Expect(gate).NotTo(gomega.BeNil())
	g.Expect(gate.State).To(gomega.Equal(konfidence.PromotionGateStateError))
	g.Expect(gate.Reason).To(gomega.Equal(ReasonSourceStageMissing))
}

func TestActivatedOnSourceStageWaitsForReadyStageVersion(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	promotion := stagePromotion("promotion")
	c := gateClient(t, promotion, landscape(), stage())
	r := &ActivatedOnSourceStageReconciler{Client: c}

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(promotion)})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(result.RequeueAfter).To(gomega.Equal(unresolvedGateRequeueAfter))

	updated := &konfidence.VectorPromotion{}
	g.Expect(c.Get(context.Background(), client.ObjectKeyFromObject(promotion), updated)).To(gomega.Succeed())
	gate := promotiongate.Find(updated, konfidence.GateTypeActivatedOnSourceStage)
	g.Expect(gate).NotTo(gomega.BeNil())
	g.Expect(gate.State).To(gomega.Equal(konfidence.PromotionGateStatePending))
	g.Expect(gate.Reason).To(gomega.Equal(ReasonWaitingForSourceStageActivation))
}

func TestActivatedOnSourceStageDoesNotChangeSatisfiedGate(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	promotion := stagePromotion("promotion")
	setAt := metav1.Now()
	promotion.Status.Gates = []konfidence.PromotionGateStatus{{
		Type:   konfidence.GateTypeActivatedOnSourceStage,
		State:  konfidence.PromotionGateStateSatisfied,
		Reason: "PreviouslySatisfied",
		SetBy:  "test",
		SetAt:  &setAt,
	}}
	promotion.Spec.Source = konfidence.PromotionSourceReference{Kind: konfidence.VectorTemplateKind, Name: "invalid-now"}
	c := gateClient(t, promotion)
	r := &ActivatedOnSourceStageReconciler{Client: c}

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(promotion)})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(result.RequeueAfter).To(gomega.BeZero())

	updated := &konfidence.VectorPromotion{}
	g.Expect(c.Get(context.Background(), client.ObjectKeyFromObject(promotion), updated)).To(gomega.Succeed())
	g.Expect(promotiongate.Find(updated, konfidence.GateTypeActivatedOnSourceStage).Reason).
		To(gomega.Equal("PreviouslySatisfied"))
}

func TestActivatedOnSourceStagePeriodicReconcileRecoversMissedReadyEvent(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()
	promotion := stagePromotion("promotion")
	c := gateClient(t, promotion, landscape(), stage())
	r := &ActivatedOnSourceStageReconciler{Client: c}
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(promotion)}

	result, err := r.Reconcile(ctx, request)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(result.RequeueAfter).To(gomega.Equal(unresolvedGateRequeueAfter))

	g.Expect(c.Create(ctx, readyStageVersion())).To(gomega.Succeed())
	result, err = r.Reconcile(ctx, request)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(result.RequeueAfter).To(gomega.BeZero())

	updated := &konfidence.VectorPromotion{}
	g.Expect(c.Get(ctx, client.ObjectKeyFromObject(promotion), updated)).To(gomega.Succeed())
	g.Expect(promotiongate.StateOf(updated, konfidence.GateTypeActivatedOnSourceStage)).
		To(gomega.Equal(konfidence.PromotionGateStateSatisfied))
}

func TestActivatedOnSourceStagePeriodicReconcileRecoversUnavailableLandscape(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	ctx := context.Background()
	promotion := stagePromotion("promotion")
	unavailableLandscape := landscape()
	unavailableLandscape.Status.Namespace = ""
	c := gateClient(t, promotion, unavailableLandscape, stage(), readyStageVersion())
	r := &ActivatedOnSourceStageReconciler{Client: c}
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(promotion)}

	result, err := r.Reconcile(ctx, request)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(result.RequeueAfter).To(gomega.Equal(unresolvedGateRequeueAfter))

	updated := &konfidence.VectorPromotion{}
	g.Expect(c.Get(ctx, client.ObjectKeyFromObject(promotion), updated)).To(gomega.Succeed())
	g.Expect(promotiongate.StateOf(updated, konfidence.GateTypeActivatedOnSourceStage)).
		To(gomega.Equal(konfidence.PromotionGateStateError))

	unavailableLandscape.Status.Namespace = landscapeNamespace
	g.Expect(c.Update(ctx, unavailableLandscape)).To(gomega.Succeed())
	result, err = r.Reconcile(ctx, request)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(result.RequeueAfter).To(gomega.BeZero())

	g.Expect(c.Get(ctx, client.ObjectKeyFromObject(promotion), updated)).To(gomega.Succeed())
	g.Expect(promotiongate.StateOf(updated, konfidence.GateTypeActivatedOnSourceStage)).
		To(gomega.Equal(konfidence.PromotionGateStateSatisfied))
}

func TestReadyStageVersionMapsOnlyMatchingWaitingPromotions(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	matching := stagePromotion("matching")
	otherVector := stagePromotion("other-vector")
	otherVector.Spec.Vector = "registry.example//example/vector:2.0.0"
	finished := stagePromotion("finished")
	finished.Status.Conditions = []metav1.Condition{{
		Type: konfidence.ConditionTypeSucceeded, Status: metav1.ConditionTrue,
	}}
	objects := []client.Object{matching, otherVector, finished, landscape()}
	c := gateClientWithIndexes(t, objects...)
	r := &ActivatedOnSourceStageReconciler{Client: c}

	requests := r.mapStageVersionToPromotions(context.Background(), readyStageVersion())
	g.Expect(requests).To(gomega.Equal([]reconcile.Request{{
		NamespacedName: types.NamespacedName{Namespace: projectNamespace, Name: matching.Name},
	}}))
}

func gateClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := konfidence.AddToScheme(scheme); err != nil {
		t.Fatalf("register scheme: %v", err)
	}
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&konfidence.VectorPromotion{}).
		WithObjects(objects...).
		Build()
}

func gateClientWithIndexes(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := konfidence.AddToScheme(scheme); err != nil {
		t.Fatalf("register scheme: %v", err)
	}
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		WithIndex(&konfidence.VectorPromotion{}, promotionSourceStageField, func(obj client.Object) []string {
			promotion := obj.(*konfidence.VectorPromotion)
			if promotion.Spec.Source.Kind != konfidence.StageKind ||
				!promotiongate.Declared(promotion, konfidence.GateTypeActivatedOnSourceStage) {
				return nil
			}
			return []string{sourceStageKey(promotion.Spec.Source.Landscape, promotion.Spec.Source.Name)}
		}).
		WithIndex(&konfidence.Landscape{}, activatedOnSourceStageLandscapeNamespace, func(obj client.Object) []string {
			return []string{obj.(*konfidence.Landscape).Status.Namespace}
		}).
		Build()
}

func stagePromotion(name string) *konfidence.VectorPromotion {
	return &konfidence.VectorPromotion{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: projectNamespace},
		Spec: konfidence.VectorPromotionSpec{
			VectorPromotionConfigName: "dev-to-test",
			Source: konfidence.PromotionSourceReference{
				Kind: konfidence.StageKind, Name: testStage, Landscape: testLandscape,
			},
			Target:   konfidence.PromotionTargetReference{Kind: konfidence.StageKind, Name: "test-stage", Landscape: "test"},
			Vector:   testVector,
			Sequence: 1,
			Gates: []konfidence.PromotionGate{{
				Type: konfidence.GateTypeActivatedOnSourceStage,
			}},
		},
	}
}

func landscape() *konfidence.Landscape {
	return &konfidence.Landscape{
		ObjectMeta: metav1.ObjectMeta{Name: testLandscape, Namespace: projectNamespace},
		Status:     konfidence.LandscapeStatus{Namespace: landscapeNamespace},
	}
}

func stage() *konfidence.Stage {
	return &konfidence.Stage{ObjectMeta: metav1.ObjectMeta{Name: testStage, Namespace: landscapeNamespace}}
}

func readyStageVersion() *konfidence.StageVersion {
	return &konfidence.StageVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "dev-stage-version", Namespace: landscapeNamespace},
		Spec: konfidence.StageVersionSpec{
			Vector: testVector, StageGeneration: 1, StageRef: &konfidence.StageReference{Name: testStage},
		},
		Status: konfidence.StageVersionStatus{Conditions: []metav1.Condition{{
			Type: konfidence.StageVersionReady, Status: metav1.ConditionTrue,
		}}},
	}
}
