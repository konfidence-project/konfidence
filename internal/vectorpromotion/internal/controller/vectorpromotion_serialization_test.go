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

		By("reconciling the older promotion supersedes it")
		reconcilePromotion(older.Name)
		refreshPromotion(older)
		Expect(older.Status.State).To(Equal(konfidence.PromotionStateSuperseded))
		cond := meta.FindStatusCondition(older.Status.Conditions, konfidence.ConditionTypeSucceeded)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Reason).To(Equal(konfidence.ReasonPromotionSuperseded))

		By("reconciling the newer promotion executes it")
		reconcilePromotion(newer.Name)
		refreshPromotion(newer)
		Expect(newer.Status.State).To(Equal(konfidence.PromotionStateSucceeded))
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
		Expect(newer.Status.State).To(Equal(konfidence.PromotionStateSucceeded))
		Expect(older.Status.State).To(Equal(konfidence.PromotionStateSuperseded))
	})

	It("requeues while a sibling promotion is in progress", func() {
		config := createConfig("exec-blocked-config",
			stageSource("other-stage"), stageTarget("blocked-stage"))
		running := createPromotion("exec-blocked-a", config.Name)
		setSucceededCondition(running, metav1.ConditionFalse, konfidence.ReasonPromotionRunning, metav1.Now().Time)
		blocked := createPromotionRequiringApproval("exec-blocked-b", config.Name)
		approvePromotion(blocked)

		result := reconcilePromotion(blocked.Name)
		Expect(result.RequeueAfter).To(Equal(siblingBlockedRequeueInterval))
		refreshPromotion(blocked)
		Expect(meta.FindStatusCondition(blocked.Status.Conditions, konfidence.ConditionTypeSucceeded)).To(BeNil())
	})

	It("sets state to Ready for a gateless promotion blocked by an in-progress sibling", func() {
		config := createConfig("exec-blocked-gateless-config",
			stageSource("other-stage"), stageTarget("blocked-gateless-stage"))
		running := createPromotion("exec-blocked-gateless-a", config.Name)
		setSucceededCondition(running, metav1.ConditionFalse, konfidence.ReasonPromotionRunning, metav1.Now().Time)
		blocked := createPromotion("exec-blocked-gateless-b", config.Name)

		result := reconcilePromotion(blocked.Name)
		Expect(result.RequeueAfter).To(Equal(siblingBlockedRequeueInterval))
		refreshPromotion(blocked)
		Expect(blocked.Status.State).To(Equal(konfidence.PromotionStateReady))
	})

	It("retires a stale in-progress sibling and proceeds", func() {
		createLandscapeWithNamespace("exec-stale-ip-landscape", "kden-l-exec-stale-ip")
		createStage("kden-l-exec-stale-ip", "stale-ip-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		config := createConfig("exec-stale-ip-config",
			stageSource("other-stage"), stageTargetInLandscape("stale-ip-stage", "exec-stale-ip-landscape"))
		phantom := createPromotion("exec-stale-ip-a", config.Name)
		setSucceededCondition(phantom, metav1.ConditionFalse, konfidence.ReasonPromotionRunning,
			time.Now().Add(-10*time.Minute))
		blocked := createPromotionTargeting("exec-stale-ip-b", config.Name,
			stageTargetInLandscape("stale-ip-stage", "exec-stale-ip-landscape"), true)
		approvePromotion(blocked)

		reconcilePromotion(blocked.Name)
		refreshPromotion(blocked)
		refreshPromotion(phantom)
		Expect(blocked.Status.State).To(Equal(konfidence.PromotionStateSucceeded))
		Expect(phantom.Status.State).To(Equal(konfidence.PromotionStateFailed))
		cond := meta.FindStatusCondition(phantom.Status.Conditions, konfidence.ConditionTypeSucceeded)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Reason).To(Equal(konfidence.ReasonPromotionTimedOut))
	})

	It("times itself out when stuck in progress past the deadline", func() {
		promotion := createPromotion("exec-selftimeout", "selftimeout-config")
		approvePromotion(promotion)
		setSucceededCondition(promotion, metav1.ConditionFalse, konfidence.ReasonPromotionRunning,
			time.Now().Add(-10*time.Minute))

		reconcilePromotion(promotion.Name)
		refreshPromotion(promotion)
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStateFailed))
		cond := meta.FindStatusCondition(promotion.Status.Conditions, konfidence.ConditionTypeSucceeded)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Reason).To(Equal(konfidence.ReasonPromotionTimedOut))
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
		Expect(older.Status.State).To(Equal(konfidence.PromotionStateSucceeded))
		Expect(newer.Status.State).To(Equal(konfidence.PromotionStateWaiting))

		By("approving the newer promotion still lets it run")
		approvePromotion(newer)
		reconcilePromotion(newer.Name)
		refreshPromotion(newer)
		Expect(newer.Status.State).To(Equal(konfidence.PromotionStateSucceeded))
	})
})
