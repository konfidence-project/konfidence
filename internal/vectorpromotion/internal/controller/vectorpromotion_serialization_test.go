package controller

import (
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("VectorPromotion serialization", Ordered, Serial, func() {

	BeforeEach(func() { cleanupPromotions() })

	It("supersedes an older approved promotion when it reconciles first", func() {
		createLandscapeWithNamespace("exec-super-landscape", "kden-l-exec-super")
		createStage("kden-l-exec-super", "super-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		config := createConfig("exec-super-config",
			stageSource("other-stage"), stageTargetInLandscape("super-stage", "exec-super-landscape"))
		older := createPromotionTargeting("exec-super-a", config.Name,
			stageTargetInLandscape("super-stage", "exec-super-landscape"), true)
		newer := createPromotionTargeting("exec-super-b", config.Name,
			stageTargetInLandscape("super-stage", "exec-super-landscape"), true)
		approvePromotion(older)
		approvePromotion(newer)

		By("the newer promotion records Ready (its approval triggered a reconcile)")
		Expect(newExecutionReconciler().reconcileGates(ctx, newer)).To(Succeed())

		By("reconciling the older promotion supersedes it")
		reconcilePromotion(older.Name)
		refreshPromotion(older)
		Expect(observedState(older)).To(Equal("Superseded"))
		cond := meta.FindStatusCondition(older.Status.Conditions, konfidence.ConditionTypeSucceeded)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Reason).To(Equal(konfidence.ReasonPromotionSuperseded))

		By("reconciling the newer promotion executes it")
		reconcilePromotion(newer.Name)
		refreshPromotion(newer)
		Expect(observedState(newer)).To(Equal("Succeeded"))
	})

	It("supersedes stale approved siblings before executing", func() {
		createLandscapeWithNamespace("exec-stale-landscape", "kden-l-exec-stale")
		createStage("kden-l-exec-stale", "stale-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		config := createConfig("exec-stale-config",
			stageSource("other-stage"), stageTargetInLandscape("stale-stage", "exec-stale-landscape"))
		older := createPromotionTargeting("exec-stale-a", config.Name,
			stageTargetInLandscape("stale-stage", "exec-stale-landscape"), true)
		newer := createPromotionTargeting("exec-stale-b", config.Name,
			stageTargetInLandscape("stale-stage", "exec-stale-landscape"), true)
		approvePromotion(older)
		approvePromotion(newer)

		By("reconciling the newest supersedes the older sibling and executes")
		reconcilePromotion(newer.Name)
		refreshPromotion(newer)
		refreshPromotion(older)
		Expect(observedState(newer)).To(Equal("Succeeded"))
		Expect(observedState(older)).To(Equal("Superseded"))
	})

	It("requeues while a sibling promotion has started", func() {
		config := createConfig("exec-blocked-config",
			stageSource("other-stage"), stageTarget("blocked-stage"))
		running := createPromotion("exec-blocked-a", config.Name)
		setSucceededCondition(running, metav1.ConditionUnknown, konfidence.ReasonPromotionRunning, metav1.Now().Time)
		blocked := createPromotionRequiringApproval("exec-blocked-b", config.Name)
		approvePromotion(blocked)

		result := reconcilePromotion(blocked.Name)
		Expect(result.RequeueAfter).To(Equal(siblingBlockedRequeueInterval))
		refreshPromotion(blocked)
		Expect(meta.IsStatusConditionTrue(blocked.Status.Conditions, konfidence.ConditionTypeGatesSatisfied)).To(BeTrue())
		Expect(blocked.Status.StartTime).To(BeNil())
		Expect(observedState(blocked)).To(Equal("Ready"))
	})

	It("sets state to Ready for a gateless promotion blocked by an in-progress sibling", func() {
		config := createConfig("exec-blocked-gateless-config",
			stageSource("other-stage"), stageTarget("blocked-gateless-stage"))
		running := createPromotion("exec-blocked-gateless-a", config.Name)
		setSucceededCondition(running, metav1.ConditionUnknown, konfidence.ReasonPromotionRunning, metav1.Now().Time)
		blocked := createPromotion("exec-blocked-gateless-b", config.Name)

		result := reconcilePromotion(blocked.Name)
		Expect(result.RequeueAfter).To(Equal(siblingBlockedRequeueInterval))
		refreshPromotion(blocked)
		Expect(observedState(blocked)).To(Equal("Ready"))
	})

	It("waits behind an old in-progress sibling until its own retry ends it", func() {
		createLandscapeWithNamespace("exec-stale-ip-landscape", "kden-l-exec-stale-ip")
		createStage("kden-l-exec-stale-ip", "stale-ip-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		config := createConfig("exec-stale-ip-config",
			stageSource("other-stage"), stageTargetInLandscape("stale-ip-stage", "exec-stale-ip-landscape"))
		interrupted := createPromotionTargeting("exec-stale-ip-a", config.Name,
			stageTargetInLandscape("stale-ip-stage", "exec-stale-ip-landscape"), false)
		setSucceededCondition(interrupted, metav1.ConditionUnknown, konfidence.ReasonPromotionRunning,
			time.Now().Add(-time.Hour))
		blocked := createPromotionTargeting("exec-stale-ip-b", config.Name,
			stageTargetInLandscape("stale-ip-stage", "exec-stale-ip-landscape"), true)
		approvePromotion(blocked)

		By("no deadline: however old, an in-progress sibling keeps blocking")
		result := reconcilePromotion(blocked.Name)
		Expect(result.RequeueAfter).To(Equal(siblingBlockedRequeueInterval))
		refreshPromotion(interrupted)
		Expect(observedState(interrupted)).To(Equal("InProgress"))

		By("the interrupted promotion's own retry ends it: the newer cleared sibling wins")
		reconcilePromotion(interrupted.Name)
		refreshPromotion(interrupted)
		Expect(observedState(interrupted)).To(Equal("Superseded"))

		By("the blocked promotion proceeds")
		reconcilePromotion(blocked.Name)
		refreshPromotion(blocked)
		Expect(observedState(blocked)).To(Equal("Succeeded"))
	})

	It("does not supersede a newer promotion that is still waiting for approval", func() {
		createLandscapeWithNamespace("exec-newer-landscape", "kden-l-exec-newer")
		createStage("kden-l-exec-newer", "newer-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		config := createConfig("exec-newer-config",
			stageSource("other-stage"), stageTargetInLandscape("newer-stage", "exec-newer-landscape"))
		older := createPromotionTargeting("exec-newer-a", config.Name,
			stageTargetInLandscape("newer-stage", "exec-newer-landscape"), true)
		newer := createPromotionTargeting("exec-newer-b", config.Name,
			stageTargetInLandscape("newer-stage", "exec-newer-landscape"), true)
		approvePromotion(older)
		reconcilePromotion(newer.Name)

		By("the older approved promotion executes without superseding the waiting newer one")
		reconcilePromotion(older.Name)
		refreshPromotion(older)
		refreshPromotion(newer)
		Expect(observedState(older)).To(Equal("Succeeded"))
		Expect(observedState(newer)).To(Equal("Waiting"))

		By("approving the newer promotion still lets it run")
		approvePromotion(newer)
		reconcilePromotion(newer.Name)
		refreshPromotion(newer)
		Expect(observedState(newer)).To(Equal("Succeeded"))
	})
})
