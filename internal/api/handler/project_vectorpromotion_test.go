package handler

import (
	"context"
	"net/http"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/auth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// expectAPIError asserts that a handler returned no response object and instead
// surfaced an *apierror.Error with the given HTTP status.
func expectAPIError(resp any, err error, status int) {
	GinkgoHelper()
	Expect(resp).To(BeNil())
	apiErr := apierror.As(err)
	Expect(apiErr).NotTo(BeNil(), "expected an *apierror.Error")
	Expect(apiErr.Status).To(Equal(status))
}

var _ = Describe("GetVectorPromotionV1", func() {
	It("returns the promotion with its mapped fields", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := vectorPromotionFixture("promo-1", "kden-p-my-project", "cfg", 7)
		h := newProjectHandlerForTest(project, promo)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.GetVectorPromotionV1(ctx, openapi.GetVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})
		Expect(err).NotTo(HaveOccurred())

		ok, is200 := resp.(openapi.GetVectorPromotionV1200JSONResponse)
		Expect(is200).To(BeTrue())
		Expect(ok.Id).To(Equal("promo-1"))
		Expect(ok.Source.Name).To(Equal("src"))
		Expect(ok.Target.Name).To(Equal("dst"))
		Expect(ok.Vector).To(Equal("registry//component:v1"))
		Expect(ok.Sequence).NotTo(BeNil())
		Expect(*ok.Sequence).To(Equal(int64(7)))
		Expect(ok.RequireApproval).NotTo(BeNil())
		Expect(*ok.RequireApproval).To(BeFalse())
	})

	It("includes approval metadata when the promotion has been approved", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := vectorPromotionFixture("promo-1", "kden-p-my-project", "cfg", 1)
		// Use a fixed time with zero nanoseconds: the fake k8s client round-trips
		// objects through JSON, which truncates metav1.Time to second precision.
		approvalTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		promo.Status.Approval = &konfidence.PromotionApproval{
			ApprovedBy: "alice@example.com",
			ApprovedAt: metav1.NewTime(approvalTime),
		}
		h := newProjectHandlerForTest(project, promo)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.GetVectorPromotionV1(ctx, openapi.GetVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})
		Expect(err).NotTo(HaveOccurred())

		ok := resp.(openapi.GetVectorPromotionV1200JSONResponse)
		Expect(ok.Approval).NotTo(BeNil())
		Expect(ok.Approval.ApprovedBy).To(Equal("alice@example.com"))
		Expect(ok.Approval.ApprovedAt).To(BeTemporally("~", approvalTime, time.Second))
	})

	It("returns 404 when the promotion does not exist", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h := newProjectHandlerForTest(project)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.GetVectorPromotionV1(ctx, openapi.GetVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "missing",
		})
		expectAPIError(resp, err, http.StatusNotFound)
	})

	It("returns 404 when the project does not exist", func() {
		h := newProjectHandlerForTest()

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"nonexistent": {"admin"}})
		resp, err := h.GetVectorPromotionV1(ctx, openapi.GetVectorPromotionV1RequestObject{
			ProjectId:         "nonexistent",
			VectorPromotionId: "promo-1",
		})

		expectAPIError(resp, err, http.StatusNotFound)
	})

	It("returns 403 when the caller has no role for the project", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h := newProjectHandlerForTest(project)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"other-project": {"admin"}})
		resp, err := h.GetVectorPromotionV1(ctx, openapi.GetVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})

		expectAPIError(resp, err, http.StatusForbidden)
	})

	It("returns 500 when the resolved project has no namespace", func() {
		project := landscapeProjectFixture("my-project", "")
		h := newProjectHandlerForTest(project)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.GetVectorPromotionV1(ctx, openapi.GetVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})

		expectAPIError(resp, err, http.StatusInternalServerError)
	})

	It("returns 401 when no session is present", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h := newProjectHandlerForTest(project)

		resp, err := h.GetVectorPromotionV1(context.Background(), openapi.GetVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})

		expectAPIError(resp, err, http.StatusUnauthorized)
	})
})

var _ = Describe("ApproveVectorPromotionV1", func() {
	// The handler maps both err==nil and ErrAlreadyApproved to 204; the already-approved
	// fixture exercises that return path without requiring a k8s Status().Patch() call.
	// A genuine first-time approval (err==nil) is not asserted here: fakeK8s does not
	// register VectorPromotion as a status subresource, so Status().Patch() would succeed
	// but bypass the subresource restriction rather than exercise the realistic path. That
	// path is covered by the repository-layer tests in internal/vectorpromotion/repository_test.go.
	It("returns 204 when the promotion is already approved (idempotent success)", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := vectorPromotionFixture("promo-1", "kden-p-my-project", "cfg", 1)
		promo.Spec.RequireApproval = true
		promo.Status.Conditions = []metav1.Condition{{
			Type:   konfidence.ConditionTypeApproved,
			Status: metav1.ConditionTrue,
			Reason: konfidence.ReasonPromotionManuallyApproved,
		}}
		h := newProjectHandlerForTest(project, promo)

		ctx := ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ApproveVectorPromotionV1(ctx, openapi.ApproveVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})
		Expect(err).NotTo(HaveOccurred())

		_, is204 := resp.(openapi.ApproveVectorPromotionV1204Response)
		Expect(is204).To(BeTrue())
	})

	It("returns 404 when the promotion does not exist", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h := newProjectHandlerForTest(project)

		ctx := ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ApproveVectorPromotionV1(ctx, openapi.ApproveVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "missing",
		})

		expectAPIError(resp, err, http.StatusNotFound)
	})

	It("returns 404 when the project does not exist", func() {
		h := newProjectHandlerForTest()

		ctx := ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"nonexistent": {"admin"}})
		resp, err := h.ApproveVectorPromotionV1(ctx, openapi.ApproveVectorPromotionV1RequestObject{
			ProjectId:         "nonexistent",
			VectorPromotionId: "promo-1",
		})

		expectAPIError(resp, err, http.StatusNotFound)
	})

	It("returns 409 when the promotion is superseded", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := vectorPromotionFixture("promo-1", "kden-p-my-project", "cfg", 1)
		promo.Spec.RequireApproval = true
		promo.Status.Conditions = []metav1.Condition{{
			Type:   konfidence.ConditionTypeSucceeded,
			Status: metav1.ConditionFalse,
			Reason: konfidence.ReasonPromotionSuperseded,
		}}
		h := newProjectHandlerForTest(project, promo)

		ctx := ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ApproveVectorPromotionV1(ctx, openapi.ApproveVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})
		expectAPIError(resp, err, http.StatusConflict)
	})

	It("returns 409 when the promotion has already finished", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := vectorPromotionFixture("promo-1", "kden-p-my-project", "cfg", 1)
		promo.Spec.RequireApproval = true
		promo.Status.Conditions = []metav1.Condition{{
			Type:   konfidence.ConditionTypeSucceeded,
			Status: metav1.ConditionTrue,
			Reason: konfidence.ReasonPromotionSucceeded,
		}}
		h := newProjectHandlerForTest(project, promo)

		ctx := ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ApproveVectorPromotionV1(ctx, openapi.ApproveVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})
		expectAPIError(resp, err, http.StatusConflict)
	})

	It("returns 409 when the promotion has no approval gate", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := vectorPromotionFixture("promo-1", "kden-p-my-project", "cfg", 1)
		// RequireApproval is false by default in the fixture; make it explicit.
		promo.Spec.RequireApproval = false
		h := newProjectHandlerForTest(project, promo)

		ctx := ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ApproveVectorPromotionV1(ctx, openapi.ApproveVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})
		expectAPIError(resp, err, http.StatusConflict)
	})

	It("returns 403 when the caller has no role for the project", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h := newProjectHandlerForTest(project)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"other-project": {"admin"}})
		resp, err := h.ApproveVectorPromotionV1(ctx, openapi.ApproveVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})

		expectAPIError(resp, err, http.StatusForbidden)
	})

	It("returns 500 when the resolved project has no namespace", func() {
		project := landscapeProjectFixture("my-project", "")
		h := newProjectHandlerForTest(project)

		ctx := ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ApproveVectorPromotionV1(ctx, openapi.ApproveVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})

		expectAPIError(resp, err, http.StatusInternalServerError)
	})

	It("returns 500 when the session carries no subject to record as approver", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := vectorPromotionFixture("promo-1", "kden-p-my-project", "cfg", 1)
		promo.Spec.RequireApproval = true
		h := newProjectHandlerForTest(project, promo)

		// Authorized for the project but with an empty subject: the repository
		// rejects the approval with ErrApproverMissing, which the handler maps
		// to a 500 via its default case.
		ctx := ctxWithSubjectAndProjectRoles("", auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ApproveVectorPromotionV1(ctx, openapi.ApproveVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})
		expectAPIError(resp, err, http.StatusInternalServerError)
	})

	It("returns 401 when no session is present", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h := newProjectHandlerForTest(project)

		resp, err := h.ApproveVectorPromotionV1(context.Background(), openapi.ApproveVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})

		expectAPIError(resp, err, http.StatusUnauthorized)
	})
})
