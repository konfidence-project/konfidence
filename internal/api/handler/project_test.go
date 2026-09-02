package handler

import (
	"context"
	"errors"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	"github.com/konfidence-project/konfidence/internal/auth"
	"github.com/konfidence-project/konfidence/internal/project"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type projectRepository struct {
	projects []konfidence.Project
	err      error
}

func (r *projectRepository) Get(_ context.Context, _ string) (*konfidence.Project, error) {
	return nil, project.ErrNotFound
}

func (r *projectRepository) List(_ context.Context, projectRoles auth.ProjectRoles) ([]konfidence.Project, error) {
	if r.err != nil {
		return nil, r.err
	}
	result := make([]konfidence.Project, 0, len(projectRoles))
	for _, item := range r.projects {
		if len(projectRoles[item.Name]) > 0 {
			result = append(result, item)
		}
	}
	return result, nil
}

func projectFixture(name, displayName string, groups ...string) konfidence.Project {
	return konfidence.Project{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: konfidence.ProjectSpec{
			DisplayName: displayName,
			RoleBindings: map[string]konfidence.Subjects{
				"admin": {{Session: &konfidence.SessionSubject{MemberOf: groups}}},
			},
		},
	}
}

func authorizedContext(projectRoles auth.ProjectRoles) context.Context {
	return session.NewContext(context.Background(), &session.Session{Context: session.Context{ProjectRoles: projectRoles}})
}

func TestListProjectsV1(t *testing.T) {
	t.Run("returns only matching projects", func(t *testing.T) {
		h := &projectHandler{
			projectRepo: &projectRepository{projects: []konfidence.Project{
				projectFixture("visible", "Visible Project", "platform-engineers"),
				projectFixture("hidden", "Hidden Project", "platform-managers"),
			}},
		}

		response, err := h.ListProjectsV1(authorizedContext(auth.ProjectRoles{
			"visible": {"admin"},
		}), openapi.ListProjectsV1RequestObject{})
		if err != nil {
			t.Fatal(err)
		}
		projects := response.(openapi.ListProjectsV1200JSONResponse).Data
		if len(projects) != 1 || projects[0].Id != "visible" || projects[0].Name != "Visible Project" {
			t.Fatalf("unexpected projects: %#v", projects)
		}
	})

	t.Run("returns an empty list when no project matches", func(t *testing.T) {
		h := &projectHandler{
			projectRepo: &projectRepository{projects: []konfidence.Project{
				projectFixture("hidden", "Hidden Project", "platform-engineers"),
			}},
		}

		response, err := h.ListProjectsV1(authorizedContext(auth.ProjectRoles{}), openapi.ListProjectsV1RequestObject{})
		if err != nil {
			t.Fatal(err)
		}
		if projects := response.(openapi.ListProjectsV1200JSONResponse).Data; len(projects) != 0 {
			t.Fatalf("expected no projects, got %#v", projects)
		}
	})

	t.Run("returns unauthorized without an identity", func(t *testing.T) {
		h := &projectHandler{projectRepo: &projectRepository{}}

		response, err := h.ListProjectsV1(context.Background(), openapi.ListProjectsV1RequestObject{})
		if err != nil {
			t.Fatal(err)
		}
		unauthorized, ok := response.(openapi.ListProjectsV1401JSONResponse)
		if !ok {
			t.Fatalf("expected unauthorized response, got %T", response)
		}
		if unauthorized.Error.Code != "unauthorized" || unauthorized.Error.Message != "authentication required or session expired" {
			t.Fatalf("unexpected unauthorized response: %#v", unauthorized.Error)
		}
	})

	t.Run("returns 500 on repository error", func(t *testing.T) {
		repositoryErr := errors.New("kubernetes unavailable")
		h := &projectHandler{projectRepo: &projectRepository{err: repositoryErr}}

		response, err := h.ListProjectsV1(authorizedContext(auth.ProjectRoles{
			"visible": {"admin"},
		}), openapi.ListProjectsV1RequestObject{})
		if err != nil {
			t.Fatal(err)
		}
		internal, ok := response.(openapi.ListProjectsV1500JSONResponse)
		if !ok {
			t.Fatalf("expected internal error response, got %T", response)
		}
		if internal.Error.Code != "internal_server_error" || internal.Error.Message != "an unexpected error occurred" {
			t.Fatalf("unexpected internal error response: %#v", internal.Error)
		}
	})
}
