package controller

import (
	"fmt"
	"time"

	global "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("VectorPromotion status propagation controller tests", Ordered, Serial, func() {

	BeforeEach(func() {
		Expect(k8sClient.DeleteAllOf(ctx, &global.VectorPromotion{}, client.InNamespace(testNamespace))).To(Succeed())
		Expect(k8sClient.DeleteAllOf(ctx, &global.VectorPromotionConfig{}, client.InNamespace(testNamespace))).To(Succeed())
		Eventually(func(g Gomega) {
			list := &global.VectorPromotionList{}
			g.Expect(k8sClient.List(ctx, list, client.InNamespace(testNamespace))).To(Succeed())
			g.Expect(list.Items).To(BeEmpty())
		}, timeout, interval).Should(Succeed())
	})

	It("should propagate successful promotion conditions to config", func() {
		By("pushing component to source registry")
		ref := sourceRef("konfidence.io/promo/sp-success:v1.0.0")
		pushComponent(ctx, ref, new("latest"))

		By("creating VectorPromotionConfig")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/sp-success:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/sp-success:promoted", targetRegistryEndpoint)
		config := createConfig("sp-success-config", source, target)

		By("creating VectorPromotion")
		createPromotion("sp-success-promotion", config.Name)

		By("asserting config has both LastPromotionConditions and LastSuccessfulPromotionConditions")
		Eventually(func(g Gomega) {
			updatedConfig := &global.VectorPromotionConfig{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, updatedConfig)).To(Succeed())
			g.Expect(updatedConfig.Status.LastPromotionConditions).NotTo(BeEmpty())
			readyCond := meta.FindStatusCondition(updatedConfig.Status.LastPromotionConditions, global.ConditionTypeSucceeded)
			g.Expect(readyCond).NotTo(BeNil())
			g.Expect(readyCond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(readyCond.Reason).To(Equal(global.ReasonPromotionSucceeded))
			g.Expect(updatedConfig.Status.LastSuccessfulPromotionConditions).NotTo(BeEmpty())
			successCond := meta.FindStatusCondition(
				updatedConfig.Status.LastSuccessfulPromotionConditions, global.ConditionTypeSucceeded)
			g.Expect(successCond).NotTo(BeNil())
			g.Expect(successCond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(successCond.Reason).To(Equal(global.ReasonPromotionSucceeded))
		}, timeout, interval).Should(Succeed())
	})

	It("should propagate failed promotion conditions without setting LastSuccessfulPromotionConditions", func() {
		By("creating VectorPromotionConfig pointing to non-existent source")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/sp-failed-nonexistent:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/sp-failed-nonexistent:promoted", targetRegistryEndpoint)
		config := createConfig("sp-failed-config", source, target)

		By("creating VectorPromotion")
		createPromotion("sp-failed-promotion", config.Name)

		By("asserting config has LastPromotionConditions with failure but no LastSuccessfulPromotionConditions")
		Eventually(func(g Gomega) {
			updatedConfig := &global.VectorPromotionConfig{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, updatedConfig)).To(Succeed())
			g.Expect(updatedConfig.Status.LastPromotionConditions).NotTo(BeEmpty())
			readyCond := meta.FindStatusCondition(updatedConfig.Status.LastPromotionConditions, global.ConditionTypeSucceeded)
			g.Expect(readyCond).NotTo(BeNil())
			g.Expect(readyCond.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(readyCond.Reason).To(Equal(global.ReasonPromotionSourceNotFound))
			g.Expect(updatedConfig.Status.LastSuccessfulPromotionConditions).To(BeEmpty())
		}, timeout, interval).Should(Succeed())
	})

	It("should update config conditions with the latest successful promotion", func() {
		By("pushing component v1 to source registry")
		ref := sourceRef("konfidence.io/promo/sp-seq:v1.0.0")
		pushComponent(ctx, ref, new("latest"))

		By("creating VectorPromotionConfig")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/sp-seq:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/sp-seq:promoted", targetRegistryEndpoint)
		config := createConfig("sp-sequential-config", source, target)

		By("creating first VectorPromotion")
		createPromotion("sp-sequential-first", config.Name)

		By("waiting for config to be updated with first promotion conditions")
		var firstTransitionTime metav1.Time
		Eventually(func(g Gomega) {
			updatedConfig := &global.VectorPromotionConfig{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, updatedConfig)).To(Succeed())
			g.Expect(updatedConfig.Status.LastSuccessfulPromotionConditions).NotTo(BeEmpty())
			cond := meta.FindStatusCondition(
				updatedConfig.Status.LastSuccessfulPromotionConditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			firstTransitionTime = cond.LastTransitionTime
		}, timeout, interval).Should(Succeed())

		By("pushing component v2 and creating second VectorPromotion")
		ref2 := sourceRef("konfidence.io/promo/sp-seq:v2.0.0")
		pushComponent(ctx, ref2, new("latest"))
		createPromotion("sp-sequential-second", config.Name)

		By("waiting for config to be updated with second promotion conditions")
		Eventually(func(g Gomega) {
			updatedConfig := &global.VectorPromotionConfig{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, updatedConfig)).To(Succeed())
			g.Expect(updatedConfig.Status.LastSuccessfulPromotionConditions).NotTo(BeEmpty())
			cond := meta.FindStatusCondition(
				updatedConfig.Status.LastSuccessfulPromotionConditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(cond.LastTransitionTime.Time.After(firstTransitionTime.Time)).To(BeTrue(),
				"second promotion's transition time should be after first")
		}, timeout, interval).Should(Succeed())
	})

	It("should not overwrite config conditions when config has more recent timestamp", func() {
		By("pushing component to source registry")
		ref := sourceRef("konfidence.io/promo/sp-dedup:v1.0.0")
		pushComponent(ctx, ref, new("latest"))

		By("creating VectorPromotionConfig")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/sp-dedup:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/sp-dedup:promoted", targetRegistryEndpoint)
		config := createConfig("sp-dedup-config", source, target)

		By("creating first VectorPromotion and waiting for propagation")
		createPromotion("sp-dedup-first", config.Name)
		Eventually(func(g Gomega) {
			updatedConfig := &global.VectorPromotionConfig{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, updatedConfig)).To(Succeed())
			g.Expect(updatedConfig.Status.LastPromotionConditions).NotTo(BeEmpty())
			cond := meta.FindStatusCondition(updatedConfig.Status.LastPromotionConditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		}, timeout, interval).Should(Succeed())

		By("patching config status with a future timestamp to simulate newer conditions")
		updatedConfig := &global.VectorPromotionConfig{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: config.Name, Namespace: testNamespace,
		}, updatedConfig)).To(Succeed())
		originalConfig := updatedConfig.DeepCopy()
		futureTime := metav1.NewTime(time.Now().Add(1 * time.Hour).Truncate(time.Second))
		updatedConfig.Status.LastPromotionConditions = []metav1.Condition{
			{
				Type:               global.ConditionTypeSucceeded,
				Status:             metav1.ConditionTrue,
				Reason:             "FuturePromotion",
				Message:            "Simulated future promotion",
				LastTransitionTime: futureTime,
			},
		}
		Expect(k8sClient.Status().Patch(ctx, updatedConfig, client.MergeFrom(originalConfig))).To(Succeed())

		By("creating a second VectorPromotion that will also succeed")
		ref2 := sourceRef("konfidence.io/promo/sp-dedup:v2.0.0")
		pushComponent(ctx, ref2, new("latest"))
		promotion2 := createPromotion("sp-dedup-second", config.Name)

		By("waiting for second promotion to reach terminal state")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion2.Name, Namespace: testNamespace,
			}, promotion2)).To(Succeed())
			cond := meta.FindStatusCondition(promotion2.Status.Conditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		}, timeout, interval).Should(Succeed())

		By("asserting config conditions were NOT overwritten (future timestamp preserved)")
		Consistently(func(g Gomega) {
			latestConfig := &global.VectorPromotionConfig{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, latestConfig)).To(Succeed())
			g.Expect(latestConfig.Status.LastPromotionConditions).NotTo(BeEmpty())
			cond := meta.FindStatusCondition(latestConfig.Status.LastPromotionConditions, global.ConditionTypeSucceeded)
			g.Expect(cond.Reason).To(Equal("FuturePromotion"),
				"config should still have the manually patched future conditions")
			g.Expect(cond.LastTransitionTime.Time).To(Equal(futureTime.Time))
		}, 15*time.Second, interval).Should(Succeed())
	})

	It("should preserve LastSuccessfulPromotionConditions when a subsequent promotion fails", func() {
		By("creating VectorPromotionConfig with non-existent source")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/sp-preserve-nonexistent:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/sp-preserve-nonexistent:promoted", targetRegistryEndpoint)
		config := createConfig("sp-preserve-config", source, target)

		By("manually setting LastSuccessfulPromotionConditions to simulate prior success")
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: config.Name, Namespace: testNamespace,
		}, config)).To(Succeed())
		originalConfig := config.DeepCopy()
		priorSuccessTime := metav1.NewTime(time.Now().Add(-1 * time.Hour).Truncate(time.Second))
		config.Status.LastSuccessfulPromotionConditions = []metav1.Condition{
			{
				Type:               global.ConditionTypeSucceeded,
				Status:             metav1.ConditionTrue,
				Reason:             global.ReasonPromotionSucceeded,
				Message:            "Prior successful promotion",
				LastTransitionTime: priorSuccessTime,
			},
		}
		Expect(k8sClient.Status().Patch(ctx, config, client.MergeFrom(originalConfig))).To(Succeed())

		By("creating VectorPromotion that will fail (source not found)")
		createPromotion("sp-preserve-promotion", config.Name)

		By("asserting config has failure in LastPromotionConditions but preserves LastSuccessfulPromotionConditions")
		Eventually(func(g Gomega) {
			updatedConfig := &global.VectorPromotionConfig{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, updatedConfig)).To(Succeed())
			g.Expect(updatedConfig.Status.LastPromotionConditions).NotTo(BeEmpty())
			lastCond := meta.FindStatusCondition(updatedConfig.Status.LastPromotionConditions, global.ConditionTypeSucceeded)
			g.Expect(lastCond).NotTo(BeNil())
			g.Expect(lastCond.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(lastCond.Reason).To(Equal(global.ReasonPromotionSourceNotFound))
			g.Expect(updatedConfig.Status.LastSuccessfulPromotionConditions).NotTo(BeEmpty())
			successCond := meta.FindStatusCondition(
				updatedConfig.Status.LastSuccessfulPromotionConditions, global.ConditionTypeSucceeded)
			g.Expect(successCond).NotTo(BeNil())
			g.Expect(successCond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(successCond.Reason).To(Equal(global.ReasonPromotionSucceeded))
			g.Expect(successCond.LastTransitionTime.Time).To(Equal(priorSuccessTime.Time))
		}, timeout, interval).Should(Succeed())
	})

	It("should propagate status when config is created after promotion completes", func() {
		By("creating VectorPromotion referencing a config that does not exist yet")
		promotion := createPromotion("sp-delayed-promotion", "sp-delayed-config")

		By("waiting for promotion to fail with config not found")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(cond.Reason).To(Equal(global.ReasonPromotionConfigurationNotFound))
		}, timeout, interval).Should(Succeed())

		By("creating VectorPromotionConfig after promotion has already completed")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/sp-delayed:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/sp-delayed:promoted", targetRegistryEndpoint)
		config := createConfig("sp-delayed-config", source, target)

		By("asserting status propagation controller retries and eventually propagates conditions to the config")
		Eventually(func(g Gomega) {
			updatedConfig := &global.VectorPromotionConfig{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, updatedConfig)).To(Succeed())
			g.Expect(updatedConfig.Status.LastPromotionConditions).NotTo(BeEmpty())
			cond := meta.FindStatusCondition(updatedConfig.Status.LastPromotionConditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(cond.Reason).To(Equal(global.ReasonPromotionConfigurationNotFound))
		}, 30*time.Second, interval).Should(Succeed())
	})
})
