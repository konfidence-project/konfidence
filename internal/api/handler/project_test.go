package handler

import (
	"context"
	"errors"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	"github.com/konfidence-project/konfidence/internal/auth"
	"github.com/konfidence-project/konfidence/internal/project"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

var _ = Describe("ListProjectsV1", func() {
	It("returns only matching projects", func() {
		h := &projectHandler{
			projectRepo: &projectRepository{projects: []konfidence.Project{
				projectFixture("visible", "Visible Project", "platform-engineers"),
				projectFixture("hidden", "Hidden Project", "platform-managers"),
			}},
		}

		response, err := h.ListProjectsV1(authorizedContext(auth.ProjectRoles{
			"visible": {"admin"},
		}), openapi.ListProjectsV1RequestObject{})
		Expect(err).NotTo(HaveOccurred())
		projects := response.(openapi.ListProjectsV1200JSONResponse).Data
		Expect(projects).To(HaveLen(1))
		Expect(projects[0].Id).To(Equal("visible"))
		Expect(projects[0].Name).To(Equal("Visible Project"))
	})

	It("returns an empty list when no project matches", func() {
		h := &projectHandler{
			projectRepo: &projectRepository{projects: []konfidence.Project{
				projectFixture("hidden", "Hidden Project", "platform-engineers"),
			}},
		}

		response, err := h.ListProjectsV1(authorizedContext(auth.ProjectRoles{}), openapi.ListProjectsV1RequestObject{})
		Expect(err).NotTo(HaveOccurred())
		Expect(response.(openapi.ListProjectsV1200JSONResponse).Data).To(BeEmpty())
	})

	It("returns unauthorized without an identity", func() {
		h := &projectHandler{projectRepo: &projectRepository{}}

		response, err := h.ListProjectsV1(context.Background(), openapi.ListProjectsV1RequestObject{})
		Expect(err).NotTo(HaveOccurred())
		unauthorized, ok := response.(openapi.ListProjectsV1401JSONResponse)
		Expect(ok).To(BeTrue())
		Expect(unauthorized.Error.Code).To(Equal("unauthorized"))
		Expect(unauthorized.Error.Message).To(Equal("authentication required or session expired"))
	})

	It("returns 500 on repository error", func() {
		repositoryErr := errors.New("kubernetes unavailable")
		h := &projectHandler{projectRepo: &projectRepository{err: repositoryErr}}

		response, err := h.ListProjectsV1(authorizedContext(auth.ProjectRoles{
			"visible": {"admin"},
		}), openapi.ListProjectsV1RequestObject{})
		Expect(err).NotTo(HaveOccurred())
		internal, ok := response.(openapi.ListProjectsV1500JSONResponse)
		Expect(ok).To(BeTrue())
		Expect(internal.Error.Code).To(Equal("internal_server_error"))
		Expect(internal.Error.Message).To(Equal("an unexpected error occurred"))
	})
})
