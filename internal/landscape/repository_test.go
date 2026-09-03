package landscape_test

import (
	"context"
	"errors"
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

func scopedLandscapeFixture(name, namespace, managedNamespace string) *konfidence.Landscape {
	l := landscapeFixture(name, namespace, name)
	l.Status = konfidence.LandscapeStatus{Namespace: managedNamespace}
	return l
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

func TestGet(t *testing.T) {
	want := landscapeFixture("dev", "kden-p-my-project", "Development")
	repo := landscape.NewRepository(fakeClient(want))

	got, err := repo.Get(context.Background(), "kden-p-my-project", "dev")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != want.Name || got.Spec.DisplayName != want.Spec.DisplayName {
		t.Fatalf("unexpected landscape: %#v", got)
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

func TestResolveScope_ReturnsAllLandscapesOfProject(t *testing.T) {
	dev := scopedLandscapeFixture("dev", "kden-p-my-project", "kden-l-dev-1234")
	staging := scopedLandscapeFixture("staging", "kden-p-my-project", "kden-l-staging-5678")
	repo := landscape.NewRepository(fakeClient(dev, staging))

	scope, err := repo.ResolveScope(context.Background(), "kden-p-my-project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scope) != 2 {
		t.Fatalf("expected 2 scope entries, got %d", len(scope))
	}
	namespaces := map[string]string{}
	for _, s := range scope {
		namespaces[s.Landscape.Name] = s.Namespace
	}
	if namespaces["dev"] != "kden-l-dev-1234" {
		t.Errorf("expected namespace %q for dev, got %q", "kden-l-dev-1234", namespaces["dev"])
	}
	if namespaces["staging"] != "kden-l-staging-5678" {
		t.Errorf("expected namespace %q for staging, got %q", "kden-l-staging-5678", namespaces["staging"])
	}
}

func TestResolveScope_WithLandscapeIdNarrowsScope(t *testing.T) {
	dev := scopedLandscapeFixture("dev", "kden-p-my-project", "kden-l-dev-1234")
	staging := scopedLandscapeFixture("staging", "kden-p-my-project", "kden-l-staging-5678")
	repo := landscape.NewRepository(fakeClient(dev, staging))

	scope, err := repo.ResolveScope(context.Background(), "kden-p-my-project", landscape.WithLandscapeId("staging"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scope) != 1 {
		t.Fatalf("expected 1 scope entry, got %d", len(scope))
	}
	if scope[0].Landscape.Name != "staging" {
		t.Errorf("expected landscape %q, got %q", "staging", scope[0].Landscape.Name)
	}
	if scope[0].Namespace != "kden-l-staging-5678" {
		t.Errorf("expected namespace %q, got %q", "kden-l-staging-5678", scope[0].Namespace)
	}
}

func TestResolveScope_UnknownLandscapeIdReturnsNotFound(t *testing.T) {
	dev := scopedLandscapeFixture("dev", "kden-p-my-project", "kden-l-dev-1234")
	repo := landscape.NewRepository(fakeClient(dev))

	scope, err := repo.ResolveScope(context.Background(), "kden-p-my-project", landscape.WithLandscapeId("nope"))
	if !errors.Is(err, landscape.ErrLandscapeNotFound) {
		t.Fatalf("expected ErrLandscapeNotFound, got %v", err)
	}
	if scope != nil {
		t.Errorf("expected no scope, got %v", scope)
	}
}

func TestResolveScope_EmptyLandscapeIdReturnsNotFound(t *testing.T) {
	dev := scopedLandscapeFixture("dev", "kden-p-my-project", "kden-l-dev-1234")
	repo := landscape.NewRepository(fakeClient(dev))

	scope, err := repo.ResolveScope(context.Background(), "kden-p-my-project", landscape.WithLandscapeId(""))
	if !errors.Is(err, landscape.ErrLandscapeNotFound) {
		t.Fatalf("expected ErrLandscapeNotFound, got %v", err)
	}
	if scope != nil {
		t.Errorf("expected no scope, got %v", scope)
	}
}

func TestResolveScope_ProvisioningLandscapeStaysInScope(t *testing.T) {
	provisioning := scopedLandscapeFixture("dev", "kden-p-my-project", "")
	repo := landscape.NewRepository(fakeClient(provisioning))

	scope, err := repo.ResolveScope(context.Background(), "kden-p-my-project", landscape.WithLandscapeId("dev"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scope) != 1 {
		t.Fatalf("expected 1 scope entry, got %d", len(scope))
	}
	if scope[0].Namespace != "" {
		t.Errorf("expected empty namespace, got %q", scope[0].Namespace)
	}
}

func TestResolveScope_OtherProjectsAreNotInScope(t *testing.T) {
	own := scopedLandscapeFixture("dev", "kden-p-project-a", "kden-l-dev-a")
	foreign := scopedLandscapeFixture("dev", "kden-p-project-b", "kden-l-dev-b")
	repo := landscape.NewRepository(fakeClient(own, foreign))

	scope, err := repo.ResolveScope(context.Background(), "kden-p-project-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scope) != 1 {
		t.Fatalf("expected 1 scope entry, got %d", len(scope))
	}
	if scope[0].Namespace != "kden-l-dev-a" {
		t.Errorf("expected namespace %q, got %q", "kden-l-dev-a", scope[0].Namespace)
	}
}
