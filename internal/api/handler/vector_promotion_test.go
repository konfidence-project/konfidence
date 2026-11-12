package handler

import (
	"context"
	"net/http"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/auth"
	vectorpromotiondomain "github.com/konfidence-project/konfidence/internal/vectorpromotion"
	"github.com/konfidence-project/konfidence/pkg/promotiongate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// expectAPIError asserts that a handler returned no response object and instead
// surfaced an *apierror.Error with the given HTTP status.
func expectAPIError(resp any, err error, status int) {
	GinkgoHelper()
	Expect(resp).To(BeNil())
	apiErr := apierror.As(err)
	Expect(apiErr).To(HaveOccurred(), "expected an *apierror.Error")
	Expect(apiErr.Status).To(Equal(status))
}

var _ = Describe("GetVectorPromotionV1", func() {
	It("returns the promotion with its mapped fields", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := vectorPromotionFixture("promo-1", "cfg", 7)
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
		Expect(ok.Gates).To(BeNil())
	})

	It("reports every declared gate with its state and provenance", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := vectorPromotionFixture("promo-1", "cfg", 1)
		// Use a fixed time with zero nanoseconds: the fake k8s client round-trips
		// objects through JSON, which truncates metav1.Time to second precision.
		approvalTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		promo.Spec.Gates = []konfidence.PromotionGate{
			{Type: konfidence.GateTypeExplicitApproval},
			{Type: "e2e-tests.example.com"},
		}
		promo.Status.Gates = []konfidence.PromotionGateStatus{{
			Type:    konfidence.GateTypeExplicitApproval,
			State:   konfidence.PromotionGateStateSatisfied,
			Reason:  konfidence.ReasonGateApprovedByUser,
			Message: "approved by alice@example.com",
			SetBy:   "alice@example.com",
			SetAt:   &metav1.Time{Time: approvalTime},
		}}
		h := newProjectHandlerForTest(project, promo)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.GetVectorPromotionV1(ctx, openapi.GetVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})
		Expect(err).NotTo(HaveOccurred())

		ok := resp.(openapi.GetVectorPromotionV1200JSONResponse)
		Expect(ok.Gates).NotTo(BeNil())
		gates := *ok.Gates
		Expect(gates).To(HaveLen(2))

		Expect(gates[0].Type).To(Equal(konfidence.GateTypeExplicitApproval))
		Expect(gates[0].State).To(Equal(openapi.PromotionGateStatusState("Satisfied")))
		Expect(*gates[0].Reason).To(Equal(konfidence.ReasonGateApprovedByUser))
		Expect(*gates[0].Message).To(Equal("approved by alice@example.com"))
		Expect(*gates[0].SetBy).To(Equal("alice@example.com"))
		Expect(*gates[0].SetAt).To(BeTemporally("~", approvalTime, time.Second))

		// A declared gate without a status entry is reported as Pending.
		Expect(gates[1].Type).To(Equal("e2e-tests.example.com"))
		Expect(gates[1].State).To(Equal(openapi.PromotionGateStatusState("Pending")))
		Expect(gates[1].SetBy).To(BeNil())
		Expect(gates[1].SetAt).To(BeNil())
	})

	It("reports the operator override and the Rejected status", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := vectorPromotionFixture("promo-1", "cfg", 1)
		setAt := time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)
		promo.Status.Override = &konfidence.PromotionOverride{
			Type:   konfidence.PromotionOverrideTypeReject,
			SetBy:  "alice@example.com",
			SetAt:  metav1.Time{Time: setAt},
			Reason: "known data-corruption bug",
		}
		promo.Status.Conditions = facts(konfidence.ConditionTypeSucceeded, metav1.ConditionFalse)
		h := newProjectHandlerForTest(project, promo)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.GetVectorPromotionV1(ctx, openapi.GetVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})
		Expect(err).NotTo(HaveOccurred())

		ok := resp.(openapi.GetVectorPromotionV1200JSONResponse)
		Expect(*ok.Status).To(Equal(openapi.VectorPromotionStatusRejected))
		Expect(ok.Override).NotTo(BeNil())
		Expect(ok.Override.Type).To(Equal(openapi.Reject))
		Expect(ok.Override.SetBy).To(Equal("alice@example.com"))
		Expect(ok.Override.SetAt).To(BeTemporally("==", setAt))
		Expect(ok.Override.Reason).To(Equal("known data-corruption bug"))
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
	// The handler maps an already-approved promotion to 204; the already-approved
	// fixture exercises the idempotent return path. A genuine first-time approval is
	// covered by the repository-layer tests in internal/vectorpromotion/repository_test.go.
	It("returns 204 when the promotion is already approved (idempotent success)", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := vectorPromotionFixture("promo-1", "cfg", 1)
		promo.Spec.Gates = []konfidence.PromotionGate{{Type: konfidence.GateTypeExplicitApproval}}
		promo.Status.Gates = []konfidence.PromotionGateStatus{{
			Type:  konfidence.GateTypeExplicitApproval,
			State: konfidence.PromotionGateStateSatisfied,
			SetBy: "bob@example.com",
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

	It("returns 422 when the promotion is superseded", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := vectorPromotionFixture("promo-1", "cfg", 1)
		promo.Spec.Gates = []konfidence.PromotionGate{{Type: konfidence.GateTypeExplicitApproval}}
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
		expectAPIError(resp, err, http.StatusUnprocessableEntity)
		Expect(apierror.As(err).Message).To(ContainSubstring(konfidence.ReasonPromotionSuperseded))
	})

	It("returns 422 when the promotion has already finished", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := vectorPromotionFixture("promo-1", "cfg", 1)
		promo.Spec.Gates = []konfidence.PromotionGate{{Type: konfidence.GateTypeExplicitApproval}}
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
		expectAPIError(resp, err, http.StatusUnprocessableEntity)
	})

	It("returns 422 when the promotion gates are already satisfied", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := vectorPromotionFixture("promo-1", "cfg", 1)
		promo.Spec.Gates = []konfidence.PromotionGate{{Type: konfidence.GateTypeExplicitApproval}}
		promo.Status.Conditions = []metav1.Condition{{
			Type:   konfidence.ConditionTypeGatesSatisfied,
			Status: metav1.ConditionTrue,
			Reason: konfidence.ReasonAllGatesSatisfied,
		}}
		h := newProjectHandlerForTest(project, promo)

		ctx := ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ApproveVectorPromotionV1(ctx, openapi.ApproveVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})
		expectAPIError(resp, err, http.StatusUnprocessableEntity)
	})

	It("returns 422 when the promotion has no explicit-approval gate", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := vectorPromotionFixture("promo-1", "cfg", 1)
		promo.Spec.Gates = []konfidence.PromotionGate{{Type: "e2e-tests.example.com"}}
		h := newProjectHandlerForTest(project, promo)

		ctx := ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ApproveVectorPromotionV1(ctx, openapi.ApproveVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})
		expectAPIError(resp, err, http.StatusUnprocessableEntity)
	})

	It("returns 422 when an operator already rejected the promotion", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := vectorPromotionFixture("promo-1", "cfg", 1)
		promo.Spec.Gates = []konfidence.PromotionGate{{Type: konfidence.GateTypeExplicitApproval}}
		promo.Status.Override = &konfidence.PromotionOverride{
			Type: konfidence.PromotionOverrideTypeReject, SetBy: "bob@example.com", Reason: "known bad",
		}
		h := newProjectHandlerForTest(project, promo)

		ctx := ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ApproveVectorPromotionV1(ctx, openapi.ApproveVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})
		expectAPIError(resp, err, http.StatusUnprocessableEntity)
		Expect(apierror.As(err).Message).To(ContainSubstring("Reject by bob@example.com"))
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
		promo := vectorPromotionFixture("promo-1", "cfg", 1)
		promo.Spec.Gates = []konfidence.PromotionGate{{Type: konfidence.GateTypeExplicitApproval}}
		h := newProjectHandlerForTest(project, promo)

		// Authorized for the project but with an empty subject: the repository
		// rejects the approval with ErrActorMissing, which the handler maps
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

var _ = Describe("RevokeApprovalVectorPromotionV1", func() {
	approvedPromotion := func() *konfidence.VectorPromotion {
		promo := vectorPromotionFixture("promo-1", "cfg", 1)
		promo.Spec.Gates = []konfidence.PromotionGate{
			{Type: konfidence.GateTypeExplicitApproval},
			{Type: "e2e-tests.example.com"},
		}
		promo.Status.Gates = []konfidence.PromotionGateStatus{
			{Type: konfidence.GateTypeExplicitApproval, State: konfidence.PromotionGateStateSatisfied},
			{Type: "e2e-tests.example.com", State: konfidence.PromotionGateStatePending},
		}
		return promo
	}

	It("returns 204 and revokes the approval", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := approvedPromotion()
		h := newProjectHandlerForTest(project, promo)
		ctx := ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"my-project": {"admin"}})

		resp, err := h.RevokeApprovalVectorPromotionV1(ctx, openapi.RevokeApprovalVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})
		Expect(err).NotTo(HaveOccurred())
		_, is204 := resp.(openapi.RevokeApprovalVectorPromotionV1204Response)
		Expect(is204).To(BeTrue())

		updated, err := h.vectorPromotionRepo.Get(ctx, "kden-p-my-project", "promo-1")
		Expect(err).NotTo(HaveOccurred())
		Expect(promotiongate.StateOf(updated, konfidence.GateTypeExplicitApproval)).
			To(Equal(konfidence.PromotionGateStatePending))
	})

	It("returns 204 when the approval was already revoked", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := approvedPromotion()
		h := newProjectHandlerForTest(project, promo)
		ctx := ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"my-project": {"admin"}})
		request := openapi.RevokeApprovalVectorPromotionV1RequestObject{
			ProjectId: "my-project", VectorPromotionId: "promo-1",
		}

		for range 2 {
			resp, err := h.RevokeApprovalVectorPromotionV1(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).To(BeAssignableToTypeOf(openapi.RevokeApprovalVectorPromotionV1204Response{}))
		}
	})

	It("returns 422 when the approval is pending but was not revoked", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := approvedPromotion()
		promo.Status.Gates[0].State = konfidence.PromotionGateStatePending
		h := newProjectHandlerForTest(project, promo)
		ctx := ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"my-project": {"admin"}})

		resp, err := h.RevokeApprovalVectorPromotionV1(ctx, openapi.RevokeApprovalVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})
		expectAPIError(resp, err, http.StatusUnprocessableEntity)
		Expect(apierror.As(err).Message).To(Equal(vectorpromotiondomain.ErrNotApproved.Error()))
	})

	It("returns 422 after the promotion gates are satisfied", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := approvedPromotion()
		promo.Status.Conditions = []metav1.Condition{{
			Type: konfidence.ConditionTypeGatesSatisfied, Status: metav1.ConditionTrue,
		}}
		h := newProjectHandlerForTest(project, promo)
		ctx := ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"my-project": {"admin"}})

		resp, err := h.RevokeApprovalVectorPromotionV1(ctx, openapi.RevokeApprovalVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "promo-1",
		})
		expectAPIError(resp, err, http.StatusUnprocessableEntity)
	})

	DescribeTable("returns 422 when the promotion has finished",
		func(status metav1.ConditionStatus, reason string) {
			project := landscapeProjectFixture("my-project", "kden-p-my-project")
			promo := approvedPromotion()
			promo.Status.Conditions = []metav1.Condition{{
				Type: konfidence.ConditionTypeSucceeded, Status: status, Reason: reason,
			}}
			h := newProjectHandlerForTest(project, promo)
			ctx := ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"my-project": {"admin"}})

			resp, err := h.RevokeApprovalVectorPromotionV1(ctx, openapi.RevokeApprovalVectorPromotionV1RequestObject{
				ProjectId: "my-project", VectorPromotionId: "promo-1",
			})
			expectAPIError(resp, err, http.StatusUnprocessableEntity)
			Expect(apierror.As(err).Message).To(ContainSubstring(reason))
		},
		Entry("succeeded", metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded),
		Entry("superseded", metav1.ConditionFalse, konfidence.ReasonPromotionSuperseded),
	)

	It("returns 422 when the promotion has no explicit-approval gate", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := approvedPromotion()
		promo.Spec.Gates = []konfidence.PromotionGate{{Type: "e2e-tests.example.com"}}
		h := newProjectHandlerForTest(project, promo)
		ctx := ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"my-project": {"admin"}})

		resp, err := h.RevokeApprovalVectorPromotionV1(ctx, openapi.RevokeApprovalVectorPromotionV1RequestObject{
			ProjectId: "my-project", VectorPromotionId: "promo-1",
		})
		expectAPIError(resp, err, http.StatusUnprocessableEntity)
		Expect(apierror.As(err).Message).To(Equal(vectorpromotiondomain.ErrApprovalNotRequired.Error()))
	})

	It("returns 404 when the promotion does not exist", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h := newProjectHandlerForTest(project)
		ctx := ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"my-project": {"admin"}})

		resp, err := h.RevokeApprovalVectorPromotionV1(ctx, openapi.RevokeApprovalVectorPromotionV1RequestObject{
			ProjectId:         "my-project",
			VectorPromotionId: "missing",
		})
		expectAPIError(resp, err, http.StatusNotFound)
	})
})

var _ = DescribeTable("promotion actions map Kubernetes conflicts to HTTP 409",
	func(revoke bool) {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		promo := vectorPromotionFixture("promo-1", "cfg", 1)
		promo.Spec.Gates = []konfidence.PromotionGate{
			{Type: konfidence.GateTypeExplicitApproval}, {Type: "e2e-tests.example.com"},
		}
		if revoke {
			promo.Status.Gates = []konfidence.PromotionGateStatus{{
				Type: konfidence.GateTypeExplicitApproval, State: konfidence.PromotionGateStateSatisfied,
			}}
		}
		h := newProjectHandlerForTest(project, promo)
		k8s := fake.NewClientBuilder().
			WithScheme(newTestScheme()).
			WithObjects(promo).
			WithStatusSubresource(&konfidence.VectorPromotion{}).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(_ context.Context, _ client.Client, _ string, obj client.Object,
					_ client.Patch, _ ...client.SubResourcePatchOption) error {
					return apierrors.NewConflict(
						konfidence.GroupVersion.WithResource("vectorpromotions").GroupResource(),
						obj.GetName(), nil,
					)
				},
			}).Build()
		h.vectorPromotionRepo = vectorpromotiondomain.NewRepository(k8s)
		ctx := ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"my-project": {"admin"}})

		var resp any
		var err error
		if revoke {
			resp, err = h.RevokeApprovalVectorPromotionV1(ctx, openapi.RevokeApprovalVectorPromotionV1RequestObject{
				ProjectId: "my-project", VectorPromotionId: "promo-1",
			})
		} else {
			resp, err = h.ApproveVectorPromotionV1(ctx, openapi.ApproveVectorPromotionV1RequestObject{
				ProjectId: "my-project", VectorPromotionId: "promo-1",
			})
		}
		expectAPIError(resp, err, http.StatusConflict)
		Expect(apierror.As(err).Code).To(Equal("conflict"))
		Expect(apierror.As(err).Message).To(Equal(vectorpromotiondomain.ErrConflict.Error()))
	},
	Entry("approve", false),
	Entry("revoke approval", true),
)

var _ = DescribeTable("displayState summarizes the observed facts",
	func(conditions []metav1.Condition, started bool, expected openapi.VectorPromotionStatus) {
		p := &konfidence.VectorPromotion{Status: konfidence.VectorPromotionStatus{Conditions: conditions}}
		if started {
			now := metav1.Now()
			p.Status.StartTime = &now
		}
		Expect(displayState(p)).To(Equal(expected))
	},
	Entry("never reconciled", nil, false, openapi.VectorPromotionStatusWaiting),
	Entry("a gate not satisfied yet", facts(konfidence.ConditionTypeGatesSatisfied, metav1.ConditionFalse), false,
		openapi.VectorPromotionStatusWaiting),
	Entry("gates satisfied", facts(konfidence.ConditionTypeGatesSatisfied, metav1.ConditionTrue), false,
		openapi.VectorPromotionStatusReady),
	Entry("target unresolved", append(facts(konfidence.ConditionTypeGatesSatisfied, metav1.ConditionTrue),
		facts(konfidence.ConditionTypeTargetResolved, metav1.ConditionFalse)...), false,
		openapi.VectorPromotionStatusBlocked),
	Entry("started", facts(konfidence.ConditionTypeGatesSatisfied, metav1.ConditionTrue), true,
		openapi.VectorPromotionStatusInProgress),
	Entry("started but target no longer resolves", append(facts(konfidence.ConditionTypeGatesSatisfied, metav1.ConditionTrue),
		facts(konfidence.ConditionTypeTargetResolved, metav1.ConditionFalse)...), true,
		openapi.VectorPromotionStatusBlocked),
	Entry("succeeded", facts(konfidence.ConditionTypeSucceeded, metav1.ConditionTrue), true,
		openapi.VectorPromotionStatusSucceeded),
	Entry("superseded", facts(konfidence.ConditionTypeSucceeded, metav1.ConditionFalse), false,
		openapi.VectorPromotionStatusSuperseded),
)

var _ = DescribeTable("displayState reads the operator override",
	func(overrideType konfidence.PromotionOverrideType, conditions []metav1.Condition, expected openapi.VectorPromotionStatus) {
		p := &konfidence.VectorPromotion{Status: konfidence.VectorPromotionStatus{
			Conditions: conditions,
			Override:   &konfidence.PromotionOverride{Type: overrideType},
		}}
		Expect(displayState(p)).To(Equal(expected))
	},
	Entry("rejection not yet observed", konfidence.PromotionOverrideTypeReject,
		facts(konfidence.ConditionTypeGatesSatisfied, metav1.ConditionFalse), openapi.VectorPromotionStatusWaiting),
	Entry("rejected", konfidence.PromotionOverrideTypeReject,
		facts(konfidence.ConditionTypeSucceeded, metav1.ConditionFalse), openapi.VectorPromotionStatusRejected),
	Entry("gates bypassed", konfidence.PromotionOverrideTypeBypassGates,
		facts(konfidence.ConditionTypeGatesSatisfied, metav1.ConditionFalse), openapi.VectorPromotionStatusReady),
	Entry("gates bypassed, then superseded", konfidence.PromotionOverrideTypeBypassGates,
		facts(konfidence.ConditionTypeSucceeded, metav1.ConditionFalse), openapi.VectorPromotionStatusSuperseded),
)

var _ = Describe("RejectVectorPromotionV1 and BypassVectorPromotionGatesV1", func() {
	type operation func(h *projectHandler, ctx context.Context, body *openapi.PromotionOverrideRequestBody) (any, error)
	reject := func(h *projectHandler, ctx context.Context, body *openapi.PromotionOverrideRequestBody) (any, error) {
		resp, err := h.RejectVectorPromotionV1(ctx, openapi.RejectVectorPromotionV1RequestObject{
			ProjectId: "my-project", VectorPromotionId: "promo-1", Body: body,
		})
		if resp == nil {
			return nil, err
		}
		return resp, err
	}
	bypassGates := func(h *projectHandler, ctx context.Context, body *openapi.PromotionOverrideRequestBody) (any, error) {
		resp, err := h.BypassVectorPromotionGatesV1(ctx, openapi.BypassVectorPromotionGatesV1RequestObject{
			ProjectId: "my-project", VectorPromotionId: "promo-1", Body: body,
		})
		if resp == nil {
			return nil, err
		}
		return resp, err
	}
	reason := &openapi.PromotionOverrideRequestBody{Reason: "known data-corruption bug"}
	admin := func() context.Context {
		return ctxWithSubjectAndProjectRoles("alice@example.com", auth.ProjectRoles{"my-project": {"admin"}})
	}
	waitingPromotion := func() *konfidence.VectorPromotion {
		promo := vectorPromotionFixture("promo-1", "cfg", 1)
		promo.Spec.Gates = []konfidence.PromotionGate{{Type: konfidence.GateTypeExplicitApproval}}
		return promo
	}
	withOverride := func(promo *konfidence.VectorPromotion, overrideType konfidence.PromotionOverrideType) *konfidence.VectorPromotion {
		promo.Status.Override = &konfidence.PromotionOverride{
			Type: overrideType, SetBy: "bob@example.com", SetAt: metav1.Now(), Reason: "first decision",
		}
		return promo
	}

	DescribeTable("records the decision for the session subject",
		func(op operation, expectedType konfidence.PromotionOverrideType, expectedResponse any) {
			project := landscapeProjectFixture("my-project", "kden-p-my-project")
			h := newProjectHandlerForTest(project, waitingPromotion())

			resp, err := op(h, admin(), reason)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).To(BeAssignableToTypeOf(expectedResponse))

			got, err := h.vectorPromotionRepo.Get(context.Background(), "kden-p-my-project", "promo-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Status.Override).NotTo(BeNil())
			Expect(got.Status.Override.Type).To(Equal(expectedType))
			Expect(got.Status.Override.SetBy).To(Equal("alice@example.com"))
			Expect(got.Status.Override.Reason).To(Equal(reason.Reason))
		},
		Entry("reject", operation(reject), konfidence.PromotionOverrideTypeReject,
			openapi.RejectVectorPromotionV1204Response{}),
		Entry("bypass gates", operation(bypassGates), konfidence.PromotionOverrideTypeBypassGates,
			openapi.BypassVectorPromotionGatesV1204Response{}),
	)

	DescribeTable("is idempotent for the same decision",
		func(op operation, existing konfidence.PromotionOverrideType) {
			project := landscapeProjectFixture("my-project", "kden-p-my-project")
			h := newProjectHandlerForTest(project, withOverride(waitingPromotion(), existing))

			resp, err := op(h, admin(), reason)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
		},
		Entry("reject twice", operation(reject), konfidence.PromotionOverrideTypeReject),
		Entry("bypass twice", operation(bypassGates), konfidence.PromotionOverrideTypeBypassGates),
	)

	DescribeTable("maps failures to client errors",
		func(op operation, promo *konfidence.VectorPromotion, body *openapi.PromotionOverrideRequestBody, status int) {
			objects := []client.Object{landscapeProjectFixture("my-project", "kden-p-my-project")}
			if promo != nil {
				objects = append(objects, promo)
			}
			h := newProjectHandlerForTest(objects...)

			resp, err := op(h, admin(), body)
			expectAPIError(resp, err, status)
		},
		Entry("reject after the gates were bypassed", operation(reject),
			withOverride(waitingPromotion(), konfidence.PromotionOverrideTypeBypassGates), reason, http.StatusUnprocessableEntity),
		Entry("bypass after a rejection", operation(bypassGates),
			withOverride(waitingPromotion(), konfidence.PromotionOverrideTypeReject), reason, http.StatusUnprocessableEntity),
		Entry("reject a finished promotion", operation(reject),
			withConditions(waitingPromotion(), facts(konfidence.ConditionTypeSucceeded, metav1.ConditionTrue)),
			reason, http.StatusUnprocessableEntity),
		Entry("bypass gates that are already satisfied", operation(bypassGates),
			withConditions(waitingPromotion(), facts(konfidence.ConditionTypeGatesSatisfied, metav1.ConditionTrue)),
			reason, http.StatusUnprocessableEntity),
		Entry("reject without a body", operation(reject), waitingPromotion(), nil, http.StatusBadRequest),
		Entry("bypass with an empty reason", operation(bypassGates), waitingPromotion(),
			&openapi.PromotionOverrideRequestBody{}, http.StatusBadRequest),
		Entry("reject a missing promotion", operation(reject), nil, reason, http.StatusNotFound),
		Entry("bypass a missing promotion", operation(bypassGates), nil, reason, http.StatusNotFound),
	)

	DescribeTable("requires access to the project",
		func(op operation) {
			project := landscapeProjectFixture("my-project", "kden-p-my-project")
			h := newProjectHandlerForTest(project, waitingPromotion())

			resp, err := op(h, ctxWithProjectRoles(auth.ProjectRoles{"other-project": {"admin"}}), reason)
			expectAPIError(resp, err, http.StatusForbidden)

			resp, err = op(h, context.Background(), reason)
			expectAPIError(resp, err, http.StatusUnauthorized)
		},
		Entry("reject", operation(reject)),
		Entry("bypass gates", operation(bypassGates)),
	)
})

func withConditions(promo *konfidence.VectorPromotion, conditions []metav1.Condition) *konfidence.VectorPromotion {
	promo.Status.Conditions = conditions
	return promo
}

func facts(conditionType string, status metav1.ConditionStatus) []metav1.Condition {
	return []metav1.Condition{{Type: conditionType, Status: status, Reason: "SetByTest"}}
}
