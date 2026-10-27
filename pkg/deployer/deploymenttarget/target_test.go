package deploymenttarget

import (
	"context"
	"errors"
	"strings"
	"testing"

	konfidencev1alpha1 "github.com/konfidence-project/konfidence/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
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
		Spec:       konfidencev1alpha1.ArtifactDeploymentSpec{ArtifactType: konfidencev1alpha1.MustArtifactType("cloud.example.helm")},
	}
	otherType := &konfidencev1alpha1.ArtifactDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: "kustomize", Namespace: "landscape"},
		Spec:       konfidencev1alpha1.ArtifactDeploymentSpec{ArtifactType: konfidencev1alpha1.MustArtifactType("kustomize.example.com")},
	}
	otherNamespace := &konfidencev1alpha1.ArtifactDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: "helm-other-landscape", Namespace: "other"},
		Spec:       konfidencev1alpha1.ArtifactDeploymentSpec{ArtifactType: konfidencev1alpha1.MustArtifactType("helm.example.com")},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(matching, otherType, otherNamespace).Build()
	target := &konfidencev1alpha1.DeploymentTarget{
		ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "landscape"},
		Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "cloud.example.helm"},
	}

	requests, err := requestsForTarget(ctx, c, target, "cloud.example.helm")
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 || requests[0].NamespacedName != client.ObjectKeyFromObject(matching) {
		t.Fatalf("requests = %v, want only %s", requests, client.ObjectKeyFromObject(matching))
	}
}

func TestWatchedResourceGVKFromManagerScheme(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	got, err := watchedResourceGVK(&corev1.Secret{}, scheme)
	if err != nil {
		t.Fatal(err)
	}
	want := corev1.SchemeGroupVersion.WithKind("Secret")
	if got.Group != want.Group || got.Kind != want.Kind {
		t.Fatalf("watchedResourceGVK() = %s, want group/kind %s", got, want)
	}
}

func TestDeploymentTargetArtifactType(t *testing.T) {
	target := &konfidencev1alpha1.DeploymentTarget{Spec: konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "helm.example.com"}}
	if got := target.ArtifactType(); got != "helm.example.com" {
		t.Fatalf("ArtifactType() = %q, want %q", got, "helm.example.com")
	}
}

func TestFindReady(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := konfidencev1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	target := &konfidencev1alpha1.DeploymentTarget{
		ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "landscape"},
		Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "cloud.example.helm"},
		Status: konfidencev1alpha1.DeploymentTargetStatus{Conditions: []metav1.Condition{{
			Type: konfidencev1alpha1.DeploymentTargetReadyCondition, Status: metav1.ConditionTrue,
		}}},
	}
	unready := &konfidencev1alpha1.DeploymentTarget{
		ObjectMeta: metav1.ObjectMeta{Name: "unready", Namespace: "landscape"},
		Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "helm.example.com"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(target, unready).Build()

	got, err := FindReady(context.Background(), c, "landscape", "cloud.example.helm")
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
		&konfidencev1alpha1.DeploymentTarget{
			ObjectMeta: metav1.ObjectMeta{Name: "one", Namespace: "landscape"},
			Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "helm.example.com"},
			Status: konfidencev1alpha1.DeploymentTargetStatus{Conditions: []metav1.Condition{{
				Type: konfidencev1alpha1.DeploymentTargetReadyCondition, Status: metav1.ConditionTrue,
			}}},
		},
		&konfidencev1alpha1.DeploymentTarget{
			ObjectMeta: metav1.ObjectMeta{Name: "two", Namespace: "landscape"},
			Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "helm.example.com"},
			Status: konfidencev1alpha1.DeploymentTargetStatus{Conditions: []metav1.Condition{{
				Type: konfidencev1alpha1.DeploymentTargetReadyCondition, Status: metav1.ConditionTrue,
			}}},
		},
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
		return NewControllerManagedBy(nil).For(konfidencev1alpha1.ArtifactTypeSet{"cloud.helm.example": {}})
	}
	if err := valid().validateConfiguration(); err != nil {
		t.Fatalf("builder.validateConfiguration(valid) error = %v", err)
	}

	cases := []struct {
		name    string
		builder *builder
	}{
		{name: "missing For", builder: NewControllerManagedBy(nil)},
		{name: "missing artifact types", builder: NewControllerManagedBy(nil).For(nil)},
		{name: "blank artifact type", builder: NewControllerManagedBy(nil).For(konfidencev1alpha1.ArtifactTypeSet{" ": {}})},
		{name: "watcher without object", builder: valid().Watches(nil, referencedTargetsMarker)},
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
		Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "cloud.example.helm"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(dt).WithObjects(dt).Build()
	r := NewControllerManagedBy(nil).
		For(konfidencev1alpha1.ArtifactTypeSet{"cloud.example.helm": {}}).
		Named("example.com/deployer:v1.2.3").build(c, nil, "example.com/deployer:v1.2.3")
	original := dt.DeepCopy()
	r.setReady(dt, metav1.ConditionTrue, "Accepted", "valid target")
	if err := c.Status().Patch(context.Background(), dt, client.MergeFrom(original)); err != nil {
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
		Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "cloud.example.helm"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(dt).WithObjects(dt).Build()
	r := NewControllerManagedBy(nil).
		For(konfidencev1alpha1.ArtifactTypeSet{"cloud.example.helm": {}}).
		Named("example.com/deployer:v1.2.3").build(c, nil, "example.com/deployer:v1.2.3")
	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(dt)}); err != nil {
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
		Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "cloud.example.helm"},
	}
	unsupported := &konfidencev1alpha1.DeploymentTarget{
		ObjectMeta: metav1.ObjectMeta{Name: "unsupported", Namespace: "landscape"},
		Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "other.example.com"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&konfidencev1alpha1.DeploymentTarget{}).WithObjects(valid, invalid, unsupported).Build()
	r := NewControllerManagedBy(nil).
		For(konfidencev1alpha1.ArtifactTypeSet{"cloud.example.helm": {}}).
		WithValidation(func(_ context.Context, _ client.Reader, dt *konfidencev1alpha1.DeploymentTarget) (ValidationResult, error) {
			if dt.Name == invalid.Name {
				return ValidationFailure("invalid parameters"), nil
			}
			return ValidationSuccess("parameters accepted"), nil
		}).build(c, nil, defaultControllerName)

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
		if got.Status.Controller != defaultControllerName || condition == nil ||
			condition.Status != tc.status || condition.Reason != tc.reason ||
			condition.ObservedGeneration != tc.generation {
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

func TestReconcileReturnsValidationErrorAfterUpdatingStatus(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	if err := konfidencev1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	target := &konfidencev1alpha1.DeploymentTarget{
		ObjectMeta: metav1.ObjectMeta{Name: "transient", Namespace: "landscape", Generation: 1},
		Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "cloud.example.helm"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(target).WithObjects(target).Build()
	validationErr := errors.New("temporary read failure")
	r := NewControllerManagedBy(nil).
		For(konfidencev1alpha1.ArtifactTypeSet{"cloud.example.helm": {}}).
		WithValidation(func(context.Context, client.Reader, *konfidencev1alpha1.DeploymentTarget) (ValidationResult, error) {
			return nil, validationErr
		}).build(c, nil, defaultControllerName)
	if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(target)}); !errors.Is(err, validationErr) {
		t.Fatalf("Reconcile() error = %v, want %v", err, validationErr)
	}
	got := &konfidencev1alpha1.DeploymentTarget{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(target), got); err != nil {
		t.Fatal(err)
	}
	condition := meta.FindStatusCondition(got.Status.Conditions, konfidencev1alpha1.DeploymentTargetReadyCondition)
	if condition == nil || condition.Status != metav1.ConditionFalse || condition.Reason != ReasonValidationError || condition.Message != validationErr.Error() {
		t.Fatalf("Ready condition = %#v, want transient validation error", condition)
	}
}
