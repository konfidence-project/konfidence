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

	configReadyCondition := func(config *konfidence.VectorPromotionConfig) *metav1.Condition {
		ExpectWithOffset(1, k8sClient.Get(ctx, types.NamespacedName{
			Name: config.Name, Namespace: testNamespace,
		}, config)).To(Succeed())
		return meta.FindStatusCondition(config.Status.Conditions, konfidence.VectorPromotionConfigReadyCondition)
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

		ready := configReadyCondition(config)
		Expect(ready).NotTo(BeNil())
		Expect(ready.Status).To(Equal(metav1.ConditionTrue))
		Expect(ready.Reason).To(Equal(konfidence.VectorPromotionConfigTargetResolvedReason))
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

	It("reports a missing landscape on the config and keeps the promotion approved", func() {
		config := createConfig("exec-nolandscape-config",
			templateSource("some-template"), stageTargetInLandscape("some-stage", "missing-landscape"))
		promotion := createPromotion("exec-nolandscape-promotion", config.Name)

		reconcile(promotion.Name)
		_, err := newReconciler().Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Name: promotion.Name, Namespace: testNamespace},
		})
		Expect(err).To(MatchError(ContainSubstring(`landscape "missing-landscape" does not exist`)))

		refresh(promotion)
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStateApproved))

		ready := configReadyCondition(config)
		Expect(ready).NotTo(BeNil())
		Expect(ready.Status).To(Equal(metav1.ConditionFalse))
		Expect(ready.Reason).To(Equal(konfidence.VectorPromotionConfigLandscapeNotFoundReason))
	})

	It("reports a missing stage on the config and promotes once it exists", func() {
		createLandscapeWithNamespace("exec-nostage-landscape", "kden-l-exec-nostage")
		config := createConfig("exec-nostage-config",
			templateSource("some-template"), stageTargetInLandscape("late-stage", "exec-nostage-landscape"))
		promotion := createPromotion("exec-nostage-promotion", config.Name)

		By("resolution fails while the stage does not exist; the stage is not created")
		reconcile(promotion.Name)
		_, err := newReconciler().Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Name: promotion.Name, Namespace: testNamespace},
		})
		Expect(err).To(MatchError(ContainSubstring(`stage "late-stage" does not exist`)))

		refresh(promotion)
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStateApproved))
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: "late-stage", Namespace: "kden-l-exec-nostage",
		}, &konfidence.Stage{})).To(MatchError(ContainSubstring("not found")))

		ready := configReadyCondition(config)
		Expect(ready).NotTo(BeNil())
		Expect(ready.Status).To(Equal(metav1.ConditionFalse))
		Expect(ready.Reason).To(Equal(konfidence.VectorPromotionConfigStageNotFoundReason))

		By("creating the stage lets the next reconcile promote")
		stage := createStage("kden-l-exec-nostage", "late-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		reconcile(promotion.Name)
		refresh(promotion)
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStateSucceeded))

		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: stage.Name, Namespace: stage.Namespace,
		}, stage)).To(Succeed())
		Expect(stage.Spec.Vector).To(Equal(testVector))

		ready = configReadyCondition(config)
		Expect(ready.Status).To(Equal(metav1.ConditionTrue))
	})

	It("does not supersede a newer promotion that is still waiting for approval", func() {
		createLandscapeWithNamespace("exec-newer-landscape", "kden-l-exec-newer")
		createStage("kden-l-exec-newer", "newer-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		config := createConfig("exec-newer-config",
			stageSource("other-stage"), stageTargetInLandscape("newer-stage", "exec-newer-landscape"))
		older := createPromotionRequiringApproval("exec-newer-a", config.Name)
		newer := createPromotionRequiringApproval("exec-newer-b", config.Name)
		approvePromotion(older)
		reconcile(newer.Name)

		By("the older approved promotion executes without superseding the waiting newer one")
		reconcile(older.Name)
		refresh(older)
		refresh(newer)
		Expect(older.Status.State).To(Equal(konfidence.PromotionStateSucceeded))
		Expect(newer.Status.State).To(Equal(konfidence.PromotionStateWaitingForApproval))

		By("approving the newer promotion still lets it run")
		approvePromotion(newer)
		reconcile(newer.Name)
		refresh(newer)
		Expect(newer.Status.State).To(Equal(konfidence.PromotionStateSucceeded))
	})

	It("resumes an interrupted execution idempotently", func() {
		createLandscapeWithNamespace("exec-resume-landscape", "kden-l-exec-resume")
		stage := createStage("kden-l-exec-resume", "resume-stage", testVector)
		config := createConfig("exec-resume-config",
			templateSource("some-template"), stageTargetInLandscape("resume-stage", "exec-resume-landscape"))
		promotion := createPromotion("exec-resume-promotion", config.Name)

		By("simulating a crash after the Running patch and the stage write")
		reconcile(promotion.Name)
		setSucceededCondition(promotion, metav1.ConditionFalse, konfidence.ReasonPromotionRunning, metav1.Now().Time)

		By("the next reconcile completes without re-patching the stage")
		reconcile(promotion.Name)
		refresh(promotion)
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStateSucceeded))
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: stage.Name, Namespace: stage.Namespace,
		}, stage)).To(Succeed())
		Expect(stage.Spec.Vector).To(Equal(testVector))
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
