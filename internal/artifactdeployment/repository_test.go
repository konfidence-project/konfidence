package artifactdeployment_test

import (
	"context"
	"errors"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/artifactdeployment"
	"github.com/konfidence-project/konfidence/internal/auth"
	"github.com/konfidence-project/konfidence/internal/project"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	landscapeId         = "dev"
	vectorDeploymentId  = "vd-a"
	artifactDeploymentA = konfidence.ArtifactDeployment{ObjectMeta: metav1.ObjectMeta{
		Name: "ad-dev",
		Labels: map[string]string{
			"konfidence.cloud/landscape-name":         landscapeId,
			"konfidence.cloud/vector-deployment-name": vectorDeploymentId,
		},
	}}
	artifactDeploymentB = konfidence.ArtifactDeployment{ObjectMeta: metav1.ObjectMeta{
		Name: "ad-prod",
		Labels: map[string]string{
			"konfidence.cloud/landscape-name":         "prod",
			"konfidence.cloud/vector-deployment-name": "vd-a",
		},
	}}
)

func projectScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := konfidence.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return scheme
}

func TestRepositoryGet(t *testing.T) {
	want := &konfidence.ArtifactDeployment{ObjectMeta: metav1.ObjectMeta{Name: "my-project"}}
	k8s := fake.NewClientBuilder().WithScheme(projectScheme(t)).WithObjects(want).Build()

	got, err := artifactdeployment.NewRepository(k8s).Get(context.Background(), want.Name, auth.ProjectRoles{"my-project": {"admin"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != want.Name {
		t.Fatalf("expected artifactdeployment %q, got %q", want.Name, got.Name)
	}
}

func TestRepositoryGetNotFound(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(projectScheme(t)).Build()
	_, err := project.NewRepository(k8s).Get(context.Background(), "missing", auth.ProjectRoles{"my-project": {"admin"}})

	if !errors.Is(err, project.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRepositoryListNoFilters(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(projectScheme(t)).WithObjects(&artifactDeploymentA, &artifactDeploymentB).Build()

	ads, err := artifactdeployment.NewRepository(k8s).List(context.Background(), "my-project", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ads) != 2 {
		t.Fatalf("expected 2 artifact deployments, got %d", len(ads))
	}
}

func TestRepositoryListFilterByLandscape(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(projectScheme(t)).WithObjects(&artifactDeploymentA, &artifactDeploymentB).Build()

	ads, err := artifactdeployment.NewRepository(k8s).List(context.Background(), "my-project", &artifactdeployment.ListFilters{
		LandscapeId: &landscapeId,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ads) != 1 || ads[0].Name != "ad-dev" {
		t.Fatalf("expected only dev artifact deployment, got %#v", ads)
	}
}

func TestRepositoryListFilterByVectorDeployment(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(projectScheme(t)).WithObjects(&artifactDeploymentA, &artifactDeploymentB).Build()

	ads, err := artifactdeployment.NewRepository(k8s).List(context.Background(), "my-project", &artifactdeployment.ListFilters{
		VectorDeploymentId: &vectorDeploymentId,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ads) != 1 || ads[0].Name != "ad-one" {
		t.Fatalf("expected only vd-a artifact deployment, got %#v", ads)
	}
}

func TestRepositoryListFilterByBoth(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(projectScheme(t)).WithObjects(
		&konfidence.ArtifactDeployment{ObjectMeta: metav1.ObjectMeta{
			Name: "ad-match",
			Labels: map[string]string{
				"konfidence.cloud/landscape-name":         landscapeId,
				"konfidence.cloud/vector-deployment-name": vectorDeploymentId,
			},
		}},
		&konfidence.ArtifactDeployment{ObjectMeta: metav1.ObjectMeta{
			Name: "ad-wrong-landscape",
			Labels: map[string]string{
				"konfidence.cloud/landscape-name":         "prod",
				"konfidence.cloud/vector-deployment-name": "vd-a",
			},
		}},
		&konfidence.ArtifactDeployment{ObjectMeta: metav1.ObjectMeta{
			Name: "ad-wrong-vector",
			Labels: map[string]string{
				"konfidence.cloud/landscape-name":         "dev",
				"konfidence.cloud/vector-deployment-name": "vd-b",
			},
		}},
	).Build()

	ads, err := artifactdeployment.NewRepository(k8s).List(context.Background(), "my-project", &artifactdeployment.ListFilters{
		LandscapeId:        &landscapeId,
		VectorDeploymentId: &vectorDeploymentId,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ads) != 1 || ads[0].Name != "ad-match" {
		t.Fatalf("expected only matching artifactdeployment, got %#v", ads)
	}
}
