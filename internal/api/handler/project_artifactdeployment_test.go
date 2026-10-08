package handler

import (
	"context"
	"errors"
	"net/http"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	"github.com/konfidence-project/konfidence/internal/artifactdeployment"
	"github.com/konfidence-project/konfidence/internal/auth"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type adRepository struct {
	items []artifactdeployment.ResolvedArtifactDeployment
	err   error
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

func assertADAPIError(response any, err error, status int) {
	GinkgoHelper()
	Expect(response).To(BeNil())
	apiErr := apierror.As(err)
	Expect(apiErr).To(HaveOccurred())
	Expect(apiErr.Status).To(Equal(status))
}

var _ = Describe("ListArtifactDeploymentsV1", func() {
	It("returns deployments for a filtered landscape", func() {
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
		_ = toArtifactDeploymentResponse(repository.items[0])
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
		Expect(err).NotTo(HaveOccurred())
		data := response.(openapi.ListArtifactDeploymentsV1200JSONResponse).Data
		Expect(data).To(HaveLen(1))
		got := data[0]
		Expect(got.Id).To(Equal("myapp-artifact"))
		Expect(got.LandscapeId).To(Equal(landscapeName))
		Expect(got.StageIds).To(Equal([]string{"prod"}))
		Expect(got.VectorDeploymentIds).To(Equal([]string{"vd-a"}))
		Expect(got.Artifact.ComponentName).To(Equal("acme.example/myapp"))
		Expect(got.Artifact.ComponentVersion).To(Equal("1.0.0"))
	})

	It("returns deployments without a landscape filter", func() {
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

		Expect(err).NotTo(HaveOccurred())
		_, ok := response.(openapi.ListArtifactDeploymentsV1200JSONResponse)
		Expect(ok).To(BeTrue())
	})

	Context("when an error occurs", func() {
		It("returns unauthorized without an identity", func() {
			h := &projectHandler{}
			response, err := h.ListArtifactDeploymentsV1(context.Background(),
				openapi.ListArtifactDeploymentsV1RequestObject{})
			assertADAPIError(response, err, http.StatusUnauthorized)
		})

		It("returns forbidden without access to the project", func() {
			h := &projectHandler{}
			response, err := h.ListArtifactDeploymentsV1(adContext("other-project"),
				openapi.ListArtifactDeploymentsV1RequestObject{
					ProjectId: "project-a",
				})
			assertADAPIError(response, err, http.StatusForbidden)
		})

		It("returns an internal server error on repository failure", func() {
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
			assertADAPIError(response, err, http.StatusInternalServerError)
		})
	})
})
