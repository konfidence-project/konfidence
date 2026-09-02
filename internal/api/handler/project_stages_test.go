package handler

import (
	"context"
	"net/http"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/auth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// scopedLandscapeFixture builds a landscape in the project namespace whose managed
// namespace is the one stages live in. An empty managed namespace models a landscape
// that is still provisioning.
func scopedLandscapeFixture(name, projectNamespace, managedNamespace string) *konfidence.Landscape {
	return &konfidence.Landscape{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: projectNamespace},
		Status:     konfidence.LandscapeStatus{Namespace: managedNamespace},
	}
}

func stageFixture(name, namespace, vector string, generation int64) *konfidence.Stage {
	return &konfidence.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Generation: generation},
		Spec:       konfidence.StageSpec{Vector: vector},
	}
}

func activeStageVersionFixture(s *konfidence.Stage, versionName string) *konfidence.Stage {
	s.Status.ActiveStageVersion = &konfidence.StageVersionReference{Name: versionName}
	return s
}

func stageVersionFixture(name, namespace, stageName, vector string, stageGeneration int64,
	conditionTypes ...string,
) *konfidence.StageVersion {
	conditions := make([]metav1.Condition, 0, len(conditionTypes))
	for _, conditionType := range conditionTypes {
		conditions = append(conditions, metav1.Condition{
			Type:               conditionType,
			Status:             metav1.ConditionTrue,
			Reason:             "Test",
			LastTransitionTime: metav1.Now(),
		})
	}

	return &konfidence.StageVersion{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: konfidence.StageVersionSpec{
			Vector:          vector,
			StageGeneration: stageGeneration,
			StageRef:        &konfidence.StageReference{Name: stageName},
		},
		Status: konfidence.StageVersionStatus{Conditions: conditions},
	}
}

func landscapeIdParam(id string) openapi.ListStagesV1Params {
	return openapi.ListStagesV1Params{LandscapeId: &id}
}

var _ = Describe("ListStagesV1", func() {
	It("returns the stages of all landscapes of the project", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		dev := scopedLandscapeFixture("dev", "kden-p-my-project", "kden-l-dev")
		prod := scopedLandscapeFixture("prod", "kden-p-my-project", "kden-l-prod")
		h := newProjectHandlerForTest(project, dev, prod,
			stageFixture("alpha", "kden-l-dev", "vector:1", 1),
			stageFixture("beta", "kden-l-prod", "vector:1", 1),
		)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ListStagesV1(ctx, openapi.ListStagesV1RequestObject{ProjectId: "my-project"})
		Expect(err).NotTo(HaveOccurred())

		ok, is200 := resp.(openapi.ListStagesV1200JSONResponse)
		Expect(is200).To(BeTrue())
		Expect(ok.Data).To(HaveLen(2))
		Expect(ok.Data[0].Id).To(Equal("alpha"))
		Expect(ok.Data[0].Name).To(Equal("alpha"))
		Expect(ok.Data[0].LandscapeId).To(Equal("dev"))
		Expect(ok.Data[1].LandscapeId).To(Equal("prod"))
	})

	// Response ordering is pinned in the stage repository suite
	// (TestListForScope_SortsByLandscapeAndName): the fake client already returns
	// landscapes and stages name-sorted per namespace, so a handler-level fixture set
	// cannot distinguish sorted output from unsorted iteration order.

	It("omits both version fields for a stage without stage versions", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		dev := scopedLandscapeFixture("dev", "kden-p-my-project", "kden-l-dev")
		h := newProjectHandlerForTest(project, dev, stageFixture("alpha", "kden-l-dev", "vector:1", 1))

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ListStagesV1(ctx, openapi.ListStagesV1RequestObject{ProjectId: "my-project"})
		Expect(err).NotTo(HaveOccurred())

		item := resp.(openapi.ListStagesV1200JSONResponse).Data[0]
		Expect(item.TargetStageVersion).To(BeNil())
		Expect(item.ActiveStageVersion).To(BeNil())
	})

	It("maps the target and active stage version with derived states", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		dev := scopedLandscapeFixture("dev", "kden-p-my-project", "kden-l-dev")
		stage := activeStageVersionFixture(stageFixture("alpha", "kden-l-dev", "vector:2", 2), "alpha-v1")
		active := stageVersionFixture("alpha-v1", "kden-l-dev", "alpha", "vector:1", 1,
			konfidence.VectorDeploymentCreatedCondition, konfidence.VectorMigratedCondition,
			konfidence.VectorActivationCreatedCondition, konfidence.StageVersionReady)
		target := stageVersionFixture("alpha-v2", "kden-l-dev", "alpha", "vector:2", 2,
			konfidence.VectorDeploymentCreatedCondition)
		h := newProjectHandlerForTest(project, dev, stage, active, target)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ListStagesV1(ctx, openapi.ListStagesV1RequestObject{ProjectId: "my-project"})
		Expect(err).NotTo(HaveOccurred())

		item := resp.(openapi.ListStagesV1200JSONResponse).Data[0]
		Expect(item.TargetStageVersion).NotTo(BeNil())
		Expect(item.TargetStageVersion.Id).To(Equal("alpha-v2"))
		Expect(item.TargetStageVersion.Vector).To(Equal("vector:2"))
		Expect(item.TargetStageVersion.StageGeneration).To(Equal(2))
		Expect(item.TargetStageVersion.Status).To(Equal(openapi.StageVersionStatusDeployingVector))
		Expect(item.ActiveStageVersion).NotTo(BeNil())
		Expect(item.ActiveStageVersion.Id).To(Equal("alpha-v1"))
		Expect(item.ActiveStageVersion.StageGeneration).To(Equal(1))
		Expect(item.ActiveStageVersion.Status).To(Equal(openapi.StageVersionStatusReady))
	})

	It("returns only the stages of the filtered landscape", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		dev := scopedLandscapeFixture("dev", "kden-p-my-project", "kden-l-dev")
		prod := scopedLandscapeFixture("prod", "kden-p-my-project", "kden-l-prod")
		h := newProjectHandlerForTest(project, dev, prod,
			stageFixture("alpha", "kden-l-dev", "vector:1", 1),
			stageFixture("beta", "kden-l-prod", "vector:1", 1),
		)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ListStagesV1(ctx, openapi.ListStagesV1RequestObject{
			ProjectId: "my-project",
			Params:    landscapeIdParam("dev"),
		})
		Expect(err).NotTo(HaveOccurred())

		ok := resp.(openapi.ListStagesV1200JSONResponse)
		Expect(ok.Data).To(HaveLen(1))
		Expect(ok.Data[0].Name).To(Equal("alpha"))
	})

	It("returns 404 when the landscape filter names an unknown landscape", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		dev := scopedLandscapeFixture("dev", "kden-p-my-project", "kden-l-dev")
		h := newProjectHandlerForTest(project, dev)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ListStagesV1(ctx, openapi.ListStagesV1RequestObject{
			ProjectId: "my-project",
			Params:    landscapeIdParam("unknown"),
		})
		expectAPIError(resp, err, http.StatusNotFound)
	})

	It("returns 404 when the landscape filter is supplied but empty", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		dev := scopedLandscapeFixture("dev", "kden-p-my-project", "kden-l-dev")
		h := newProjectHandlerForTest(project, dev, stageFixture("alpha", "kden-l-dev", "vector:1", 1))

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ListStagesV1(ctx, openapi.ListStagesV1RequestObject{
			ProjectId: "my-project",
			Params:    landscapeIdParam(""),
		})
		expectAPIError(resp, err, http.StatusNotFound)
	})

	It("returns an empty list for a landscape that is still provisioning", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		provisioning := scopedLandscapeFixture("dev", "kden-p-my-project", "")
		h := newProjectHandlerForTest(project, provisioning)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ListStagesV1(ctx, openapi.ListStagesV1RequestObject{
			ProjectId: "my-project",
			Params:    landscapeIdParam("dev"),
		})
		Expect(err).NotTo(HaveOccurred())

		ok, is200 := resp.(openapi.ListStagesV1200JSONResponse)
		Expect(is200).To(BeTrue())
		Expect(ok.Data).To(BeEmpty())
	})

	It("returns an empty list when the project has no landscapes", func() {
		project := landscapeProjectFixture("empty-project", "kden-p-empty-project")
		h := newProjectHandlerForTest(project)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"empty-project": {"admin"}})
		resp, err := h.ListStagesV1(ctx, openapi.ListStagesV1RequestObject{ProjectId: "empty-project"})
		Expect(err).NotTo(HaveOccurred())

		ok := resp.(openapi.ListStagesV1200JSONResponse)
		Expect(ok.Data).To(BeEmpty())
	})

	It("never returns stages from another project's landscape namespace", func() {
		project := landscapeProjectFixture("project-a", "kden-p-project-a")
		dev := scopedLandscapeFixture("dev", "kden-p-project-a", "kden-l-dev-a")
		foreign := scopedLandscapeFixture("dev", "kden-p-project-b", "kden-l-dev-b")
		h := newProjectHandlerForTest(project, dev, foreign,
			stageFixture("alpha", "kden-l-dev-a", "vector:1", 1),
			stageFixture("foreign", "kden-l-dev-b", "vector:1", 1),
		)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"project-a": {"admin"}})
		resp, err := h.ListStagesV1(ctx, openapi.ListStagesV1RequestObject{ProjectId: "project-a"})
		Expect(err).NotTo(HaveOccurred())

		ok := resp.(openapi.ListStagesV1200JSONResponse)
		Expect(ok.Data).To(HaveLen(1))
		Expect(ok.Data[0].Name).To(Equal("alpha"))
	})

	It("returns 401 when no session is present", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h := newProjectHandlerForTest(project)

		resp, err := h.ListStagesV1(context.Background(), openapi.ListStagesV1RequestObject{ProjectId: "my-project"})
		expectAPIError(resp, err, http.StatusUnauthorized)
	})

	It("returns 403 for an existing project the caller has no role for", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h := newProjectHandlerForTest(project)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"other-project": {"admin"}})
		resp, err := h.ListStagesV1(ctx, openapi.ListStagesV1RequestObject{ProjectId: "my-project"})
		expectAPIError(resp, err, http.StatusForbidden)
	})

	It("returns 403 for a nonexistent project the caller has no role for", func() {
		h := newProjectHandlerForTest()

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"other-project": {"admin"}})
		resp, err := h.ListStagesV1(ctx, openapi.ListStagesV1RequestObject{ProjectId: "nonexistent"})
		expectAPIError(resp, err, http.StatusForbidden)
	})

	It("returns 404 when an authorized caller requests a nonexistent project", func() {
		h := newProjectHandlerForTest()

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"nonexistent": {"admin"}})
		resp, err := h.ListStagesV1(ctx, openapi.ListStagesV1RequestObject{ProjectId: "nonexistent"})
		expectAPIError(resp, err, http.StatusNotFound)
	})

	It("returns 500 when the project has no managed namespace yet", func() {
		project := landscapeProjectFixture("my-project", "")
		h := newProjectHandlerForTest(project)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ListStagesV1(ctx, openapi.ListStagesV1RequestObject{ProjectId: "my-project"})
		expectAPIError(resp, err, http.StatusInternalServerError)
	})
})
