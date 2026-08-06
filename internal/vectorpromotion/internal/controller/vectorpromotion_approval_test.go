package controller

import (
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("VectorPromotion approval phase", Ordered, Serial, func() {

	BeforeEach(func() { cleanupPromotions() })

	It("waits for approval and promotes once approved", func() {
		createLandscapeWithNamespace("exec-manual-landscape", "kden-l-exec-manual")
		stage := createStage("kden-l-exec-manual", "manual-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		config := createConfig("exec-manual-config",
			stageSource("other-stage"), stageTargetInLandscape("manual-stage", "exec-manual-landscape"))
		promotion := createPromotionTargeting("exec-manual-promotion", config.Name,
			stageTargetInLandscape("manual-stage", "exec-manual-landscape"), true)

		By("first reconcile parks the promotion in WaitingForApproval")
		reconcilePromotion(promotion.Name)
		refreshPromotion(promotion)
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStateWaitingForApproval))

		By("reconciling again without approval does not execute")
		reconcilePromotion(promotion.Name)
		refreshPromotion(promotion)
		Expect(meta.FindStatusCondition(promotion.Status.Conditions, konfidence.ConditionTypeSucceeded)).To(BeNil())

		By("approving executes the promotion")
		approvePromotion(promotion)
		reconcilePromotion(promotion.Name)
		refreshPromotion(promotion)
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStateSucceeded))

		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: stage.Name, Namespace: stage.Namespace,
		}, stage)).To(Succeed())
		Expect(stage.Spec.Vector).To(Equal(testVector))
	})
	It("does not touch a terminal promotion", func() {
		promotion := createPromotion("exec-terminal-promotion", "some-config")
		setSucceededCondition(promotion, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, metav1.Now().Time)
		refreshPromotion(promotion)
		resourceVersion := promotion.ResourceVersion

		reconcilePromotion(promotion.Name)
		refreshPromotion(promotion)
		Expect(promotion.ResourceVersion).To(Equal(resourceVersion))
	})
})
