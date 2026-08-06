package v1alpha1

import (
	"context"
	"testing"

	controller "github.com/konfidence-project/konfidence/pkg/controller"
	pkgwebhook "github.com/konfidence-project/konfidence/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDeploymentTargetLandscapeNamespaceValidation(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core API to scheme: %v", err)
	}

	landscapeNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: "landscape",
		Labels: map[string]string{
			controller.ProjectTypeLabel:   "landscape",
			controller.LandscapeNameLabel: "test-landscape",
		},
	}}
	ordinaryNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ordinary"}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(landscapeNamespace, ordinaryNamespace).Build()
	validator := pkgwebhook.NewLandscapeNamespaceValidator[*DeploymentTarget](c, DeploymentTargetKind)

	t.Run("accepts landscape namespace", func(t *testing.T) {
		target := &DeploymentTarget{ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: landscapeNamespace.Name}}
		if _, err := validator.ValidateCreate(context.Background(), target); err != nil {
			t.Fatalf("validate DeploymentTarget: %v", err)
		}
	})

	t.Run("rejects ordinary namespace", func(t *testing.T) {
		target := &DeploymentTarget{ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: ordinaryNamespace.Name}}
		if _, err := validator.ValidateCreate(context.Background(), target); err == nil {
			t.Fatal("expected namespace validation to fail")
		}
	})
}
