package deploymenttarget

import (
	"context"
	"errors"
	"strings"
	"testing"

	konfidencev1alpha1 "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
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
		Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "cloud.example.helm"},
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

func TestDeploymentTargetControllerBuilderRequiresValidConfiguration(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := konfidencev1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	valid := func() *builder {
		return NewDeploymentTargetControllerManagedBy(nil).For(konfidencev1alpha1.ArtifactTypeSet{"cloud.helm.example": {}})
	}
	if err := valid().validateConfiguration(); err != nil {
		t.Fatalf("builder.validateConfiguration(valid) error = %v", err)
	}

	cases := []struct {
		name    string
		builder *builder
	}{
		{name: "missing For", builder: NewDeploymentTargetControllerManagedBy(nil)},
		{name: "missing artifact types", builder: NewDeploymentTargetControllerManagedBy(nil).For(nil)},
		{name: "blank artifact type", builder: NewDeploymentTargetControllerManagedBy(nil).For(konfidencev1alpha1.ArtifactTypeSet{" ": {}})},
		{name: "watcher without object", builder: valid().Watches(nil, nil)},
		{name: "watcher without handler", builder: valid().Watches(&konfidencev1alpha1.DeploymentTarget{}, nil)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.builder.validateConfiguration(); err == nil {
				t.Fatal("builder configuration validation succeeded with invalid configuration")
			}
		})
	}
	if err := valid().Complete(); err == nil {
		t.Fatal("Complete() succeeded without a manager")
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
	r, err := NewDeploymentTargetControllerManagedBy(nil).
		For(konfidencev1alpha1.ArtifactTypeSet{"cloud.example.helm": {}}).
		Named("example.com/deployer:v1.2.3").build(c, "example.com/deployer:v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	original := dt.DeepCopy()
	r.setReady(dt, metav1.ConditionTrue, "Accepted", "valid target")
	if err := r.patchStatus(context.Background(), dt, original); err != nil {
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
	r, err := NewDeploymentTargetControllerManagedBy(nil).
		For(konfidencev1alpha1.ArtifactTypeSet{"cloud.example.helm": {}}).
		Named("example.com/deployer:v1.2.3").build(c, "example.com/deployer:v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	original := dt.DeepCopy()
	r.setController(dt)
	if err := r.patchStatus(context.Background(), dt, original); err != nil {
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

func TestReconcileValidatesSupportedTargets(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	if err := konfidencev1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	valid := &konfidencev1alpha1.DeploymentTarget{
		ObjectMeta: metav1.ObjectMeta{Name: "valid", Namespace: "landscape", Generation: 2},
		Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "cloud.example.helm"},
	}
	invalid := &konfidencev1alpha1.DeploymentTarget{
		ObjectMeta: metav1.ObjectMeta{Name: "invalid", Namespace: "landscape", Generation: 3},
		Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "helm.example.com"},
	}
	unsupported := &konfidencev1alpha1.DeploymentTarget{
		ObjectMeta: metav1.ObjectMeta{Name: "unsupported", Namespace: "landscape"},
		Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "other.example.com"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&konfidencev1alpha1.DeploymentTarget{}).WithObjects(valid, invalid, unsupported).Build()
	r, err := NewDeploymentTargetControllerManagedBy(nil).
		For(konfidencev1alpha1.ArtifactTypeSet{"helm.cloud.example.com": {}}).
		WithValidation(func(_ context.Context, _ client.Reader, dt *konfidencev1alpha1.DeploymentTarget) error {
			if dt.Name == invalid.Name {
				return errors.New("invalid parameters")
			}
			return nil
		}).build(c, defaultControllerName)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name       string
		status     metav1.ConditionStatus
		reason     string
		generation int64
	}{
		{name: valid.Name, status: metav1.ConditionTrue, reason: ReasonAccepted, generation: 2},
		{name: invalid.Name, status: metav1.ConditionFalse, reason: ReasonInvalidParameters, generation: 3},
	} {
		if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKey{Namespace: "landscape", Name: tc.name}}); err != nil {
			t.Fatalf("Reconcile(%q): %v", tc.name, err)
		}
		got := &konfidencev1alpha1.DeploymentTarget{}
		if err := c.Get(ctx, client.ObjectKey{Namespace: "landscape", Name: tc.name}, got); err != nil {
			t.Fatal(err)
		}
		condition := meta.FindStatusCondition(got.Status.Conditions, konfidencev1alpha1.DeploymentTargetReadyCondition)
		if got.Status.Controller != defaultControllerName || condition == nil || condition.Status != tc.status || condition.Reason != tc.reason || condition.ObservedGeneration != tc.generation {
			t.Errorf("%s status = %#v; want controller identity and Ready=%s (%s) at generation %d", tc.name, got.Status, tc.status, tc.reason, tc.generation)
		}
	}
	if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(unsupported)}); err != nil {
		t.Fatalf("Reconcile unsupported target: %v", err)
	}
	gotUnsupported := &konfidencev1alpha1.DeploymentTarget{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(unsupported), gotUnsupported); err != nil {
		t.Fatal(err)
	}
	if gotUnsupported.Status.Controller != "" || len(gotUnsupported.Status.Conditions) != 0 {
		t.Errorf("unsupported target status changed: %#v", gotUnsupported.Status)
	}
}
