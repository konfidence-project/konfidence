package deploymenttarget

import (
	"context"
	"testing"

	konfidencev1alpha1 "github.com/konfidence-project/konfidence/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRequestsForReference(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := konfidencev1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	gvk := corev1.SchemeGroupVersion.WithKind("Secret")
	targets := []client.Object{
		&konfidencev1alpha1.DeploymentTarget{
			ObjectMeta: metav1.ObjectMeta{Name: "matching", Namespace: "landscape"},
			Spec: konfidencev1alpha1.DeploymentTargetSpec{
				ArtifactType: "helm.example.com",
				Parameters:   &konfidencev1alpha1.Parameters{Kind: "Secret", Name: "credentials"},
			},
		},
		&konfidencev1alpha1.DeploymentTarget{
			ObjectMeta: metav1.ObjectMeta{Name: "other-name", Namespace: "landscape"},
			Spec: konfidencev1alpha1.DeploymentTargetSpec{
				ArtifactType: "helm.example.com",
				Parameters:   &konfidencev1alpha1.Parameters{Kind: "Secret", Name: "another"},
			},
		},
		&konfidencev1alpha1.DeploymentTarget{
			ObjectMeta: metav1.ObjectMeta{Name: "other-kind", Namespace: "landscape"},
			Spec: konfidencev1alpha1.DeploymentTargetSpec{
				ArtifactType: "helm.example.com",
				Parameters:   &konfidencev1alpha1.Parameters{Kind: "ConfigMap", Name: "credentials"},
			},
		},
		&konfidencev1alpha1.DeploymentTarget{
			ObjectMeta: metav1.ObjectMeta{Name: "unsupported", Namespace: "landscape"},
			Spec: konfidencev1alpha1.DeploymentTargetSpec{
				ArtifactType: "kustomize.example.com",
				Parameters:   &konfidencev1alpha1.Parameters{Kind: "Secret", Name: "credentials"},
			},
		},
		&konfidencev1alpha1.DeploymentTarget{
			ObjectMeta: metav1.ObjectMeta{Name: "other-namespace", Namespace: "other"},
			Spec: konfidencev1alpha1.DeploymentTargetSpec{
				ArtifactType: "helm.example.com",
				Parameters:   &konfidencev1alpha1.Parameters{Kind: "Secret", Name: "credentials"},
			},
		},
	}
	objects := make([]runtime.Object, len(targets))
	for i := range targets {
		objects[i] = targets[i]
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build()

	requests, err := requestsForReference(context.Background(), c,
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: "landscape"}},
		gvk, konfidencev1alpha1.ArtifactTypeSet{"helm.example.com": {}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 || requests[0].NamespacedName != (client.ObjectKey{Namespace: "landscape", Name: "matching"}) {
		t.Fatalf("requests = %v, want only landscape/matching", requests)
	}
}
