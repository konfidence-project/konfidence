package vectordeployment_test

import (
	"context"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	"github.com/konfidence-project/konfidence/internal/vectordeployment"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func vectorDeploymentClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := konfidence.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

func scope(name, projectNamespace, namespace string) landscapedomain.ScopedLandscape {
	return landscapedomain.ScopedLandscape{
		Landscape: konfidence.Landscape{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: projectNamespace},
			Status:     konfidence.LandscapeStatus{Namespace: namespace},
		},
		Namespace: namespace,
	}
}

func vectorDeployment(name, namespace, stageVersionName string) *konfidence.VectorDeployment {
	return &konfidence.VectorDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{{
				Kind: konfidence.StageVersionKind,
				Name: stageVersionName,
			}},
		},
		Spec: konfidence.VectorDeploymentSpec{
			Vector: "https://registry.example.com/ocm//acme.example/vector:1.0.0",
		},
	}
}

func stageVersion(name, namespace, stageId string) *konfidence.StageVersion {
	return &konfidence.StageVersion{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: konfidence.StageVersionSpec{
			Vector:          "https://registry.example.com/ocm//acme.example/vector:1.0.0",
			StageGeneration: 1,
			StageRef:        &konfidence.StageReference{Name: stageId},
		},
	}
}

func TestRepositoryList(t *testing.T) {
	k8s := vectorDeploymentClient(t,
		vectorDeployment("checkout-v1", "landscape-dev", "checkout-v1"),
		stageVersion("checkout-v1", "landscape-dev", "checkout"),
		vectorDeployment("checkout-v2", "landscape-prod", "checkout-v2"),
		stageVersion("checkout-v2", "landscape-prod", "checkout"),
		vectorDeployment("hidden", "landscape-other", "hidden"),
		stageVersion("hidden", "landscape-other", "hidden"),
	)
	repository := vectordeployment.NewRepository(k8s)
	dev := scope("dev", "project-a", "landscape-dev")
	prod := scope("prod", "project-a", "landscape-prod")

	items, err := repository.ListForScope(context.Background(), []landscapedomain.ScopedLandscape{dev, prod})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected two deployments, got %#v", items)
	}

	items, err = repository.ListForScope(context.Background(), []landscapedomain.ScopedLandscape{dev})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].VectorDeployment.Name != "checkout-v1" ||
		items[0].LandscapeId != "dev" || items[0].StageId != "checkout" {
		t.Fatalf("unexpected filtered deployments: %#v", items)
	}
}

func TestRepositoryListRejectsMissingStageVersion(t *testing.T) {
	k8s := vectorDeploymentClient(t,
		vectorDeployment("checkout-v1", "landscape-dev", "missing"),
	)

	_, err := vectordeployment.NewRepository(k8s).ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
		scope("dev", "project-a", "landscape-dev"),
	})
	if err == nil {
		t.Fatal("expected missing stage version error")
	}
}

func TestStateFromConditions(t *testing.T) {
	tests := []struct {
		name       string
		conditions []metav1.Condition
		want       vectordeployment.State
	}{
		{name: "no conditions", want: vectordeployment.StateDeployingVector},
		{
			name: "deployment progressing",
			conditions: []metav1.Condition{{
				Type:   konfidence.VectorDownloadedCondition,
				Status: metav1.ConditionTrue,
			}},
			want: vectordeployment.StateDeployingVector,
		},
		{
			name: "deployment ready",
			conditions: []metav1.Condition{{
				Type:   konfidence.VectorReadyCondition,
				Status: metav1.ConditionTrue,
			}},
			want: vectordeployment.StateDeploymentReady,
		},
		{
			name: "unmet condition remains deploying",
			conditions: []metav1.Condition{{
				Type:   konfidence.VectorDataCreatedCondition,
				Status: metav1.ConditionFalse,
			}},
			want: vectordeployment.StateDeployingVector,
		},
		{
			name: "ready with an unmet milestone",
			conditions: []metav1.Condition{
				{Type: konfidence.VectorDataCreatedCondition, Status: metav1.ConditionFalse},
				{Type: konfidence.VectorReadyCondition, Status: metav1.ConditionTrue},
			},
			want: vectordeployment.StateDeploymentReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vectordeployment.StateFromConditions(tt.conditions); got != tt.want {
				t.Fatalf("StateFromConditions() = %q, want %q", got, tt.want)
			}
		})
	}
}
