package controller

import (
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("VectorPromotion status propagation controller tests", Ordered, Serial, func() {

	BeforeEach(func() { cleanupPromotions() })

	It("should propagate successful promotion conditions to config", func() {
		By("creating VectorPromotionConfig")
		config := createConfig("sp-success-config", templateSource("sp-template"), stageTarget("sp-stage"))

		By("creating VectorPromotion and marking it succeeded")
		promotion := createPromotion("sp-success-promotion", config.Name)
		setSucceededCondition(promotion, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, time.Now())

		By("asserting config has both LastPromotionConditions and LastSuccessfulPromotionConditions")
		Eventually(func(g Gomega) {
			updatedConfig := &konfidence.VectorPromotionConfig{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, updatedConfig)).To(Succeed())
			g.Expect(updatedConfig.Status.LastPromotionConditions).NotTo(BeEmpty())
			readyCond := meta.FindStatusCondition(updatedConfig.Status.LastPromotionConditions, konfidence.ConditionTypeSucceeded)
			g.Expect(readyCond).NotTo(BeNil())
			g.Expect(readyCond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(readyCond.Reason).To(Equal(konfidence.ReasonPromotionSucceeded))
			g.Expect(updatedConfig.Status.LastSuccessfulPromotionConditions).NotTo(BeEmpty())
			successCond := meta.FindStatusCondition(
				updatedConfig.Status.LastSuccessfulPromotionConditions, konfidence.ConditionTypeSucceeded)
			g.Expect(successCond).NotTo(BeNil())
			g.Expect(successCond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(successCond.Reason).To(Equal(konfidence.ReasonPromotionSucceeded))
		}, timeout, interval).Should(Succeed())
	})

	It("should propagate failed promotion conditions without setting LastSuccessfulPromotionConditions", func() {
		By("creating VectorPromotionConfig")
		config := createConfig("sp-failed-config", templateSource("sp-template"), stageTarget("sp-stage"))

		By("creating VectorPromotion and marking it failed")
		promotion := createPromotion("sp-failed-promotion", config.Name)
		setSucceededCondition(promotion, metav1.ConditionFalse, konfidence.ReasonPromotionFailed, time.Now())

		By("asserting config has LastPromotionConditions with failure but no LastSuccessfulPromotionConditions")
		Eventually(func(g Gomega) {
			updatedConfig := &konfidence.VectorPromotionConfig{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, updatedConfig)).To(Succeed())
			g.Expect(updatedConfig.Status.LastPromotionConditions).NotTo(BeEmpty())
			readyCond := meta.FindStatusCondition(updatedConfig.Status.LastPromotionConditions, konfidence.ConditionTypeSucceeded)
			g.Expect(readyCond).NotTo(BeNil())
			g.Expect(readyCond.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(readyCond.Reason).To(Equal(konfidence.ReasonPromotionFailed))
			g.Expect(updatedConfig.Status.LastSuccessfulPromotionConditions).To(BeEmpty())
		}, timeout, interval).Should(Succeed())
	})

	It("should update config conditions with the latest successful promotion", func() {
		By("creating VectorPromotionConfig")
		config := createConfig("sp-sequential-config", templateSource("sp-template"), stageTarget("sp-stage"))

		By("creating first VectorPromotion and marking it succeeded")
		first := createPromotion("sp-sequential-first", config.Name)
		setSucceededCondition(first, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, time.Now())

		By("waiting for config to be updated with first promotion conditions")
		var firstTransitionTime metav1.Time
		Eventually(func(g Gomega) {
			updatedConfig := &konfidence.VectorPromotionConfig{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, updatedConfig)).To(Succeed())
			g.Expect(updatedConfig.Status.LastSuccessfulPromotionConditions).NotTo(BeEmpty())
			cond := meta.FindStatusCondition(
				updatedConfig.Status.LastSuccessfulPromotionConditions, konfidence.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			firstTransitionTime = cond.LastTransitionTime
		}, timeout, interval).Should(Succeed())

		By("creating second VectorPromotion succeeding strictly later")
		second := createPromotion("sp-sequential-second", config.Name)
		setSucceededCondition(second, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded,
			firstTransitionTime.Add(2*time.Second))

		By("waiting for config to be updated with second promotion conditions")
		Eventually(func(g Gomega) {
			updatedConfig := &konfidence.VectorPromotionConfig{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, updatedConfig)).To(Succeed())
			g.Expect(updatedConfig.Status.LastSuccessfulPromotionConditions).NotTo(BeEmpty())
			cond := meta.FindStatusCondition(
				updatedConfig.Status.LastSuccessfulPromotionConditions, konfidence.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(cond.LastTransitionTime.Time.After(firstTransitionTime.Time)).To(BeTrue(),
				"second promotion's transition time should be after first")
		}, timeout, interval).Should(Succeed())
	})

	It("should not overwrite config conditions when config has more recent timestamp", func() {
		By("creating VectorPromotionConfig")
		config := createConfig("sp-dedup-config", templateSource("sp-template"), stageTarget("sp-stage"))

		By("patching config status with a future timestamp to simulate newer conditions")
		originalConfig := config.DeepCopy()
		futureTime := metav1.NewTime(time.Now().Add(1 * time.Hour).Truncate(time.Second))
		config.Status.LastPromotionConditions = []metav1.Condition{
			{
				Type:               konfidence.ConditionTypeSucceeded,
				Status:             metav1.ConditionTrue,
				Reason:             "FuturePromotion",
				Message:            "Simulated future promotion",
				LastTransitionTime: futureTime,
			},
		}
		Expect(k8sClient.Status().Patch(ctx, config, client.MergeFrom(originalConfig))).To(Succeed())

		By("creating a VectorPromotion succeeding now (older than the config's conditions)")
		promotion := createPromotion("sp-dedup-promotion", config.Name)
		setSucceededCondition(promotion, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, time.Now())

		By("asserting config conditions were NOT overwritten (future timestamp preserved)")
		Consistently(func(g Gomega) {
			latestConfig := &konfidence.VectorPromotionConfig{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, latestConfig)).To(Succeed())
			g.Expect(latestConfig.Status.LastPromotionConditions).NotTo(BeEmpty())
			cond := meta.FindStatusCondition(latestConfig.Status.LastPromotionConditions, konfidence.ConditionTypeSucceeded)
			g.Expect(cond.Reason).To(Equal("FuturePromotion"),
				"config should still have the manually patched future conditions")
			g.Expect(cond.LastTransitionTime.Time).To(Equal(futureTime.Time))
		}, 15*time.Second, interval).Should(Succeed())
	})

	It("should preserve LastSuccessfulPromotionConditions when a subsequent promotion fails", func() {
		By("creating VectorPromotionConfig")
		config := createConfig("sp-preserve-config", templateSource("sp-template"), stageTarget("sp-stage"))

		By("manually setting LastSuccessfulPromotionConditions to simulate prior success")
		originalConfig := config.DeepCopy()
		priorSuccessTime := metav1.NewTime(time.Now().Add(-1 * time.Hour).Truncate(time.Second))
		config.Status.LastSuccessfulPromotionConditions = []metav1.Condition{
			{
				Type:               konfidence.ConditionTypeSucceeded,
				Status:             metav1.ConditionTrue,
				Reason:             konfidence.ReasonPromotionSucceeded,
				Message:            "Prior successful promotion",
				LastTransitionTime: priorSuccessTime,
			},
		}
		Expect(k8sClient.Status().Patch(ctx, config, client.MergeFrom(originalConfig))).To(Succeed())

		By("creating VectorPromotion and marking it failed")
		promotion := createPromotion("sp-preserve-promotion", config.Name)
		setSucceededCondition(promotion, metav1.ConditionFalse, konfidence.ReasonPromotionSourceNotFound, time.Now())

		By("asserting config has failure in LastPromotionConditions but preserves LastSuccessfulPromotionConditions")
		Eventually(func(g Gomega) {
			updatedConfig := &konfidence.VectorPromotionConfig{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, updatedConfig)).To(Succeed())
			g.Expect(updatedConfig.Status.LastPromotionConditions).NotTo(BeEmpty())
			lastCond := meta.FindStatusCondition(updatedConfig.Status.LastPromotionConditions, konfidence.ConditionTypeSucceeded)
			g.Expect(lastCond).NotTo(BeNil())
			g.Expect(lastCond.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(lastCond.Reason).To(Equal(konfidence.ReasonPromotionSourceNotFound))
			g.Expect(updatedConfig.Status.LastSuccessfulPromotionConditions).NotTo(BeEmpty())
			successCond := meta.FindStatusCondition(
				updatedConfig.Status.LastSuccessfulPromotionConditions, konfidence.ConditionTypeSucceeded)
			g.Expect(successCond).NotTo(BeNil())
			g.Expect(successCond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(successCond.Reason).To(Equal(konfidence.ReasonPromotionSucceeded))
			g.Expect(successCond.LastTransitionTime.Time).To(Equal(priorSuccessTime.Time))
		}, timeout, interval).Should(Succeed())
	})

	It("should propagate reason-only transitions that keep the transition timestamp", func() {
		By("creating VectorPromotionConfig and a running promotion")
		config := createConfig("sp-reason-config", templateSource("sp-template"), stageTarget("sp-stage"))
		promotion := createPromotion("sp-reason-promotion", config.Name)
		transitionTime := time.Now()
		setSucceededCondition(promotion, metav1.ConditionFalse, konfidence.ReasonPromotionRunning, transitionTime)

		By("waiting for the running conditions to reach the config")
		Eventually(func(g Gomega) {
			updatedConfig := &konfidence.VectorPromotionConfig{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, updatedConfig)).To(Succeed())
			cond := meta.FindStatusCondition(updatedConfig.Status.LastPromotionConditions, konfidence.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Reason).To(Equal(konfidence.ReasonPromotionRunning))
		}, timeout, interval).Should(Succeed())

		By("terminating with the same status and transition timestamp, only the reason changes")
		setSucceededCondition(promotion, metav1.ConditionFalse, konfidence.ReasonPromotionSuperseded, transitionTime)

		By("asserting the terminal reason still reaches the config")
		Eventually(func(g Gomega) {
			updatedConfig := &konfidence.VectorPromotionConfig{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, updatedConfig)).To(Succeed())
			cond := meta.FindStatusCondition(updatedConfig.Status.LastPromotionConditions, konfidence.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Reason).To(Equal(konfidence.ReasonPromotionSuperseded))
		}, timeout, interval).Should(Succeed())
	})

	It("should propagate status when config is created after promotion completes", func() {
		By("creating VectorPromotion referencing a config that does not exist yet")
		promotion := createPromotion("sp-delayed-promotion", "sp-delayed-config")

		By("marking the promotion as failed with config not found")
		setSucceededCondition(promotion, metav1.ConditionFalse, konfidence.ReasonPromotionConfigurationNotFound, time.Now())

		By("creating VectorPromotionConfig after promotion has already completed")
		config := createConfig("sp-delayed-config", templateSource("sp-template"), stageTarget("sp-stage"))

		By("asserting status propagation controller retries and eventually propagates conditions to the config")
		Eventually(func(g Gomega) {
			updatedConfig := &konfidence.VectorPromotionConfig{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, updatedConfig)).To(Succeed())
			g.Expect(updatedConfig.Status.LastPromotionConditions).NotTo(BeEmpty())
			cond := meta.FindStatusCondition(updatedConfig.Status.LastPromotionConditions, konfidence.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(cond.Reason).To(Equal(konfidence.ReasonPromotionConfigurationNotFound))
		}, 30*time.Second, interval).Should(Succeed())
	})
})
