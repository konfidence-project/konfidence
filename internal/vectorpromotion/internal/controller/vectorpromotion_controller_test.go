package controller

import (
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
)

// The execution controller is exercised directly instead of through the shared
// manager: registering it there would race the manually driven conditions the
// TTL and status propagation tests rely on, and direct calls keep the
// superseding and serialization scenarios deterministic.
var _ = Describe("VectorPromotion execution controller", Ordered, Serial, func() {

	BeforeEach(func() { cleanupPromotions() })

	newReconciler := func() *VectorPromotionReconciler {
		return &VectorPromotionReconciler{Client: k8sClient, Recorder: events.NewFakeRecorder(20)}
	}

	reconcile := func(name string) ctrl.Result {
		result, err := newReconciler().Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Name: name, Namespace: testNamespace},
		})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		return result
	}

	refresh := func(promotion *konfidence.VectorPromotion) {
		ExpectWithOffset(1, k8sClient.Get(ctx, types.NamespacedName{
			Name: promotion.Name, Namespace: testNamespace,
		}, promotion)).To(Succeed())
	}

	It("auto-approves and promotes to the target stage", func() {
		createLandscapeWithNamespace("exec-auto-landscape", "kden-l-exec-auto")
		stage := createStage("kden-l-exec-auto", "exec-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		config := createConfig("exec-auto-config",
			templateSource("some-template"), stageTargetInLandscape("exec-stage", "exec-auto-landscape"))
		promotion := createPromotion("exec-auto-promotion", config.Name)

		By("first reconcile approves automatically")
		reconcile(promotion.Name)
		refresh(promotion)
		approved := meta.FindStatusCondition(promotion.Status.Conditions, konfidence.ConditionTypeApproved)
		Expect(approved).NotTo(BeNil())
		Expect(approved.Status).To(Equal(metav1.ConditionTrue))
		Expect(approved.Reason).To(Equal(konfidence.ReasonPromotionAutoApproved))

		By("second reconcile executes the promotion")
		reconcile(promotion.Name)
		refresh(promotion)
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStateSucceeded))
		succeeded := meta.FindStatusCondition(promotion.Status.Conditions, konfidence.ConditionTypeSucceeded)
		Expect(succeeded).NotTo(BeNil())
		Expect(succeeded.Status).To(Equal(metav1.ConditionTrue))

		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: stage.Name, Namespace: stage.Namespace,
		}, stage)).To(Succeed())
		Expect(stage.Spec.Vector).To(Equal(testVector))
	})

	It("waits for approval and promotes once approved", func() {
		createLandscapeWithNamespace("exec-manual-landscape", "kden-l-exec-manual")
		stage := createStage("kden-l-exec-manual", "manual-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		config := createConfig("exec-manual-config",
			stageSource("other-stage"), stageTargetInLandscape("manual-stage", "exec-manual-landscape"))
		promotion := createPromotionRequiringApproval("exec-manual-promotion", config.Name)

		By("first reconcile parks the promotion in WaitingForApproval")
		reconcile(promotion.Name)
		refresh(promotion)
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStateWaitingForApproval))

		By("reconciling again without approval does not execute")
		reconcile(promotion.Name)
		refresh(promotion)
		Expect(meta.FindStatusCondition(promotion.Status.Conditions, konfidence.ConditionTypeSucceeded)).To(BeNil())

		By("approving executes the promotion")
		approvePromotion(promotion)
		reconcile(promotion.Name)
		refresh(promotion)
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStateSucceeded))

		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: stage.Name, Namespace: stage.Namespace,
		}, stage)).To(Succeed())
		Expect(stage.Spec.Vector).To(Equal(testVector))
	})

	It("supersedes an older approved promotion when it reconciles first", func() {
		createLandscapeWithNamespace("exec-super-landscape", "kden-l-exec-super")
		createStage("kden-l-exec-super", "super-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		config := createConfig("exec-super-config",
			stageSource("other-stage"), stageTargetInLandscape("super-stage", "exec-super-landscape"))
		older := createPromotionRequiringApproval("exec-super-a", config.Name)
		newer := createPromotionRequiringApproval("exec-super-b", config.Name)
		approvePromotion(older)
		approvePromotion(newer)

		By("reconciling the older promotion supersedes it")
		reconcile(older.Name)
		refresh(older)
		Expect(older.Status.State).To(Equal(konfidence.PromotionStateSuperseded))
		cond := meta.FindStatusCondition(older.Status.Conditions, konfidence.ConditionTypeSucceeded)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Reason).To(Equal(konfidence.ReasonPromotionSuperseded))

		By("reconciling the newer promotion executes it")
		reconcile(newer.Name)
		refresh(newer)
		Expect(newer.Status.State).To(Equal(konfidence.PromotionStateSucceeded))
	})

	It("supersedes stale approved siblings before executing", func() {
		createLandscapeWithNamespace("exec-stale-landscape", "kden-l-exec-stale")
		createStage("kden-l-exec-stale", "stale-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		config := createConfig("exec-stale-config",
			stageSource("other-stage"), stageTargetInLandscape("stale-stage", "exec-stale-landscape"))
		older := createPromotionRequiringApproval("exec-stale-a", config.Name)
		newer := createPromotionRequiringApproval("exec-stale-b", config.Name)
		approvePromotion(older)
		approvePromotion(newer)

		By("reconciling the newest supersedes the older sibling and executes")
		reconcile(newer.Name)
		refresh(newer)
		refresh(older)
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

		result := reconcile(blocked.Name)
		Expect(result.RequeueAfter).To(Equal(siblingBlockedRequeueInterval))
		refresh(blocked)
		Expect(meta.FindStatusCondition(blocked.Status.Conditions, konfidence.ConditionTypeSucceeded)).To(BeNil())
	})

	It("fails terminally when the config is missing", func() {
		promotion := createPromotion("exec-noconfig-promotion", "does-not-exist")

		reconcile(promotion.Name)
		reconcile(promotion.Name)
		refresh(promotion)
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStateFailed))
		cond := meta.FindStatusCondition(promotion.Status.Conditions, konfidence.ConditionTypeSucceeded)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Reason).To(Equal(konfidence.ReasonPromotionConfigurationNotFound))
	})

	It("retries with backoff while the landscape is unresolved", func() {
		config := createConfig("exec-nolandscape-config",
			templateSource("some-template"), stageTargetInLandscape("some-stage", "missing-landscape"))
		promotion := createPromotion("exec-nolandscape-promotion", config.Name)

		reconcile(promotion.Name)
		_, err := newReconciler().Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Name: promotion.Name, Namespace: testNamespace},
		})
		Expect(err).To(MatchError(ContainSubstring(`failed to resolve landscape "missing-landscape"`)))
		refresh(promotion)
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStateInProgress))
	})

	It("does not touch a terminal promotion", func() {
		promotion := createPromotion("exec-terminal-promotion", "some-config")
		setSucceededCondition(promotion, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, metav1.Now().Time)
		refresh(promotion)
		resourceVersion := promotion.ResourceVersion

		reconcile(promotion.Name)
		refresh(promotion)
		Expect(promotion.ResourceVersion).To(Equal(resourceVersion))
	})
})
