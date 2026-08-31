package handler

import (
	"context"
	"net/http"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/auth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func vectorPromotionConfigFixture(name, namespace string) *konfidence.VectorPromotionConfig {
	keepLast := int32(5)
	return &konfidence.VectorPromotionConfig{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: konfidence.VectorPromotionConfigSpec{
			Source:             konfidence.PromotionSourceReference{Kind: "Stage", Name: "src", Landscape: "dev"},
			Target:             konfidence.PromotionTargetReference{Kind: "Stage", Name: "dst", Landscape: "prod"},
			TTLAfterFinished:   &metav1.Duration{Duration: time.Hour},
			KeepLastPromotions: &keepLast,
		},
	}
}

func vectorPromotionFixture(name, namespace, configName string, sequence int64) *konfidence.VectorPromotion {
	return &konfidence.VectorPromotion{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: konfidence.VectorPromotionSpec{
			VectorPromotionConfigName: configName,
			Source:                    konfidence.PromotionSourceReference{Kind: "Stage", Name: "src", Landscape: "dev"},
			Target:                    konfidence.PromotionTargetReference{Kind: "Stage", Name: "dst", Landscape: "prod"},
			Vector:                    "registry//component:v1",
			Sequence:                  sequence,
		},
	}
}

var _ = Describe("GetVectorPromotionConfigV1", func() {
	It("returns the config with its mapped fields and aggregated promotions", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		config := vectorPromotionConfigFixture("cfg", "kden-p-my-project")
		p1 := vectorPromotionFixture("cfg-2", "kden-p-my-project", "cfg", 2)
		p2 := vectorPromotionFixture("cfg-1", "kden-p-my-project", "cfg", 1)
		other := vectorPromotionFixture("elsewhere-1", "kden-p-my-project", "other-cfg", 1)
		h := newProjectHandlerForTest(project, config, p1, p2, other)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.GetVectorPromotionConfigV1(ctx, openapi.GetVectorPromotionConfigV1RequestObject{
			ProjectId:               "my-project",
			VectorPromotionConfigId: "cfg",
		})
		Expect(err).NotTo(HaveOccurred())

		ok, is200 := resp.(openapi.GetVectorPromotionConfigV1200JSONResponse)
		Expect(is200).To(BeTrue())
		Expect(ok.Id).To(Equal("cfg"))
		Expect(ok.Source.Name).To(Equal("src"))
		Expect(ok.Target.Name).To(Equal("dst"))
		Expect(ok.TtlAfterFinished).NotTo(BeNil())
		Expect(*ok.TtlAfterFinished).To(Equal("1h0m0s"))
		Expect(ok.KeepLastPromotions).NotTo(BeNil())
		Expect(*ok.KeepLastPromotions).To(Equal(5))

		// Only this config's promotions, ordered by sequence.
		Expect(ok.Promotions).To(HaveLen(2))
		Expect(ok.Promotions[0].Id).To(Equal("cfg-1"))
		Expect(ok.Promotions[1].Id).To(Equal("cfg-2"))
	})

	It("returns an empty promotions list when the config has none", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		config := vectorPromotionConfigFixture("cfg", "kden-p-my-project")
		h := newProjectHandlerForTest(project, config)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.GetVectorPromotionConfigV1(ctx, openapi.GetVectorPromotionConfigV1RequestObject{
			ProjectId:               "my-project",
			VectorPromotionConfigId: "cfg",
		})
		Expect(err).NotTo(HaveOccurred())

		ok := resp.(openapi.GetVectorPromotionConfigV1200JSONResponse)
		Expect(ok.Promotions).NotTo(BeNil())
		Expect(ok.Promotions).To(BeEmpty())
	})

	It("returns 404 when the config does not exist", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h := newProjectHandlerForTest(project)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.GetVectorPromotionConfigV1(ctx, openapi.GetVectorPromotionConfigV1RequestObject{
			ProjectId:               "my-project",
			VectorPromotionConfigId: "missing",
		})

		expectAPIError(resp, err, http.StatusNotFound)
	})

	It("returns 404 when the project does not exist", func() {
		h := newProjectHandlerForTest()

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"nonexistent": {"admin"}})
		resp, err := h.GetVectorPromotionConfigV1(ctx, openapi.GetVectorPromotionConfigV1RequestObject{
			ProjectId:               "nonexistent",
			VectorPromotionConfigId: "cfg",
		})

		expectAPIError(resp, err, http.StatusNotFound)
	})

	It("returns 403 when the caller has no role for the project", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h := newProjectHandlerForTest(project)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"other-project": {"admin"}})
		resp, err := h.GetVectorPromotionConfigV1(ctx, openapi.GetVectorPromotionConfigV1RequestObject{
			ProjectId:               "my-project",
			VectorPromotionConfigId: "cfg",
		})

		expectAPIError(resp, err, http.StatusForbidden)
	})

	It("returns 500 when the resolved project has no namespace", func() {
		project := landscapeProjectFixture("my-project", "")
		h := newProjectHandlerForTest(project)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.GetVectorPromotionConfigV1(ctx, openapi.GetVectorPromotionConfigV1RequestObject{
			ProjectId:               "my-project",
			VectorPromotionConfigId: "cfg",
		})

		expectAPIError(resp, err, http.StatusInternalServerError)
	})

	It("returns 401 when no session is present", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h := newProjectHandlerForTest(project)

		resp, err := h.GetVectorPromotionConfigV1(context.Background(), openapi.GetVectorPromotionConfigV1RequestObject{
			ProjectId:               "my-project",
			VectorPromotionConfigId: "cfg",
		})

		expectAPIError(resp, err, http.StatusUnauthorized)
	})
})

var _ = Describe("ListVectorPromotionConfigsV1", func() {
	It("returns all configs with their mapped fields and aggregated promotions", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		cfg1 := vectorPromotionConfigFixture("cfg-a", "kden-p-my-project")
		cfg2 := vectorPromotionConfigFixture("cfg-b", "kden-p-my-project")
		p1 := vectorPromotionFixture("cfg-a-1", "kden-p-my-project", "cfg-a", 1)
		p2 := vectorPromotionFixture("cfg-a-2", "kden-p-my-project", "cfg-a", 2)
		p3 := vectorPromotionFixture("cfg-b-1", "kden-p-my-project", "cfg-b", 1)
		h := newProjectHandlerForTest(project, cfg1, cfg2, p1, p2, p3)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ListVectorPromotionConfigsV1(ctx, openapi.ListVectorPromotionConfigsV1RequestObject{
			ProjectId: "my-project",
		})
		Expect(err).NotTo(HaveOccurred())

		ok, is200 := resp.(openapi.ListVectorPromotionConfigsV1200JSONResponse)
		Expect(is200).To(BeTrue())
		Expect(ok.Data).To(HaveLen(2))

		ids := []string{ok.Data[0].Id, ok.Data[1].Id}
		Expect(ids).To(ConsistOf("cfg-a", "cfg-b"))

		// Verify promotions are populated per config.
		for _, cfg := range ok.Data {
			switch cfg.Id {
			case "cfg-a":
				Expect(cfg.Promotions).To(HaveLen(2))
				Expect(cfg.Promotions[0].Id).To(Equal("cfg-a-1"))
				Expect(cfg.Promotions[1].Id).To(Equal("cfg-a-2"))
			case "cfg-b":
				Expect(cfg.Promotions).To(HaveLen(1))
				Expect(cfg.Promotions[0].Id).To(Equal("cfg-b-1"))
			}
		}
	})

	It("maps config fields correctly", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		cfg := vectorPromotionConfigFixture("cfg", "kden-p-my-project")
		h := newProjectHandlerForTest(project, cfg)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ListVectorPromotionConfigsV1(ctx, openapi.ListVectorPromotionConfigsV1RequestObject{
			ProjectId: "my-project",
		})
		Expect(err).NotTo(HaveOccurred())

		ok := resp.(openapi.ListVectorPromotionConfigsV1200JSONResponse)
		Expect(ok.Data).To(HaveLen(1))
		item := ok.Data[0]
		Expect(item.Id).To(Equal("cfg"))
		Expect(item.Source.Name).To(Equal("src"))
		Expect(item.Target.Name).To(Equal("dst"))
		Expect(item.TtlAfterFinished).NotTo(BeNil())
		Expect(*item.TtlAfterFinished).To(Equal("1h0m0s"))
		Expect(item.KeepLastPromotions).NotTo(BeNil())
		Expect(*item.KeepLastPromotions).To(Equal(5))
		Expect(item.Promotions).To(BeEmpty())
	})

	It("returns an empty list when the project has no configs", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h := newProjectHandlerForTest(project)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ListVectorPromotionConfigsV1(ctx, openapi.ListVectorPromotionConfigsV1RequestObject{
			ProjectId: "my-project",
		})
		Expect(err).NotTo(HaveOccurred())

		ok := resp.(openapi.ListVectorPromotionConfigsV1200JSONResponse)
		Expect(ok.Data).To(BeEmpty())
	})

	It("only returns configs belonging to the requested project's namespace", func() {
		projectA := landscapeProjectFixture("project-a", "kden-p-project-a")
		cfgA := vectorPromotionConfigFixture("cfg-a", "kden-p-project-a")
		cfgB := vectorPromotionConfigFixture("cfg-b", "kden-p-project-b")
		h := newProjectHandlerForTest(projectA, cfgA, cfgB)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"project-a": {"admin"}})
		resp, err := h.ListVectorPromotionConfigsV1(ctx, openapi.ListVectorPromotionConfigsV1RequestObject{
			ProjectId: "project-a",
		})
		Expect(err).NotTo(HaveOccurred())

		ok := resp.(openapi.ListVectorPromotionConfigsV1200JSONResponse)
		Expect(ok.Data).To(HaveLen(1))
		Expect(ok.Data[0].Id).To(Equal("cfg-a"))
	})

	It("returns 404 when the project does not exist", func() {
		h := newProjectHandlerForTest()

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"nonexistent": {"admin"}})
		resp, err := h.ListVectorPromotionConfigsV1(ctx, openapi.ListVectorPromotionConfigsV1RequestObject{
			ProjectId: "nonexistent",
		})

		expectAPIError(resp, err, http.StatusNotFound)
	})

	It("returns 403 when the caller has no role for the project", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h := newProjectHandlerForTest(project)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"other-project": {"admin"}})
		resp, err := h.ListVectorPromotionConfigsV1(ctx, openapi.ListVectorPromotionConfigsV1RequestObject{
			ProjectId: "my-project",
		})

		expectAPIError(resp, err, http.StatusForbidden)
	})

	It("returns 500 when the resolved project has no namespace", func() {
		project := landscapeProjectFixture("my-project", "")
		h := newProjectHandlerForTest(project)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ListVectorPromotionConfigsV1(ctx, openapi.ListVectorPromotionConfigsV1RequestObject{
			ProjectId: "my-project",
		})

		expectAPIError(resp, err, http.StatusInternalServerError)
	})

	It("returns 401 when no session is present", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h := newProjectHandlerForTest(project)

		resp, err := h.ListVectorPromotionConfigsV1(context.Background(), openapi.ListVectorPromotionConfigsV1RequestObject{
			ProjectId: "my-project",
		})

		expectAPIError(resp, err, http.StatusUnauthorized)
	})
})
