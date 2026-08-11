package landscape_test

import (
	"context"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/landscape"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(s)
	_ = konfidence.AddToScheme(s)
	return s
}

func fakeClient(objs ...client.Object) client.Client {
	return fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(objs...).Build()
}

func landscapeFixture(name, namespace, displayName string) *konfidence.Landscape {
	return &konfidence.Landscape{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       konfidence.LandscapeSpec{DisplayName: displayName},
	}
}

func TestListForProject_ReturnsLandscapes(t *testing.T) {
	l1 := landscapeFixture("dev", "kden-p-my-project", "Dev")
	l2 := landscapeFixture("staging", "kden-p-my-project", "Staging")
	repo := landscape.NewRepository(fakeClient(l1, l2))

	result, err := repo.ListForProject(context.Background(), "kden-p-my-project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 landscapes, got %d", len(result))
	}
}

func TestListForProject_EmptyNamespace(t *testing.T) {
	repo := landscape.NewRepository(fakeClient())

	result, err := repo.ListForProject(context.Background(), "kden-p-empty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 landscapes, got %d", len(result))
	}
}

func TestListForProject_OnlyReturnsNamespacedLandscapes(t *testing.T) {
	lA := landscapeFixture("dev", "kden-p-project-a", "Dev A")
	lB := landscapeFixture("dev", "kden-p-project-b", "Dev B")
	repo := landscape.NewRepository(fakeClient(lA, lB))

	result, err := repo.ListForProject(context.Background(), "kden-p-project-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 landscape, got %d", len(result))
	}
	if result[0].Name != "dev" {
		t.Errorf("expected Name %q, got %q", "dev", result[0].Name)
	}
}
