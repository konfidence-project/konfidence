package controller

import (
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	utils "github.com/konfidence-project/konfidence/pkg/controller"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Shared drivers for the execution controller specs: direct Reconcile calls
// keep superseding and serialization scenarios deterministic (see suite_test.go).

func newExecutionReconciler() *VectorPromotionReconciler {
	return &VectorPromotionReconciler{Client: k8sClient, Recorder: events.NewFakeRecorder(20)}
}

func reconcilePromotion(name string) ctrl.Result {
	result, err := newExecutionReconciler().Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: name, Namespace: testNamespace},
	})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return result
}

func refreshPromotion(promotion *konfidence.VectorPromotion) {
	ExpectWithOffset(1, k8sClient.Get(ctx, types.NamespacedName{
		Name: promotion.Name, Namespace: testNamespace,
	}, promotion)).To(Succeed())
}

func configReadyCondition(config *konfidence.VectorPromotionConfig) *metav1.Condition {
	ExpectWithOffset(1, k8sClient.Get(ctx, types.NamespacedName{
		Name: config.Name, Namespace: testNamespace,
	}, config)).To(Succeed())
	return meta.FindStatusCondition(config.Status.Conditions, konfidence.VectorPromotionConfigReadyCondition)
}

var _ = Describe("VectorPromotion execution", Ordered, Serial, func() {

	BeforeEach(func() { cleanupPromotions() })

	It("auto-approves and promotes to the target stage", func() {
		createLandscapeWithNamespace("exec-auto-landscape", "kden-l-exec-auto")
		stage := createStage("kden-l-exec-auto", "exec-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		config := createConfig("exec-auto-config",
			templateSource("some-template"), stageTargetInLandscape("exec-stage", "exec-auto-landscape"))
		promotion := createPromotion("exec-auto-promotion", config.Name)

		By("first reconcile approves automatically")
		reconcilePromotion(promotion.Name)
		refreshPromotion(promotion)
		approved := meta.FindStatusCondition(promotion.Status.Conditions, konfidence.ConditionTypeApproved)
		Expect(approved).NotTo(BeNil())
		Expect(approved.Status).To(Equal(metav1.ConditionTrue))
		Expect(approved.Reason).To(Equal(konfidence.ReasonPromotionAutoApproved))

		By("second reconcile executes the promotion")
		reconcilePromotion(promotion.Name)
		refreshPromotion(promotion)
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStateSucceeded))
		succeeded := meta.FindStatusCondition(promotion.Status.Conditions, konfidence.ConditionTypeSucceeded)
		Expect(succeeded).NotTo(BeNil())
		Expect(succeeded.Status).To(Equal(metav1.ConditionTrue))

		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: stage.Name, Namespace: stage.Namespace,
		}, stage)).To(Succeed())
		Expect(stage.Spec.Vector).To(Equal(testVector))
		Expect(stage.Annotations).To(HaveKeyWithValue(utils.PromotedByAnnotation,
			testNamespace+"/"+promotion.Name))

		Expect(promotion.Status.PromotedStageRef).NotTo(BeNil())
		Expect(promotion.Status.PromotedStageRef.Name).To(Equal(stage.Name))
		Expect(*promotion.Status.PromotedStageRef.Namespace).To(Equal(stage.Namespace))

		By("the promotion outcome is mirrored onto the config")
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: config.Name, Namespace: testNamespace,
		}, config)).To(Succeed())
		mirrored := meta.FindStatusCondition(config.Status.LastPromotionConditions, konfidence.ConditionTypeSucceeded)
		Expect(mirrored).NotTo(BeNil())
		Expect(mirrored.Status).To(Equal(metav1.ConditionTrue))
		Expect(config.Status.LastSuccessfulPromotionConditions).NotTo(BeEmpty())
	})

	It("fails terminally when the config is missing", func() {
		promotion := createPromotion("exec-noconfig-promotion", "does-not-exist")

		reconcilePromotion(promotion.Name)
		reconcilePromotion(promotion.Name)
		refreshPromotion(promotion)
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStateFailed))
		cond := meta.FindStatusCondition(promotion.Status.Conditions, konfidence.ConditionTypeSucceeded)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Reason).To(Equal(konfidence.ReasonPromotionConfigurationNotFound))
	})

	It("reports a missing landscape on the config and keeps the promotion approved", func() {
		config := createConfig("exec-nolandscape-config",
			templateSource("some-template"), stageTargetInLandscape("some-stage", "missing-landscape"))
		promotion := createPromotion("exec-nolandscape-promotion", config.Name)

		reconcilePromotion(promotion.Name)
		_, err := newExecutionReconciler().Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Name: promotion.Name, Namespace: testNamespace},
		})
		Expect(err).To(MatchError(ContainSubstring(`landscape "missing-landscape" does not exist`)))

		refreshPromotion(promotion)
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStateBlocked))

	})

	It("reports a missing stage on the config and promotes once it exists", func() {
		createLandscapeWithNamespace("exec-nostage-landscape", "kden-l-exec-nostage")
		config := createConfig("exec-nostage-config",
			templateSource("some-template"), stageTargetInLandscape("late-stage", "exec-nostage-landscape"))
		promotion := createPromotion("exec-nostage-promotion", config.Name)

		By("resolution fails while the stage does not exist; the stage is not created")
		reconcilePromotion(promotion.Name)
		_, err := newExecutionReconciler().Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Name: promotion.Name, Namespace: testNamespace},
		})
		Expect(err).To(MatchError(ContainSubstring(`stage "late-stage" does not exist`)))

		refreshPromotion(promotion)
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStateBlocked))
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: "late-stage", Namespace: "kden-l-exec-nostage",
		}, &konfidence.Stage{})).To(MatchError(ContainSubstring("not found")))

		By("creating the stage lets the next reconcile promote")
		stage := createStage("kden-l-exec-nostage", "late-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		reconcilePromotion(promotion.Name)
		refreshPromotion(promotion)
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStateSucceeded))

		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: stage.Name, Namespace: stage.Namespace,
		}, stage)).To(Succeed())
		Expect(stage.Spec.Vector).To(Equal(testVector))

	})

	It("preserves the last successful conditions while a later promotion is blocked", func() {
		createLandscapeWithNamespace("exec-preserve-landscape", "kden-l-exec-preserve")
		createStage("kden-l-exec-preserve", "preserve-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		config := createConfig("exec-preserve-config",
			stageSource("other-stage"), stageTargetInLandscape("preserve-stage", "exec-preserve-landscape"))

		By("a first promotion succeeds")
		first := createPromotion("exec-preserve-a", config.Name)
		reconcilePromotion(first.Name)
		reconcilePromotion(first.Name)
		refreshPromotion(first)
		Expect(first.Status.State).To(Equal(konfidence.PromotionStateSucceeded))

		By("the target stage disappears and a second promotion blocks")
		stage := &konfidence.Stage{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "preserve-stage", Namespace: "kden-l-exec-preserve"}, stage)).To(Succeed())
		Expect(k8sClient.Delete(ctx, stage)).To(Succeed())
		second := createPromotion("exec-preserve-b", config.Name)
		reconcilePromotion(second.Name)
		_, err := newExecutionReconciler().Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Name: second.Name, Namespace: testNamespace},
		})
		Expect(err).To(HaveOccurred())

		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: config.Name, Namespace: testNamespace}, config)).To(Succeed())
		blockedCond := meta.FindStatusCondition(config.Status.LastPromotionConditions, konfidence.ConditionTypeSucceeded)
		Expect(blockedCond).NotTo(BeNil())
		Expect(blockedCond.Reason).To(Equal(konfidence.ReasonPromotionTargetUnresolved))
		successCond := meta.FindStatusCondition(config.Status.LastSuccessfulPromotionConditions, konfidence.ConditionTypeSucceeded)
		Expect(successCond).NotTo(BeNil())
		Expect(successCond.Status).To(Equal(metav1.ConditionTrue))
	})
	It("resumes an interrupted execution idempotently", func() {
		createLandscapeWithNamespace("exec-resume-landscape", "kden-l-exec-resume")
		stage := createStage("kden-l-exec-resume", "resume-stage", testVector)
		config := createConfig("exec-resume-config",
			templateSource("some-template"), stageTargetInLandscape("resume-stage", "exec-resume-landscape"))
		promotion := createPromotion("exec-resume-promotion", config.Name)

		By("simulating a crash after the Running patch and the stage write")
		reconcilePromotion(promotion.Name)
		setSucceededCondition(promotion, metav1.ConditionFalse, konfidence.ReasonPromotionRunning, metav1.Now().Time)

		By("the next reconcile completes without re-patching the stage")
		reconcilePromotion(promotion.Name)
		refreshPromotion(promotion)
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStateSucceeded))
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: stage.Name, Namespace: stage.Namespace,
		}, stage)).To(Succeed())
		Expect(stage.Spec.Vector).To(Equal(testVector))
	})
})
