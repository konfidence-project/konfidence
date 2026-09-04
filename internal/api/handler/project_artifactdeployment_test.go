package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	"github.com/konfidence-project/konfidence/internal/artifactdeployment"
	"github.com/konfidence-project/konfidence/internal/auth"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type adRepository struct {
	items []artifactdeployment.ResolvedArtifactDeployment
	err   error
	scope []landscapedomain.ScopedLandscape
}

type adLandscapeRepository struct {
	landscapes []konfidence.Landscape
	landscape  *konfidence.Landscape
	err        error
	scope      []landscapedomain.ScopedLandscape
}

type adProjectRepository struct {
	project *konfidence.Project
	err     error
}

func (r *adRepository) Get(_ context.Context, _ string) (
	*konfidence.ArtifactDeployment, error) {
	return nil, r.err
}

func (r *adProjectRepository) Get(_ context.Context, _ string) (
	*konfidence.Project, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.project, nil
}

func (r *adProjectRepository) List(_ context.Context,
	_ auth.ProjectRoles) ([]konfidence.Project, error) {
	return nil, nil
}

func (r *adRepository) ListForScope(
	_ context.Context, _ string,
	_ ...artifactdeployment.ListOption) (
	[]artifactdeployment.ResolvedArtifactDeployment, error) {
	return r.items, r.err
}

func (r *adLandscapeRepository) Get(_ context.Context, _, _ string) (
	*konfidence.Landscape, error) {
	return r.landscape, r.err
}

func (r *adLandscapeRepository) ListForProject(_ context.Context, _ string) ([]konfidence.Landscape, error) {
	return r.landscapes, r.err
}

func (r *adLandscapeRepository) ResolveScope(_ context.Context, _ string,
	_ ...landscapedomain.ScopeOption) ([]landscapedomain.ScopedLandscape, error) {
	return r.scope, r.err
}

func adContext(projectID string) context.Context {
	return session.NewContext(context.Background(), &session.Session{
		Context: session.Context{
			ProjectRoles: auth.ProjectRoles{projectID: {"viewer"}},
		}})
}

func assertADAPIError(t *testing.T, response any, err error, status int) {
	t.Helper()
	if response != nil {
		t.Fatalf("expected no response, got %T", response)
	}
	apiErr := apierror.As(err)
	if apiErr == nil || apiErr.Status != status {
		t.Fatalf("expected API error status %d, got %v", status, err)
	}
}

func TestListArtifactDeploymentsV1(t *testing.T) {
	const landscapeName = "dev"
	landscapeID := landscapeName
	selectedLandscape := &konfidence.Landscape{
		ObjectMeta: metav1.ObjectMeta{Name: landscapeName},
		Status:     konfidence.LandscapeStatus{Namespace: "landscape-dev"},
	}
	repository := &adRepository{items: []artifactdeployment.
		ResolvedArtifactDeployment{{
		ArtifactDeployment: konfidence.ArtifactDeployment{
			ObjectMeta: metav1.ObjectMeta{Name: "myapp-artifact"},
			Spec: konfidence.ArtifactDeploymentSpec{
				Component: konfidence.OCMComponent{
					Name:    "acme.example/myapp",
					Version: "1.0.0",
				},
			},
		},
		LandscapeId:         landscapeName,
		StageIds:            []string{"prod"},
		VectorDeploymentIds: []string{"vd-a"},
	}}}
	if _, err := toArtifactDeploymentResponse(repository.items[0]); err != nil {
		t.Fatalf("test fixture cannot be mapped: %v", err)
	}
	landscapeRepository := &adLandscapeRepository{scope: []landscapedomain.ScopedLandscape{{
		Landscape: *selectedLandscape,
		Namespace: selectedLandscape.Status.Namespace,
	}}}
	h := &projectHandler{
		projectRepo: &adProjectRepository{project: &konfidence.Project{
			ObjectMeta: metav1.ObjectMeta{Name: "project-a"},
			Status:     konfidence.ProjectStatus{Namespace: "kden-p-project-a"},
		}},
		landscapeRepo:          landscapeRepository,
		artifactDeploymentRepo: repository,
	}

	response, err := h.ListArtifactDeploymentsV1(adContext("project-a"),
		openapi.ListArtifactDeploymentsV1RequestObject{
			ProjectId: "project-a",
			Params: openapi.ListArtifactDeploymentsV1Params{
				LandscapeId: &landscapeID,
			},
		})
	if err != nil {
		t.Fatal(err)
	}
	data := response.(openapi.ListArtifactDeploymentsV1200JSONResponse).Data
	if len(data) != 1 {
		t.Fatalf("expected one deployment, got %#v", data)
	}
	got := data[0]
	if got.Id != "myapp-artifact" || got.LandscapeId != landscapeName ||
		len(got.StageIds) != 1 || got.StageIds[0] != "prod" ||
		len(got.VectorDeploymentIds) != 1 ||
		got.VectorDeploymentIds[0] != "vd-a" ||
		got.Artifact.ComponentName != "acme.example/myapp" ||
		got.Artifact.ComponentVersion != "1.0.0" {
		t.Fatalf("unexpected deployment response: %#v", got)
	}
}

func TestListArtifactDeploymentsV1WithoutLandscapeFilter(t *testing.T) {
	landscapes := []konfidence.Landscape{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "dev"},
			Status:     konfidence.LandscapeStatus{Namespace: "landscape-dev"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "prod"},
			Status:     konfidence.LandscapeStatus{Namespace: "landscape-prod"},
		},
	}
	landscapeRepository := &adLandscapeRepository{scope: []landscapedomain.ScopedLandscape{
		{Landscape: landscapes[0], Namespace: landscapes[0].Status.Namespace},
		{Landscape: landscapes[1], Namespace: landscapes[1].Status.Namespace},
	}}
	repository := &adRepository{}
	h := &projectHandler{
		projectRepo: &adProjectRepository{project: &konfidence.Project{
			ObjectMeta: metav1.ObjectMeta{Name: "project-a"},
			Status:     konfidence.ProjectStatus{Namespace: "kden-p-project-a"},
		}},
		landscapeRepo:          landscapeRepository,
		artifactDeploymentRepo: repository,
	}

	response, err := h.ListArtifactDeploymentsV1(adContext("project-a"),
		openapi.ListArtifactDeploymentsV1RequestObject{ProjectId: "project-a"})

	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.(openapi.ListArtifactDeploymentsV1200JSONResponse); !ok {
		t.Fatalf("expected successful response, got %T", response)
	}
}

func TestListArtifactDeploymentsV1Errors(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		h := &projectHandler{}
		response, err := h.ListArtifactDeploymentsV1(context.Background(),
			openapi.ListArtifactDeploymentsV1RequestObject{})
		assertADAPIError(t, response, err, http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		h := &projectHandler{}
		response, err := h.ListArtifactDeploymentsV1(adContext("other-project"),
			openapi.ListArtifactDeploymentsV1RequestObject{
				ProjectId: "project-a",
			})
		assertADAPIError(t, response, err, http.StatusForbidden)
	})

	t.Run("repository failure", func(t *testing.T) {
		h := &projectHandler{
			projectRepo: &adProjectRepository{project: &konfidence.Project{
				ObjectMeta: metav1.ObjectMeta{Name: "project-a"},
				Status:     konfidence.ProjectStatus{Namespace: "kden-p-project-a"},
			}},
			landscapeRepo: &adLandscapeRepository{},
			artifactDeploymentRepo: &adRepository{
				err: errors.New("cache failure"),
			},
		}
		response, err := h.ListArtifactDeploymentsV1(adContext("project-a"),
			openapi.ListArtifactDeploymentsV1RequestObject{
				ProjectId: "project-a",
			})
		assertADAPIError(t, response, err, http.StatusInternalServerError)
	})
}
