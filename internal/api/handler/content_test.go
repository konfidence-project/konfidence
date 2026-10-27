package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/auth"
	pkgcontent "github.com/konfidence-project/konfidence/pkg/ocm/content"
	contentmocks "github.com/konfidence-project/konfidence/pkg/ocm/content/mocks"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("GetVectorContentV1", func() {
	var (
		mockCtrl   *gomock.Controller
		mockReader *contentmocks.MockReader
		h          *projectHandler
		ctx        context.Context
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockReader = contentmocks.NewMockReader(mockCtrl)
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h = newProjectHandlerForTest(project)
		h.ocmReader = mockReader
		ctx = ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("returns VectorContentList with one entry per ref", func() {
		mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(pkgcontent.VectorContent{
			Components: []pkgcontent.VectorComponent{
				{ComponentName: "github.com/example/svc-a", Version: "1.0.0"},
			},
		}, nil)

		resp, err := h.GetVectorContentV1(ctx, openapi.GetVectorContentV1RequestObject{
			ProjectId: "my-project",
			Params:    openapi.GetVectorContentV1Params{Refs: []string{"registry.example.com//github.com/example/my-vector:1.0.0"}},
		})

		Expect(err).NotTo(HaveOccurred())
		ok := resp.(openapi.GetVectorContentV1200JSONResponse)
		Expect(ok.Data).To(HaveLen(1))
		Expect(ok.Data[0].Ref).To(Equal("registry.example.com//github.com/example/my-vector:1.0.0"))
		Expect(ok.Data[0].Components).To(HaveLen(1))
		Expect(ok.Data[0].Components[0].ComponentName).To(Equal("github.com/example/svc-a"))
	})

	It("returns 400 for a malformed ref", func() {
		resp, err := h.GetVectorContentV1(ctx, openapi.GetVectorContentV1RequestObject{
			ProjectId: "my-project",
			Params:    openapi.GetVectorContentV1Params{Refs: []string{"not-a-valid-ref"}},
		})
		expectAPIError(resp, err, http.StatusBadRequest)
	})

	It("returns 404 when the OCM client reports not found", func() {
		mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(pkgcontent.VectorContent{}, pkgocm.ErrNotFound)

		resp, err := h.GetVectorContentV1(ctx, openapi.GetVectorContentV1RequestObject{
			ProjectId: "my-project",
			Params:    openapi.GetVectorContentV1Params{Refs: []string{"registry.example.com//github.com/example/my-vector:1.0.0"}},
		})
		expectAPIError(resp, err, http.StatusNotFound)
	})

	It("returns 500 when the OCM client fails", func() {
		mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(pkgcontent.VectorContent{}, errors.New("registry unavailable"))

		resp, err := h.GetVectorContentV1(ctx, openapi.GetVectorContentV1RequestObject{
			ProjectId: "my-project",
			Params:    openapi.GetVectorContentV1Params{Refs: []string{"registry.example.com//github.com/example/my-vector:1.0.0"}},
		})
		expectAPIError(resp, err, http.StatusInternalServerError)
	})

	It("returns 401 when no session is present", func() {
		resp, err := h.GetVectorContentV1(context.Background(), openapi.GetVectorContentV1RequestObject{
			ProjectId: "my-project",
			Params:    openapi.GetVectorContentV1Params{Refs: []string{"registry.example.com//github.com/example/my-vector:1.0.0"}},
		})
		expectAPIError(resp, err, http.StatusUnauthorized)
	})

	It("returns 404 when project does not exist", func() {
		h2 := newProjectHandlerForTest()
		h2.ocmReader = mockReader

		resp, err := h2.GetVectorContentV1(ctx, openapi.GetVectorContentV1RequestObject{
			ProjectId: "my-project",
			Params:    openapi.GetVectorContentV1Params{Refs: []string{"registry.example.com//github.com/example/my-vector:1.0.0"}},
		})
		expectAPIError(resp, err, http.StatusNotFound)
	})
})

var _ = Describe("GetArtifactContentV1", func() {
	var (
		mockCtrl   *gomock.Controller
		mockReader *contentmocks.MockReader
		h          *projectHandler
		ctx        context.Context
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockReader = contentmocks.NewMockReader(mockCtrl)
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h = newProjectHandlerForTest(project)
		h.ocmReader = mockReader
		ctx = ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("returns ArtifactContent with mapped fields", func() {
		mockReader.EXPECT().GetArtifactContent(gomock.Any(), gomock.Any()).Return(pkgcontent.ArtifactContent{
			Type:       "cloud.konfidence.flux.helm",
			AllowReuse: true,
			Tasks: []pkgcontent.ArtifactTask{
				{Name: "deploy", Type: "flux-helm-release", DependsOn: []string{"pre-deploy"}},
			},
			Resources: []pkgcontent.ArtifactResource{
				{Name: "helm-chart", Type: "helmChart"},
			},
		}, nil)

		resp, err := h.GetArtifactContentV1(ctx, openapi.GetArtifactContentV1RequestObject{
			ProjectId: "my-project",
			Params:    openapi.GetArtifactContentV1Params{Ref: "registry.example.com//github.com/example/my-artifact:2.0.0"},
		})

		Expect(err).NotTo(HaveOccurred())
		ok := resp.(openapi.GetArtifactContentV1200JSONResponse)
		Expect(ok.Ref).To(Equal("registry.example.com//github.com/example/my-artifact:2.0.0"))
		Expect(ok.Type).To(Equal("cloud.konfidence.flux.helm"))
		Expect(ok.AllowReuse).To(BeTrue())
		Expect(ok.Tasks).NotTo(BeNil())
		Expect(*ok.Tasks).To(HaveLen(1))
		Expect((*ok.Tasks)[0].Name).To(Equal("deploy"))
		Expect(ok.Resources).NotTo(BeNil())
		Expect(*ok.Resources).To(HaveLen(1))
		Expect((*ok.Resources)[0].Name).To(Equal("helm-chart"))
	})

	It("returns 400 for a malformed ref", func() {
		resp, err := h.GetArtifactContentV1(ctx, openapi.GetArtifactContentV1RequestObject{
			ProjectId: "my-project",
			Params:    openapi.GetArtifactContentV1Params{Ref: "not-a-valid-ref"},
		})
		expectAPIError(resp, err, http.StatusBadRequest)
	})

	It("returns 404 when the OCM client reports not found", func() {
		mockReader.EXPECT().GetArtifactContent(gomock.Any(), gomock.Any()).Return(pkgcontent.ArtifactContent{}, pkgocm.ErrNotFound)

		resp, err := h.GetArtifactContentV1(ctx, openapi.GetArtifactContentV1RequestObject{
			ProjectId: "my-project",
			Params:    openapi.GetArtifactContentV1Params{Ref: "registry.example.com//github.com/example/my-artifact:2.0.0"},
		})
		expectAPIError(resp, err, http.StatusNotFound)
	})

	It("returns 500 when the OCM client fails", func() {
		mockReader.EXPECT().GetArtifactContent(gomock.Any(), gomock.Any()).Return(pkgcontent.ArtifactContent{}, errors.New("registry unavailable"))

		resp, err := h.GetArtifactContentV1(ctx, openapi.GetArtifactContentV1RequestObject{
			ProjectId: "my-project",
			Params:    openapi.GetArtifactContentV1Params{Ref: "registry.example.com//github.com/example/my-artifact:2.0.0"},
		})
		expectAPIError(resp, err, http.StatusInternalServerError)
	})

	It("returns 401 when no session is present", func() {
		resp, err := h.GetArtifactContentV1(context.Background(), openapi.GetArtifactContentV1RequestObject{
			ProjectId: "my-project",
			Params:    openapi.GetArtifactContentV1Params{Ref: "registry.example.com//github.com/example/my-artifact:2.0.0"},
		})
		expectAPIError(resp, err, http.StatusUnauthorized)
	})
})

var _ = Describe("GetVectorDiffV1", func() {
	var (
		mockCtrl   *gomock.Controller
		mockReader *contentmocks.MockReader
		h          *projectHandler
		ctx        context.Context
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockReader = contentmocks.NewMockReader(mockCtrl)
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h = newProjectHandlerForTest(project)
		h.ocmReader = mockReader
		ctx = ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("returns VectorDiff with incoming and current populated", func() {
		incomingContent := pkgcontent.VectorContent{
			Components: []pkgcontent.VectorComponent{{ComponentName: "github.com/example/svc-a", Version: "2.0.0"}},
		}
		currentContent := pkgcontent.VectorContent{
			Components: []pkgcontent.VectorComponent{{ComponentName: "github.com/example/svc-a", Version: "1.0.0"}},
		}
		gomock.InOrder(
			mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(incomingContent, nil),
			mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(currentContent, nil),
		)

		resp, err := h.GetVectorDiffV1(ctx, openapi.GetVectorDiffV1RequestObject{
			ProjectId: "my-project",
			Params: openapi.GetVectorDiffV1Params{
				Incoming: "registry.example.com//github.com/example/my-vector:2.0.0",
				Current:  "registry.example.com//github.com/example/my-vector:1.0.0",
			},
		})

		Expect(err).NotTo(HaveOccurred())
		ok := resp.(openapi.GetVectorDiffV1200JSONResponse)
		Expect(ok.Incoming.Ref).To(Equal("registry.example.com//github.com/example/my-vector:2.0.0"))
		Expect(ok.Incoming.Components[0].Version).To(Equal("2.0.0"))
		Expect(ok.Current.Ref).To(Equal("registry.example.com//github.com/example/my-vector:1.0.0"))
		Expect(ok.Current.Components[0].Version).To(Equal("1.0.0"))
	})

	It("returns 400 for a malformed incoming ref", func() {
		resp, err := h.GetVectorDiffV1(ctx, openapi.GetVectorDiffV1RequestObject{
			ProjectId: "my-project",
			Params: openapi.GetVectorDiffV1Params{
				Incoming: "not-a-valid-ref",
				Current:  "registry.example.com//github.com/example/my-vector:1.0.0",
			},
		})
		expectAPIError(resp, err, http.StatusBadRequest)
	})

	It("returns 400 for a malformed current ref", func() {
		resp, err := h.GetVectorDiffV1(ctx, openapi.GetVectorDiffV1RequestObject{
			ProjectId: "my-project",
			Params: openapi.GetVectorDiffV1Params{
				Incoming: "registry.example.com//github.com/example/my-vector:2.0.0",
				Current:  "not-a-valid-ref",
			},
		})
		expectAPIError(resp, err, http.StatusBadRequest)
	})

	It("returns 404 when incoming vector is not found", func() {
		mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(pkgcontent.VectorContent{}, pkgocm.ErrNotFound)

		resp, err := h.GetVectorDiffV1(ctx, openapi.GetVectorDiffV1RequestObject{
			ProjectId: "my-project",
			Params: openapi.GetVectorDiffV1Params{
				Incoming: "registry.example.com//github.com/example/my-vector:2.0.0",
				Current:  "registry.example.com//github.com/example/my-vector:1.0.0",
			},
		})
		expectAPIError(resp, err, http.StatusNotFound)
	})

	It("returns 404 when current vector is not found", func() {
		mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(pkgcontent.VectorContent{
			Components: []pkgcontent.VectorComponent{{ComponentName: "github.com/example/svc-a", Version: "2.0.0"}},
		}, nil)
		mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(pkgcontent.VectorContent{}, pkgocm.ErrNotFound)

		resp, err := h.GetVectorDiffV1(ctx, openapi.GetVectorDiffV1RequestObject{
			ProjectId: "my-project",
			Params: openapi.GetVectorDiffV1Params{
				Incoming: "registry.example.com//github.com/example/my-vector:2.0.0",
				Current:  "registry.example.com//github.com/example/my-vector:1.0.0",
			},
		})
		expectAPIError(resp, err, http.StatusNotFound)
	})

	It("returns 401 when no session is present", func() {
		resp, err := h.GetVectorDiffV1(context.Background(), openapi.GetVectorDiffV1RequestObject{
			ProjectId: "my-project",
			Params: openapi.GetVectorDiffV1Params{
				Incoming: "registry.example.com//github.com/example/my-vector:2.0.0",
				Current:  "registry.example.com//github.com/example/my-vector:1.0.0",
			},
		})
		expectAPIError(resp, err, http.StatusUnauthorized)
	})
})
