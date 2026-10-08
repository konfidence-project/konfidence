package handler

import (
	"context"
	"errors"
	"net/http"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	"github.com/konfidence-project/konfidence/internal/auth"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	"github.com/konfidence-project/konfidence/internal/vectordeployment"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

func assertVectorDeploymentAPIError(response any, err error, status int) {
	GinkgoHelper()
	Expect(response).To(BeNil())
	apiErr := apierror.As(err)
	Expect(apiErr).To(HaveOccurred())
	Expect(apiErr.Status).To(Equal(status))
}

var _ = Describe("ListVectorDeploymentsV1", func() {
	It("returns deployments for a filtered landscape", func() {
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
		_, err := toVectorDeploymentResponse(repository.items[0])
		Expect(err).NotTo(HaveOccurred(), "test fixture cannot be mapped")
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
		Expect(err).NotTo(HaveOccurred())
		data := response.(openapi.ListVectorDeploymentsV1200JSONResponse).Data
		Expect(data).To(HaveLen(1))
		got := data[0]
		Expect(got.Id).To(Equal("checkout-v1"))
		Expect(got.LandscapeId).To(Equal(landscapeName))
		Expect(got.StageId).To(Equal("checkout"))
		Expect(got.Vector.Repository).To(Equal("https://registry.example.com/ocm"))
		Expect(got.Vector.ComponentName).To(Equal("acme.example/checkout"))
		Expect(got.Vector.ComponentVersion).To(Equal("1.2.3"))
		Expect(got.Status).To(Equal(openapi.VectorDeploymentStatusDeployingVector))
		Expect(repository.scope).To(HaveLen(1))
		Expect(repository.scope[0].Landscape.Name).To(Equal(landscapeName))
		Expect(landscapeRepository.getCalls).To(BeZero())
		Expect(landscapeRepository.listCalls).To(Equal(1))
	})

	It("returns deployments without a landscape filter", func() {
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
		Expect(err).NotTo(HaveOccurred())
		_, ok := response.(openapi.ListVectorDeploymentsV1200JSONResponse)
		Expect(ok).To(BeTrue())
		Expect(repository.scope).To(HaveLen(2))
		Expect(landscapeRepository.getCalls).To(BeZero())
		Expect(landscapeRepository.listCalls).To(Equal(1))
	})

	Context("when an error occurs", func() {
		It("returns unauthorized without an identity", func() {
			h := &projectHandler{}
			response, err := h.ListVectorDeploymentsV1(context.Background(), openapi.ListVectorDeploymentsV1RequestObject{})
			assertVectorDeploymentAPIError(response, err, http.StatusUnauthorized)
		})

		It("returns forbidden without access to the project", func() {
			h := &projectHandler{}
			response, err := h.ListVectorDeploymentsV1(vectorDeploymentContext("other-project"),
				openapi.ListVectorDeploymentsV1RequestObject{ProjectId: "project-a"})
			assertVectorDeploymentAPIError(response, err, http.StatusForbidden)
		})

		It("returns an internal server error on repository failure", func() {
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
			assertVectorDeploymentAPIError(response, err, http.StatusInternalServerError)
		})

		It("returns not found for a missing filtered landscape", func() {
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
			assertVectorDeploymentAPIError(response, err, http.StatusNotFound)
		})

		It("returns an internal server error for an invalid vector reference", func() {
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
			assertVectorDeploymentAPIError(response, err, http.StatusInternalServerError)
		})
	})
})
