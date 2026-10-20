package target

import (
	"context"
	"strings"
	"testing"

	konfidencev1alpha1 "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnqueueArtifactDeploymentsForTarget(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	if err := konfidencev1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	matching := &konfidencev1alpha1.ArtifactDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: "helm", Namespace: "landscape"},
		Spec:       konfidencev1alpha1.ArtifactDeploymentSpec{Manifest: konfidencev1alpha1.ArtifactManifest{Type: "helm.example.com"}},
	}
	otherType := &konfidencev1alpha1.ArtifactDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: "kustomize", Namespace: "landscape"},
		Spec:       konfidencev1alpha1.ArtifactDeploymentSpec{Manifest: konfidencev1alpha1.ArtifactManifest{Type: "kustomize.example.com"}},
	}
	otherNamespace := &konfidencev1alpha1.ArtifactDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: "helm-other-landscape", Namespace: "other"},
		Spec:       konfidencev1alpha1.ArtifactDeploymentSpec{Manifest: konfidencev1alpha1.ArtifactManifest{Type: "helm.example.com"}},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(matching, otherType, otherNamespace).Build()
	target := &konfidencev1alpha1.DeploymentTarget{
		ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "landscape"},
		Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "helm.example.com"},
	}

	requests, err := requestsForTarget(ctx, c, target, "helm.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 || requests[0].NamespacedName != client.ObjectKeyFromObject(matching) {
		t.Fatalf("requests = %v, want only %s", requests, client.ObjectKeyFromObject(matching))
	}
}

func TestMatchesArtifactType(t *testing.T) {
	target := &konfidencev1alpha1.DeploymentTarget{Spec: konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "helm.example.com"}}
	if !target.MatchesArtifactType("helm.example.com") || target.MatchesArtifactType("kustomize.example.com") || (*konfidencev1alpha1.DeploymentTarget)(nil).MatchesArtifactType("helm.example.com") {
		t.Fatal("unexpected artifact type match result")
	}
}

func TestFindReady(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := konfidencev1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	target := &konfidencev1alpha1.DeploymentTarget{
		ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "landscape"},
		Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "helm.example.com"},
		Status: konfidencev1alpha1.DeploymentTargetStatus{Conditions: []metav1.Condition{{
			Type: konfidencev1alpha1.DeploymentTargetReadyCondition, Status: metav1.ConditionTrue,
		}}},
	}
	unready := &konfidencev1alpha1.DeploymentTarget{
		ObjectMeta: metav1.ObjectMeta{Name: "unready", Namespace: "landscape"},
		Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "helm.example.com"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(target, unready).Build()

	got, err := FindReady(context.Background(), c, "landscape", "helm.example.com")
	if err != nil || got.Name != target.Name {
		t.Fatalf("FindReady() = (%v, %v), want target %q", got, err, target.Name)
	}
	if _, err := FindReady(context.Background(), c, "other", "helm.example.com"); err == nil {
		t.Fatal("FindReady() for absent target succeeded")
	}
	if _, err := FindReady(context.Background(), c, "landscape", "kustomize.example.com"); err == nil {
		t.Fatal("FindReady() for only unready targets succeeded")
	}
}

func TestFindRejectsMultipleTargets(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := konfidencev1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	objects := []runtime.Object{
		&konfidencev1alpha1.DeploymentTarget{ObjectMeta: metav1.ObjectMeta{Name: "one", Namespace: "landscape"}, Spec: konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "helm.example.com"}, Status: konfidencev1alpha1.DeploymentTargetStatus{Conditions: []metav1.Condition{{Type: konfidencev1alpha1.DeploymentTargetReadyCondition, Status: metav1.ConditionTrue}}}},
		&konfidencev1alpha1.DeploymentTarget{ObjectMeta: metav1.ObjectMeta{Name: "two", Namespace: "landscape"}, Spec: konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "helm.example.com"}, Status: konfidencev1alpha1.DeploymentTargetStatus{Conditions: []metav1.Condition{{Type: konfidencev1alpha1.DeploymentTargetReadyCondition, Status: metav1.ConditionTrue}}}},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build()
	if _, err := FindReady(context.Background(), c, "landscape", "helm.example.com"); err == nil || !strings.Contains(err.Error(), "multiple DeploymentTargets") {
		t.Fatalf("FindReady() error = %v, want duplicate-target error", err)
	}
}

func TestIsReadyRequiresCurrentGeneration(t *testing.T) {
	target := &konfidencev1alpha1.DeploymentTarget{
		ObjectMeta: metav1.ObjectMeta{Generation: 3},
		Status: konfidencev1alpha1.DeploymentTargetStatus{Conditions: []metav1.Condition{{
			Type: konfidencev1alpha1.DeploymentTargetReadyCondition, Status: metav1.ConditionTrue, ObservedGeneration: 2,
		}}},
	}
	if IsReady(target) {
		t.Fatal("IsReady() = true for a stale Ready condition")
	}
	target.Status.Conditions[0].ObservedGeneration = target.Generation
	if !IsReady(target) {
		t.Fatal("IsReady() = false for a current Ready condition")
	}
}

func TestSetReady(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := konfidencev1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	dt := &konfidencev1alpha1.DeploymentTarget{
		ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "landscape", Generation: 4},
		Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "helm.example.com"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(dt).WithObjects(dt).Build()
	r := &DeploymentTargetReconciler{Client: c}
	if err := r.setReady(context.Background(), dt, metav1.ConditionTrue, "Accepted", "valid target"); err != nil {
		t.Fatal(err)
	}
	updated := &konfidencev1alpha1.DeploymentTarget{}
	if err := c.Get(context.Background(), client.ObjectKeyFromObject(dt), updated); err != nil {
		t.Fatal(err)
	}
	condition := meta.FindStatusCondition(updated.Status.Conditions, konfidencev1alpha1.DeploymentTargetReadyCondition)
	if condition == nil || condition.Status != metav1.ConditionTrue || condition.ObservedGeneration != 4 {
		t.Fatalf("Ready condition = %#v, want True at generation 4", condition)
	}
}

func TestSetController(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := konfidencev1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	dt := &konfidencev1alpha1.DeploymentTarget{
		ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "landscape"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(dt).WithObjects(dt).Build()
	r := &DeploymentTargetReconciler{Client: c, ControllerName: "example.com/deployer:v1.2.3"}
	if err := r.setController(context.Background(), dt); err != nil {
		t.Fatal(err)
	}
	updated := &konfidencev1alpha1.DeploymentTarget{}
	if err := c.Get(context.Background(), client.ObjectKeyFromObject(dt), updated); err != nil {
		t.Fatal(err)
	}
	if updated.Status.Controller != "example.com/deployer:v1.2.3" {
		t.Fatalf("status.controller = %q, want controller identity", updated.Status.Controller)
	}
}
