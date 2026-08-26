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
	"github.com/konfidence-project/konfidence/internal/auth"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	"github.com/konfidence-project/konfidence/internal/vectordeployment"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type vectorDeploymentRepository struct {
	items []vectordeployment.ResolvedVectorDeployment
	err   error
	scope []landscapedomain.ScopedLandscape
}

type vectorDeploymentLandscapeRepository struct {
	landscapes []konfidence.Landscape
	landscape  *konfidence.Landscape
	err        error
	getCalls   int
	listCalls  int
	scope      []landscapedomain.ScopedLandscape
}

type vectorDeploymentProjectRepository struct {
	project *konfidence.Project
	err     error
}

func (r *vectorDeploymentProjectRepository) Get(_ context.Context, _ string) (*konfidence.Project, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.project, nil
}

func (r *vectorDeploymentProjectRepository) List(_ context.Context, _ auth.ProjectRoles) ([]konfidence.Project, error) {
	return nil, nil
}

func (r *vectorDeploymentRepository) ListForScope(
	_ context.Context, scope []landscapedomain.ScopedLandscape) ([]vectordeployment.ResolvedVectorDeployment, error) {
	r.scope = scope
	return r.items, r.err
}

func (r *vectorDeploymentLandscapeRepository) Get(_ context.Context, _, _ string) (*konfidence.Landscape, error) {
	r.getCalls++
	return r.landscape, r.err
}

func (r *vectorDeploymentLandscapeRepository) ListForProject(_ context.Context, _ string) ([]konfidence.Landscape, error) {
	r.listCalls++
	return r.landscapes, r.err
}

func (r *vectorDeploymentLandscapeRepository) ResolveScope(_ context.Context, _ string,
	_ ...landscapedomain.ScopeOption) ([]landscapedomain.ScopedLandscape, error) {
	r.listCalls++
	return r.scope, r.err
}

func vectorDeploymentContext(projectID string) context.Context {
	return session.NewContext(context.Background(), &session.Session{Context: session.Context{
		ProjectRoles: auth.ProjectRoles{projectID: {"viewer"}},
	}})
}

func assertVectorDeploymentAPIError(t *testing.T, response any, err error, status int) {
	t.Helper()
	if response != nil {
		t.Fatalf("expected no response, got %T", response)
	}
	apiErr := apierror.As(err)
	if apiErr == nil || apiErr.Status != status {
		t.Fatalf("expected API error status %d, got %v", status, err)
	}
}

func TestListVectorDeploymentsV1(t *testing.T) {
	const landscapeName = "dev"
	landscapeID := landscapeName
	selectedLandscape := &konfidence.Landscape{
		ObjectMeta: metav1.ObjectMeta{Name: landscapeName},
		Status:     konfidence.LandscapeStatus{Namespace: "landscape-dev"},
	}
	repository := &vectorDeploymentRepository{items: []vectordeployment.ResolvedVectorDeployment{{
		VectorDeployment: konfidence.VectorDeployment{
			ObjectMeta: metav1.ObjectMeta{Name: "checkout-v1"},
			Spec: konfidence.VectorDeploymentSpec{
				Vector: "https://registry.example.com/ocm//acme.example/checkout:1.2.3",
			},
		},
		LandscapeId: landscapeName,
		StageId:     "checkout",
	}}}
	if _, err := toVectorDeploymentResponse(repository.items[0]); err != nil {
		t.Fatalf("test fixture cannot be mapped: %v", err)
	}
	landscapeRepository := &vectorDeploymentLandscapeRepository{scope: []landscapedomain.ScopedLandscape{{
		Landscape: *selectedLandscape,
		Namespace: selectedLandscape.Status.Namespace,
	}}}
	h := &projectHandler{
		projectRepo: &vectorDeploymentProjectRepository{project: &konfidence.Project{
			ObjectMeta: metav1.ObjectMeta{Name: "project-a"},
			Status:     konfidence.ProjectStatus{Namespace: "kden-p-project-a"},
		}},
		landscapeRepo:        landscapeRepository,
		vectorDeploymentRepo: repository,
	}

	response, err := h.ListVectorDeploymentsV1(vectorDeploymentContext("project-a"),
		openapi.ListVectorDeploymentsV1RequestObject{
			ProjectId: "project-a",
			Params:    openapi.ListVectorDeploymentsV1Params{LandscapeId: &landscapeID},
		})
	if err != nil {
		t.Fatal(err)
	}
	data := response.(openapi.ListVectorDeploymentsV1200JSONResponse).Data
	if len(data) != 1 {
		t.Fatalf("expected one deployment, got %#v", data)
	}
	got := data[0]
	if got.Id != "checkout-v1" || got.LandscapeId != landscapeName || got.StageId != "checkout" ||
		got.Vector.Repository != "https://registry.example.com/ocm" ||
		got.Vector.ComponentName != "acme.example/checkout" || got.Vector.ComponentVersion != "1.2.3" ||
		got.Status != openapi.VectorDeploymentStatusDeployingVector {
		t.Fatalf("unexpected deployment response: %#v", got)
	}
	if len(repository.scope) != 1 || repository.scope[0].Landscape.Name != landscapeName {
		t.Fatalf("unexpected repository scope: %#v", repository.scope)
	}
	if landscapeRepository.getCalls != 0 || landscapeRepository.listCalls != 1 {
		t.Fatalf("unexpected landscape repository calls: get=%d list=%d", landscapeRepository.getCalls, landscapeRepository.listCalls)
	}
}

func TestListVectorDeploymentsV1WithoutLandscapeFilter(t *testing.T) {
	landscapes := []konfidence.Landscape{
		{ObjectMeta: metav1.ObjectMeta{Name: "dev"}, Status: konfidence.LandscapeStatus{Namespace: "landscape-dev"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "prod"}, Status: konfidence.LandscapeStatus{Namespace: "landscape-prod"}},
	}
	landscapeRepository := &vectorDeploymentLandscapeRepository{scope: []landscapedomain.ScopedLandscape{
		{Landscape: landscapes[0], Namespace: landscapes[0].Status.Namespace},
		{Landscape: landscapes[1], Namespace: landscapes[1].Status.Namespace},
	}}
	repository := &vectorDeploymentRepository{}
	h := &projectHandler{
		projectRepo: &vectorDeploymentProjectRepository{project: &konfidence.Project{
			ObjectMeta: metav1.ObjectMeta{Name: "project-a"},
			Status:     konfidence.ProjectStatus{Namespace: "kden-p-project-a"},
		}},
		landscapeRepo:        landscapeRepository,
		vectorDeploymentRepo: repository,
	}

	response, err := h.ListVectorDeploymentsV1(vectorDeploymentContext("project-a"),
		openapi.ListVectorDeploymentsV1RequestObject{ProjectId: "project-a"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.(openapi.ListVectorDeploymentsV1200JSONResponse); !ok {
		t.Fatalf("expected successful response, got %T", response)
	}
	if len(repository.scope) != 2 || landscapeRepository.getCalls != 0 || landscapeRepository.listCalls != 1 {
		t.Fatalf("unexpected landscape selection: %#v", repository.scope)
	}
}

func TestListVectorDeploymentsV1Errors(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		h := &projectHandler{}
		response, err := h.ListVectorDeploymentsV1(context.Background(), openapi.ListVectorDeploymentsV1RequestObject{})
		assertVectorDeploymentAPIError(t, response, err, http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		h := &projectHandler{}
		response, err := h.ListVectorDeploymentsV1(vectorDeploymentContext("other-project"),
			openapi.ListVectorDeploymentsV1RequestObject{ProjectId: "project-a"})
		assertVectorDeploymentAPIError(t, response, err, http.StatusForbidden)
	})

	t.Run("repository failure", func(t *testing.T) {
		h := &projectHandler{
			projectRepo: &vectorDeploymentProjectRepository{project: &konfidence.Project{
				ObjectMeta: metav1.ObjectMeta{Name: "project-a"},
				Status:     konfidence.ProjectStatus{Namespace: "kden-p-project-a"},
			}},
			landscapeRepo:        &vectorDeploymentLandscapeRepository{},
			vectorDeploymentRepo: &vectorDeploymentRepository{err: errors.New("cache failure")},
		}
		response, err := h.ListVectorDeploymentsV1(vectorDeploymentContext("project-a"),
			openapi.ListVectorDeploymentsV1RequestObject{ProjectId: "project-a"})
		assertVectorDeploymentAPIError(t, response, err, http.StatusInternalServerError)
	})

	t.Run("filtered landscape not found", func(t *testing.T) {
		landscapeID := "missing"
		h := &projectHandler{
			projectRepo: &vectorDeploymentProjectRepository{project: &konfidence.Project{
				ObjectMeta: metav1.ObjectMeta{Name: "project-a"},
				Status:     konfidence.ProjectStatus{Namespace: "kden-p-project-a"},
			}},
			landscapeRepo: &vectorDeploymentLandscapeRepository{err: landscapedomain.ErrLandscapeNotFound},
		}
		response, err := h.ListVectorDeploymentsV1(vectorDeploymentContext("project-a"),
			openapi.ListVectorDeploymentsV1RequestObject{
				ProjectId: "project-a",
				Params:    openapi.ListVectorDeploymentsV1Params{LandscapeId: &landscapeID},
			})
		assertVectorDeploymentAPIError(t, response, err, http.StatusNotFound)
	})

	t.Run("invalid vector reference", func(t *testing.T) {
		h := &projectHandler{
			projectRepo: &vectorDeploymentProjectRepository{project: &konfidence.Project{
				ObjectMeta: metav1.ObjectMeta{Name: "project-a"},
				Status:     konfidence.ProjectStatus{Namespace: "kden-p-project-a"},
			}},
			landscapeRepo: &vectorDeploymentLandscapeRepository{},
			vectorDeploymentRepo: &vectorDeploymentRepository{items: []vectordeployment.ResolvedVectorDeployment{{
				VectorDeployment: konfidence.VectorDeployment{
					ObjectMeta: metav1.ObjectMeta{Name: "invalid"},
					Spec:       konfidence.VectorDeploymentSpec{Vector: "not a vector reference"},
				},
			}}},
		}
		response, err := h.ListVectorDeploymentsV1(vectorDeploymentContext("project-a"),
			openapi.ListVectorDeploymentsV1RequestObject{ProjectId: "project-a"})
		assertVectorDeploymentAPIError(t, response, err, http.StatusInternalServerError)
	})
}
