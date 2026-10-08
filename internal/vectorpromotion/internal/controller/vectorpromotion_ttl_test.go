package controller

import (
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
)

var _ = Describe("VectorPromotion retention controller tests", Ordered, Serial, func() {

	BeforeEach(func() { cleanupPromotions() })

	It("should delete VectorPromotion after TTL expires (successful promotion)", func() {
		By("creating VectorPromotion with short TTL")
		promotion := createPromotionWithTTL("ttl-success-promotion", "ttl-success-config", 2*time.Second)

		By("marking the promotion as succeeded")
		setSucceededCondition(promotion, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, time.Now())

		By("asserting the VectorPromotion is eventually deleted")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)
			return errors.IsNotFound(err)
		}, 15*time.Second, interval).Should(BeTrue())
	})

	It("should delete VectorPromotion after TTL expires (failed promotion)", func() {
		By("creating VectorPromotion with short TTL")
		promotion := createPromotionWithTTL("ttl-failed-promotion", "ttl-failed-config", 2*time.Second)

		By("marking the promotion as failed")
		setSucceededCondition(promotion, metav1.ConditionFalse, konfidence.ReasonPromotionFailed, time.Now())

		By("asserting the VectorPromotion is eventually deleted")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)
			return errors.IsNotFound(err)
		}, 15*time.Second, interval).Should(BeTrue())
	})

	It("should not delete VectorPromotion when no TTL is configured", func() {
		By("creating VectorPromotion without TTL")
		promotion := createPromotion("ttl-none-promotion", "ttl-none-config")

		By("marking the promotion as succeeded")
		setSucceededCondition(promotion, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, time.Now())

		By("asserting the VectorPromotion persists")
		Consistently(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
		}, 5*time.Second, interval).Should(Succeed())
	})

	It("should not delete VectorPromotion when TTL has not yet expired", func() {
		By("creating VectorPromotion with long TTL")
		promotion := createPromotionWithTTL("ttl-long-promotion", "ttl-long-config", time.Hour)

		By("marking the promotion as succeeded")
		setSucceededCondition(promotion, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, time.Now())

		By("asserting the VectorPromotion persists (TTL not expired)")
		Consistently(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
		}, 5*time.Second, interval).Should(Succeed())
	})

	It("should reap terminal promotions beyond the config's retention bound", func() {
		By("creating a config that keeps a single terminal promotion")
		config := createConfigWithRetention("ttl-retention-config", 1)

		By("driving three promotions to terminal states")
		for _, name := range []string{"ttl-retention-a", "ttl-retention-b", "ttl-retention-c"} {
			promotion := createPromotion(name, config.Name)
			setSucceededCondition(promotion, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, time.Now())
		}

		By("asserting only the newest terminal promotion survives")
		Eventually(func(g Gomega) {
			promotions := &konfidence.VectorPromotionList{}
			g.Expect(k8sClient.List(ctx, promotions, client.InNamespace(testNamespace))).To(Succeed())
			names := []string{}
			for _, item := range promotions.Items {
				if item.Spec.VectorPromotionConfigName == config.Name {
					names = append(names, item.Name)
				}
			}
			g.Expect(names).To(ConsistOf("ttl-retention-c"))
		}, timeout, interval).Should(Succeed())
	})

	It("should supersede the oldest gated promotion beyond the outstanding retention bound", func() {
		keep := int32(2)
		config := createConfig("outstanding-retention-config",
			stageSource("missing-source-stage"), stageTarget("missing-target-stage"))
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, config)).To(Succeed())
			original := config.DeepCopy()
			config.Spec.KeepOutstandingPromotions = &keep
			g.Expect(k8sClient.Patch(ctx, config, client.MergeFrom(original))).To(Succeed())
		}, timeout, interval).Should(Succeed())

		oldest := createPromotionRequiringApproval("outstanding-retention-a", config.Name)
		middle := createPromotionRequiringApproval("outstanding-retention-b", config.Name)
		newest := createPromotionRequiringApproval("outstanding-retention-c", config.Name)

		Eventually(func(g Gomega) {
			refreshPromotion(oldest)
			condition := meta.FindStatusCondition(oldest.Status.Conditions, konfidence.ConditionTypeSucceeded)
			g.Expect(oldest.Status.State).To(Equal(konfidence.PromotionStateSuperseded))
			g.Expect(condition).NotTo(BeNil())
			g.Expect(condition.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(condition.Reason).To(Equal(konfidence.ReasonPromotionOutstandingLimitReached))
		}, timeout, interval).Should(Succeed())

		Consistently(func(g Gomega) {
			refreshPromotion(middle)
			refreshPromotion(newest)
			g.Expect(promotion.IsTerminal(middle)).To(BeFalse())
			g.Expect(promotion.IsTerminal(newest)).To(BeFalse())
		}, 3*time.Second, interval).Should(Succeed())
	})

	It("should apply a reduced outstanding retention bound to existing promotions", func() {
		config := createConfig("outstanding-retention-update-config",
			stageSource("missing-source-stage"), stageTarget("missing-target-stage"))
		oldest := createPromotionRequiringApproval("outstanding-retention-update-a", config.Name)
		createPromotionRequiringApproval("outstanding-retention-update-b", config.Name)
		createPromotionRequiringApproval("outstanding-retention-update-c", config.Name)

		keep := int32(2)
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, config)).To(Succeed())
			original := config.DeepCopy()
			config.Spec.KeepOutstandingPromotions = &keep
			g.Expect(k8sClient.Patch(ctx, config, client.MergeFrom(original))).To(Succeed())
		}, timeout, interval).Should(Succeed())

		Eventually(func(g Gomega) {
			refreshPromotion(oldest)
			condition := meta.FindStatusCondition(oldest.Status.Conditions, konfidence.ConditionTypeSucceeded)
			g.Expect(oldest.Status.State).To(Equal(konfidence.PromotionStateSuperseded))
			g.Expect(condition).NotTo(BeNil())
			g.Expect(condition.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(condition.Reason).To(Equal(konfidence.ReasonPromotionOutstandingLimitReached))
		}, timeout, interval).Should(Succeed())
	})

	It("should delete VectorPromotion when TTL is added via spec update after promotion completes", func() {
		By("creating VectorPromotion without TTL")
		promotion := createPromotion("ttl-patch-promotion", "ttl-patch-config")

		By("marking the promotion as succeeded")
		setSucceededCondition(promotion, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, time.Now())

		By("patching TTLAfterFinished onto the completed promotion")
		patch := client.MergeFrom(promotion.DeepCopy())
		promotion.Spec.TTLAfterFinished = &metav1.Duration{Duration: 2 * time.Second}
		Expect(k8sClient.Patch(ctx, promotion, patch)).To(Succeed())

		By("asserting the VectorPromotion is eventually deleted")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)
			return errors.IsNotFound(err)
		}, 15*time.Second, interval).Should(BeTrue())
	})
})
