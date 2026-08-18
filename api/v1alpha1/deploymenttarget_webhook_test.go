package v1alpha1

import (
	"context"
	"testing"

	controller "github.com/konfidence-project/konfidence/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDeploymentTargetValidation(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core API to scheme: %v", err)
	}
	if err := AddToScheme(scheme); err != nil {
		t.Fatalf("add Konfidence API to scheme: %v", err)
	}

	landscapeNamespace := func(name string) *corev1.Namespace {
		return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				controller.ProjectTypeLabel:   "landscape",
				controller.LandscapeNameLabel: name,
			},
		}}
	}
	target := func(name, namespace, deploymentType string) *DeploymentTarget {
		return &DeploymentTarget{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec:       DeploymentTargetSpec{Type: deploymentType},
		}
	}

	t.Run("accepts a unique type in a landscape namespace", func(t *testing.T) {
		ns := landscapeNamespace("landscape")
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns).Build()
		validator := &DeploymentTargetValidator{Client: c}

		if _, err := validator.ValidateCreate(context.Background(), target("helm", ns.Name, "konfidence.cloud/helm")); err != nil {
			t.Fatalf("validate DeploymentTarget: %v", err)
		}
	})

	t.Run("rejects a duplicate type in the same namespace", func(t *testing.T) {
		ns := landscapeNamespace("landscape")
		existing := target("existing", ns.Name, "konfidence.cloud/helm")
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns, existing).Build()
		validator := &DeploymentTargetValidator{Client: c}

		if _, err := validator.ValidateCreate(context.Background(), target("duplicate", ns.Name, existing.Spec.Type)); err == nil {
			t.Fatal("expected duplicate type validation to fail")
		}
	})

	t.Run("accepts the same type in another landscape namespace", func(t *testing.T) {
		first := landscapeNamespace("first")
		second := landscapeNamespace("second")
		existing := target("existing", first.Name, "konfidence.cloud/helm")
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(first, second, existing).Build()
		validator := &DeploymentTargetValidator{Client: c}

		if _, err := validator.ValidateCreate(context.Background(), target("other", second.Name, existing.Spec.Type)); err != nil {
			t.Fatalf("validate DeploymentTarget: %v", err)
		}
	})

	t.Run("rejects changing type to an existing type", func(t *testing.T) {
		ns := landscapeNamespace("landscape")
		existing := target("existing", ns.Name, "konfidence.cloud/helm")
		oldTarget := target("updated", ns.Name, "konfidence.cloud/kustomize")
		newTarget := oldTarget.DeepCopy()
		newTarget.Spec.Type = existing.Spec.Type
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns, existing, oldTarget).Build()
		validator := &DeploymentTargetValidator{Client: c}

		if _, err := validator.ValidateUpdate(context.Background(), oldTarget, newTarget); err == nil {
			t.Fatal("expected duplicate type validation to fail")
		}
	})

	t.Run("rejects an ordinary namespace", func(t *testing.T) {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ordinary"}}
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns).Build()
		validator := &DeploymentTargetValidator{Client: c}

		if _, err := validator.ValidateCreate(context.Background(), target("target", ns.Name, "konfidence.cloud/helm")); err == nil {
			t.Fatal("expected namespace validation to fail")
		}
	})
}
