package artifactdeployment_test

import (
	"context"
	"errors"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/artifactdeployment"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	landscapeId        = "dev"
	vectorDeploymentId = "vd-a"
	stageId            = "stage-dev"
	projectNamespace   = "kden-project"
	landscapeNamespace = "kden-l-dev"
)

func projectScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := konfidence.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return scheme
}

func landscapeResource(name, namespace string) *konfidence.Landscape {
	return &konfidence.Landscape{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: projectNamespace},
		Status:     konfidence.LandscapeStatus{Namespace: namespace},
	}
}

func stageVersionResource(name, landscapeNamespace, stageName string) *konfidence.StageVersion {
	return &konfidence.StageVersion{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: landscapeNamespace},
		Spec: konfidence.StageVersionSpec{
			StageRef: &konfidence.StageReference{Name: stageName},
		},
	}
}

func artifactDeploymentResource(name, landscapeNamespace, landscapeId,
	vectorDeploymentId, stageVersionName string) *konfidence.ArtifactDeployment {
	return &konfidence.ArtifactDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: landscapeNamespace,
			Labels: map[string]string{
				"konfidence.cloud/landscape-name":         landscapeId,
				"konfidence.cloud/vector-deployment-name": vectorDeploymentId,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: konfidence.StageVersionKind,
					Name: stageVersionName,
				},
			},
		},
	}
}

func TestRepositoryGet(t *testing.T) {
	want := &konfidence.ArtifactDeployment{ObjectMeta: metav1.ObjectMeta{Name: "my-project"}}
	k8s := fake.NewClientBuilder().WithScheme(projectScheme(t)).WithObjects(want).Build()

	got, err := artifactdeployment.NewRepository(k8s).Get(context.Background(), want.Name)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != want.Name {
		t.Fatalf("expected artifactdeployment %q, got %q", want.Name, got.Name)
	}
}

func TestRepositoryGetNotFound(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(projectScheme(t)).Build()
	_, err := artifactdeployment.NewRepository(k8s).Get(context.Background(), "missing")

	if !errors.Is(err, artifactdeployment.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListForScopeNoFilters(t *testing.T) {
	landscape := landscapeResource(landscapeId, landscapeNamespace)
	stageVersion := stageVersionResource("sv-1", landscapeNamespace, stageId)
	ad := artifactDeploymentResource("ad-a", landscapeNamespace, landscapeId, vectorDeploymentId, "sv-1")

	k8s := fake.NewClientBuilder().
		WithScheme(projectScheme(t)).
		WithObjects(landscape, stageVersion, ad).
		Build()

	resolved, err := artifactdeployment.NewRepository(k8s).ListForScope(context.Background(), projectNamespace)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 artifact deployment, got %d", len(resolved))
	}
	if resolved[0].ArtifactDeployment.Name != "ad-a" {
		t.Fatalf("expected ad-a, got %q", resolved[0].ArtifactDeployment.Name)
	}
	if len(resolved[0].StageIds) != 1 || resolved[0].StageIds[0] != stageId {
		t.Fatalf("expected stage %q, got %v", stageId, resolved[0].StageIds)
	}
}

func TestListForScopeFilterByLandscape(t *testing.T) {
	devLandscape := landscapeResource("dev", landscapeNamespace)
	prodLandscape := landscapeResource("prod", "kden-l-prod")

	devStageVersion := stageVersionResource("sv-1", landscapeNamespace, stageId)
	prodStageVersion := stageVersionResource("sv-2", "kden-l-prod", "stage-prod")

	devAd := artifactDeploymentResource("ad-dev", landscapeNamespace, "dev", vectorDeploymentId, "sv-1")
	prodAd := artifactDeploymentResource("ad-prod", "kden-l-prod", "prod", vectorDeploymentId, "sv-2")

	k8s := fake.NewClientBuilder().
		WithScheme(projectScheme(t)).
		WithObjects(devLandscape, prodLandscape, devStageVersion, prodStageVersion, devAd, prodAd).
		Build()

	resolved, err := artifactdeployment.NewRepository(k8s).ListForScope(
		context.Background(),
		projectNamespace,
		artifactdeployment.WithLandscapeId("dev"),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 artifact deployment, got %d", len(resolved))
	}
	if resolved[0].ArtifactDeployment.Name != "ad-dev" {
		t.Fatalf("expected ad-dev, got %q", resolved[0].ArtifactDeployment.Name)
	}
	if resolved[0].LandscapeId != "dev" {
		t.Fatalf("expected landscape dev, got %q", resolved[0].LandscapeId)
	}
}

func TestListForScopeFilterByVectorDeployment(t *testing.T) {
	landscape := landscapeResource(landscapeId, landscapeNamespace)
	stageVersion := stageVersionResource("sv-1", landscapeNamespace, stageId)
	adA := artifactDeploymentResource("ad-vd-a", landscapeNamespace, landscapeId, "vd-a", "sv-1")
	adB := artifactDeploymentResource("ad-vd-b", landscapeNamespace, landscapeId, "vd-b", "sv-1")

	k8s := fake.NewClientBuilder().
		WithScheme(projectScheme(t)).
		WithObjects(landscape, stageVersion, adA, adB).
		Build()

	resolved, err := artifactdeployment.NewRepository(k8s).ListForScope(
		context.Background(),
		projectNamespace,
		artifactdeployment.WithVectorDeploymentId("vd-a"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 artifact deployment, got %d", len(resolved))
	}
	if resolved[0].ArtifactDeployment.Name != "ad-vd-a" {
		t.Fatalf("expected ad-vd-a, got %q", resolved[0].ArtifactDeployment.Name)
	}
}

func TestListForScopeFilterByBoth(t *testing.T) {
	landscape := landscapeResource(landscapeId, landscapeNamespace)
	stageVersion := stageVersionResource("sv-1", landscapeNamespace, stageId)

	adMatch := artifactDeploymentResource("ad-match", landscapeNamespace, landscapeId, vectorDeploymentId, "sv-1")
	adWrongVD := artifactDeploymentResource("ad-wrong-vd", landscapeNamespace, landscapeId, "vd-b", "sv-1")

	k8s := fake.NewClientBuilder().
		WithScheme(projectScheme(t)).
		WithObjects(landscape, stageVersion, adMatch, adWrongVD).
		Build()

	resolved, err := artifactdeployment.NewRepository(k8s).ListForScope(
		context.Background(),
		projectNamespace,
		artifactdeployment.WithLandscapeId(landscapeId),
		artifactdeployment.WithVectorDeploymentId(vectorDeploymentId),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 artifact deployment, got %d", len(resolved))
	}
	if resolved[0].ArtifactDeployment.Name != "ad-match" {
		t.Fatalf("expected ad-match, got %q", resolved[0].ArtifactDeployment.Name)
	}
}

func TestListForScopeResolvesStageIds(t *testing.T) {
	landscape := landscapeResource(landscapeId, landscapeNamespace)
	stageVersion := stageVersionResource("sv-1", landscapeNamespace, stageId)
	ad := artifactDeploymentResource("ad-a", landscapeNamespace, landscapeId, vectorDeploymentId, "sv-1")

	k8s := fake.NewClientBuilder().
		WithScheme(projectScheme(t)).
		WithObjects(landscape, stageVersion, ad).
		Build()

	resolved, err := artifactdeployment.NewRepository(k8s).ListForScope(context.Background(), projectNamespace)
	if err != nil {
		t.Fatal(err)
	}

	if len(resolved) != 1 {
		t.Fatalf("expected 1 artifact deployment, got %d", len(resolved))
	}

	if len(resolved[0].StageIds) != 1 {
		t.Fatalf("expected 1 stage ID, got %d", len(resolved[0].StageIds))
	}
	if resolved[0].StageIds[0] != stageId {
		t.Fatalf("expected stage %q, got %q", stageId, resolved[0].StageIds[0])
	}
}

func TestListForScopeResolvesStageIdsAndVectorDeploymentIds(t *testing.T) {
	landscape := landscapeResource(landscapeId, landscapeNamespace)
	stageVersion := stageVersionResource("sv-1", landscapeNamespace, stageId)
	ad := artifactDeploymentResource("ad-a", landscapeNamespace, landscapeId, vectorDeploymentId, "sv-1")

	k8s := fake.NewClientBuilder().
		WithScheme(projectScheme(t)).
		WithObjects(landscape, stageVersion, ad).
		Build()

	resolved, err := artifactdeployment.NewRepository(k8s).ListForScope(context.Background(), projectNamespace)
	if err != nil {
		t.Fatal(err)
	}

	if len(resolved) != 1 {
		t.Fatalf("expected 1 artifact deployment, got %d", len(resolved))
	}

	if len(resolved[0].StageIds) != 1 {
		t.Fatalf("expected 1 stage ID, got %d", len(resolved[0].StageIds))
	}
	if resolved[0].StageIds[0] != stageId {
		t.Fatalf("expected stage %q, got %q", stageId, resolved[0].StageIds[0])
	}

	if len(resolved[0].VectorDeploymentIds) != 1 {
		t.Fatalf("expected 1 vector deployment ID, got %d", len(resolved[0].VectorDeploymentIds))
	}
	if resolved[0].VectorDeploymentIds[0] != vectorDeploymentId {
		t.Fatalf("expected vector deployment %q, got %q", vectorDeploymentId, resolved[0].VectorDeploymentIds[0])
	}
}
