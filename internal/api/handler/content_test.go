package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/auth"
	kompref "github.com/konfidence-project/konfidence/pkg/ocm/compref"
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
		h = newProjectHandlerWithReader(mockReader, project)
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
		mockReader.EXPECT().GetVectorContent(gomock.Any(), "not-a-valid-ref").
			Return(pkgcontent.VectorContent{}, fmt.Errorf("%w", kompref.ErrInvalidComponentReference))

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
		h = newProjectHandlerWithReader(mockReader, project)
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
		mockReader.EXPECT().GetArtifactContent(gomock.Any(), "not-a-valid-ref").
			Return(pkgcontent.ArtifactContent{}, fmt.Errorf("%w", kompref.ErrInvalidComponentReference))

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

	leftContent := pkgcontent.VectorContent{
		Components: []pkgcontent.VectorComponent{
			{ComponentName: "github.com/example/svc-a", Version: "2.0.0"},
			{ComponentName: "github.com/example/svc-b", Version: "1.0.0"},
		},
	}
	rightContent := pkgcontent.VectorContent{
		Components: []pkgcontent.VectorComponent{
			{ComponentName: "github.com/example/svc-a", Version: "1.0.0"},
			{ComponentName: "github.com/example/svc-b", Version: "1.0.0"},
		},
	}

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockReader = contentmocks.NewMockReader(mockCtrl)
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h = newProjectHandlerWithReader(mockReader, project)
		ctx = ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("returns VectorDiff with computed component diff", func() {
		leftContent := pkgcontent.VectorContent{
			Components: []pkgcontent.VectorComponent{{ComponentName: "github.com/example/svc-a", Version: "2.0.0"}},
		}
		rightContent := pkgcontent.VectorContent{
			Components: []pkgcontent.VectorComponent{{ComponentName: "github.com/example/svc-a", Version: "1.0.0"}},
		}
		gomock.InOrder(
			mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(leftContent, nil),
			mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(rightContent, nil),
		)

		resp, err := h.GetVectorDiffV1(ctx, openapi.GetVectorDiffV1RequestObject{
			ProjectId: "my-project",
			Params: openapi.GetVectorDiffV1Params{
				Left:  "registry.example.com//github.com/example/my-vector:2.0.0",
				Right: "registry.example.com//github.com/example/my-vector:1.0.0",
			},
		})

		Expect(err).NotTo(HaveOccurred())
		ok := resp.(openapi.GetVectorDiffV1200JSONResponse)
		Expect(ok.Left).To(Equal("registry.example.com//github.com/example/my-vector:2.0.0"))
		Expect(ok.Right).To(Equal("registry.example.com//github.com/example/my-vector:1.0.0"))
		Expect(ok.Components).To(HaveLen(1))
		Expect(ok.Components[0].ComponentName).To(Equal("github.com/example/svc-a"))
		Expect(ok.Components[0].ChangeType).To(Equal(openapi.ChangeTypeChanged))
		Expect(*ok.Components[0].LeftVersion).To(Equal("2.0.0"))
		Expect(*ok.Components[0].RightVersion).To(Equal("1.0.0"))
	})

	It("includes unchanged components when includeUnchanged is tru", func() {
		includeUnchanged := openapi.GetVectorDiffV1ParamsIncludeUnchanged("true")
		gomock.InOrder(
			mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(leftContent, nil),
			mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(rightContent, nil),
		)

		resp, err := h.GetVectorDiffV1(ctx, openapi.GetVectorDiffV1RequestObject{
			ProjectId: "my-project",
			Params: openapi.GetVectorDiffV1Params{
				Left:             "registry.example.com//github.com/example/my-vector:2.0.0",
				Right:            "registry.example.com//github.com/example/my-vector:1.0.0",
				IncludeUnchanged: &includeUnchanged,
			},
		})

		Expect(err).NotTo(HaveOccurred())
		ok := resp.(openapi.GetVectorDiffV1200JSONResponse)
		Expect(ok.Components).To(HaveLen(2))
		Expect(ok.Components[0].ChangeType).To(Equal(openapi.ChangeTypeChanged))
		Expect(ok.Components[1].ChangeType).To(Equal(openapi.ChangeTypeUnchanged))
	})

	It("omits unchanged components when includeUnchanged is false", func() {
		gomock.InOrder(
			mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(leftContent, nil),
			mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(rightContent, nil),
		)

		resp, err := h.GetVectorDiffV1(ctx, openapi.GetVectorDiffV1RequestObject{
			ProjectId: "my-project",
			Params: openapi.GetVectorDiffV1Params{
				Left:  "registry.example.com//github.com/example/my-vector:2.0.0",
				Right: "registry.example.com//github.com/example/my-vector:1.0.0",
			},
		})

		Expect(err).NotTo(HaveOccurred())
		ok := resp.(openapi.GetVectorDiffV1200JSONResponse)
		Expect(ok.Components).To(HaveLen(1))
		Expect(ok.Components[0].ComponentName).To(Equal("github.com/example/svc-a"))
		Expect(ok.Components[0].ChangeType).To(Equal(openapi.ChangeTypeChanged))
	})

	It("returns empty non-null slices when vectors have no components or config", func() {
		gomock.InOrder(
			mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(pkgcontent.VectorContent{}, nil),
			mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(pkgcontent.VectorContent{}, nil),
		)

		resp, err := h.GetVectorDiffV1(ctx, openapi.GetVectorDiffV1RequestObject{
			ProjectId: "my-project",
			Params: openapi.GetVectorDiffV1Params{
				Left:  "registry.example.com//github.com/example/my-vector:2.0.0",
				Right: "registry.example.com//github.com/example/my-vector:1.0.0",
			},
		})

		Expect(err).NotTo(HaveOccurred())
		ok := resp.(openapi.GetVectorDiffV1200JSONResponse)
		Expect(ok.Components).NotTo(BeNil())
		Expect(ok.Components).To(BeEmpty())
		Expect(ok.Config).NotTo(BeNil())
		Expect(*ok.Config.Features).NotTo(BeNil())
		Expect(*ok.Config.Features).To(BeEmpty())
		Expect(*ok.Config.Authored).NotTo(BeNil())
		Expect(*ok.Config.Authored).To(BeEmpty())
	})

	It("returns correct config diff with changed, unchanged, removed, and added entries", func() {
		includeUnchanged := openapi.GetVectorDiffV1ParamsIncludeUnchanged("true")
		leftContent := pkgcontent.VectorContent{
			Components: []pkgcontent.VectorComponent{},
			Config: &pkgcontent.VectorConfig{
				Features: map[string]any{
					"darkMode":   false,
					"betaLayout": true,
					"oldFeature": true,
				},
				Authored: map[string]any{
					"replicaCount": 3,
				},
			},
		}
		rightContent := pkgcontent.VectorContent{
			Components: []pkgcontent.VectorComponent{},
			Config: &pkgcontent.VectorConfig{
				Features: map[string]any{
					"darkMode":   true,
					"betaLayout": true,
					"newFeature": false,
				},
				Authored: map[string]any{
					"replicaCount": 3,
				},
			},
		}
		gomock.InOrder(
			mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(leftContent, nil),
			mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(rightContent, nil),
		)

		resp, err := h.GetVectorDiffV1(ctx, openapi.GetVectorDiffV1RequestObject{
			ProjectId: "my-project",
			Params: openapi.GetVectorDiffV1Params{
				Left:             "registry.example.com//github.com/example/my-vector:2.0.0",
				Right:            "registry.example.com//github.com/example/my-vector:1.0.0",
				IncludeUnchanged: &includeUnchanged,
			},
		})

		Expect(err).NotTo(HaveOccurred())
		ok := resp.(openapi.GetVectorDiffV1200JSONResponse)
		Expect(ok.Config).NotTo(BeNil())

		features := *ok.Config.Features
		findFeature := func(key string) openapi.FeatureFlagEntryDiff {
			for _, e := range features {
				if e.Key == key {
					return e
				}
			}
			Fail("feature key not found: " + key)
			return openapi.FeatureFlagEntryDiff{}
		}

		Expect(findFeature("darkMode").ChangeType).To(Equal(openapi.FeatureFlagsChangeTypeEnabled))
		Expect(findFeature("betaLayout").ChangeType).To(Equal(openapi.FeatureFlagsChangeTypeUnchanged))
		Expect(findFeature("oldFeature").ChangeType).To(Equal(openapi.FeatureFlagsChangeTypeRemoved))
		Expect(findFeature("newFeature").ChangeType).To(Equal(openapi.FeatureFlagsChangeTypeAdded))

		authored := *ok.Config.Authored
		Expect(authored).To(HaveLen(1))
		Expect(authored[0].Key).To(Equal("replicaCount"))
		Expect(authored[0].ChangeType).To(Equal(openapi.ChangeTypeUnchanged))
	})

	It("returns 400 for a malformed left ref", func() {
		mockReader.EXPECT().GetVectorContent(gomock.Any(), "not-a-valid-ref").
			Return(pkgcontent.VectorContent{}, fmt.Errorf("%w", kompref.ErrInvalidComponentReference))

		resp, err := h.GetVectorDiffV1(ctx, openapi.GetVectorDiffV1RequestObject{
			ProjectId: "my-project",
			Params: openapi.GetVectorDiffV1Params{
				Left:  "not-a-valid-ref",
				Right: "registry.example.com//github.com/example/my-vector:1.0.0",
			},
		})
		expectAPIError(resp, err, http.StatusBadRequest)
	})

	It("returns 400 for a malformed right ref", func() {
		mockReader.EXPECT().GetVectorContent(gomock.Any(), "registry.example.com//github.com/example/my-vector:2.0.0").
			Return(pkgcontent.VectorContent{
				Components: []pkgcontent.VectorComponent{{ComponentName: "github.com/example/svc-a", Version: "2.0.0"}},
			}, nil)
		mockReader.EXPECT().GetVectorContent(gomock.Any(), "not-a-valid-ref").
			Return(pkgcontent.VectorContent{}, fmt.Errorf("%w", kompref.ErrInvalidComponentReference))

		resp, err := h.GetVectorDiffV1(ctx, openapi.GetVectorDiffV1RequestObject{
			ProjectId: "my-project",
			Params: openapi.GetVectorDiffV1Params{
				Left:  "registry.example.com//github.com/example/my-vector:2.0.0",
				Right: "not-a-valid-ref",
			},
		})
		expectAPIError(resp, err, http.StatusBadRequest)
	})

	It("returns 404 when left vector is not found", func() {
		mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(pkgcontent.VectorContent{}, pkgocm.ErrNotFound)

		resp, err := h.GetVectorDiffV1(ctx, openapi.GetVectorDiffV1RequestObject{
			ProjectId: "my-project",
			Params: openapi.GetVectorDiffV1Params{
				Left:  "registry.example.com//github.com/example/my-vector:2.0.0",
				Right: "registry.example.com//github.com/example/my-vector:1.0.0",
			},
		})
		expectAPIError(resp, err, http.StatusNotFound)
	})

	It("returns 404 when right vector is not found", func() {
		mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(pkgcontent.VectorContent{
			Components: []pkgcontent.VectorComponent{{ComponentName: "github.com/example/svc-a", Version: "2.0.0"}},
		}, nil)
		mockReader.EXPECT().GetVectorContent(gomock.Any(), gomock.Any()).Return(pkgcontent.VectorContent{}, pkgocm.ErrNotFound)

		resp, err := h.GetVectorDiffV1(ctx, openapi.GetVectorDiffV1RequestObject{
			ProjectId: "my-project",
			Params: openapi.GetVectorDiffV1Params{
				Left:  "registry.example.com//github.com/example/my-vector:2.0.0",
				Right: "registry.example.com//github.com/example/my-vector:1.0.0",
			},
		})
		expectAPIError(resp, err, http.StatusNotFound)
	})

	It("returns 401 when no session is present", func() {
		resp, err := h.GetVectorDiffV1(context.Background(), openapi.GetVectorDiffV1RequestObject{
			ProjectId: "my-project",
			Params: openapi.GetVectorDiffV1Params{
				Left:  "registry.example.com//github.com/example/my-vector:2.0.0",
				Right: "registry.example.com//github.com/example/my-vector:1.0.0",
			},
		})
		expectAPIError(resp, err, http.StatusUnauthorized)
	})
})
