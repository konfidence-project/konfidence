package controller

import (
	"fmt"
	"time"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("VectorPromotion TTL controller tests", Ordered, Serial, func() {

	BeforeEach(func() { cleanupPromotions() })

	It("should delete VectorPromotion after TTL expires (successful promotion)", func() {
		By("pushing component to source registry")
		ref := sourceRef("konfidence.io/promo/ttl-success:v1.0.0")
		pushComponent(ctx, ref, new("latest"))

		By("creating VectorPromotionConfig")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/ttl-success:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/ttl-success:promoted", targetRegistryEndpoint)
		config := createConfig("ttl-success-config", source, target)

		By("creating VectorPromotion with short TTL")
		promotion := createPromotionWithTTL("ttl-success-promotion", config.Name, 2*time.Second)

		By("asserting the VectorPromotion is eventually deleted")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)
			return errors.IsNotFound(err)
		}, 15*time.Second, interval).Should(BeTrue())
	})

	It("should delete VectorPromotion after TTL expires (failed promotion)", func() {
		By("creating VectorPromotion referencing non-existent config with short TTL")
		promotion := createPromotionWithTTL("ttl-failed-promotion", "non-existent-config", 2*time.Second)

		By("asserting the VectorPromotion is eventually deleted")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)
			return errors.IsNotFound(err)
		}, 15*time.Second, interval).Should(BeTrue())
	})

	It("should not delete VectorPromotion when no TTL is configured", func() {
		By("pushing component to source registry")
		ref := sourceRef("konfidence.io/promo/ttl-none:v1.0.0")
		pushComponent(ctx, ref, new("latest"))

		By("creating VectorPromotionConfig")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/ttl-none:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/ttl-none:promoted", targetRegistryEndpoint)
		config := createConfig("ttl-none-config", source, target)

		By("creating VectorPromotion without TTL")
		promotion := createPromotion("ttl-none-promotion", config.Name)

		By("waiting for promotion to complete")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, galaxy.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		}, timeout, interval).Should(Succeed())

		By("asserting the VectorPromotion persists")
		Consistently(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
		}, 5*time.Second, interval).Should(Succeed())
	})

	It("should not delete VectorPromotion when TTL has not yet expired", func() {
		By("pushing component to source registry")
		ref := sourceRef("konfidence.io/promo/ttl-long:v1.0.0")
		pushComponent(ctx, ref, new("latest"))

		By("creating VectorPromotionConfig")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/ttl-long:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/ttl-long:promoted", targetRegistryEndpoint)
		config := createConfig("ttl-long-config", source, target)

		By("creating VectorPromotion with long TTL")
		promotion := createPromotionWithTTL("ttl-long-promotion", config.Name, time.Hour)

		By("waiting for promotion to complete")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, galaxy.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		}, timeout, interval).Should(Succeed())

		By("asserting the VectorPromotion persists (TTL not expired)")
		Consistently(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
		}, 5*time.Second, interval).Should(Succeed())
	})

	It("should delete VectorPromotion when TTL is added via spec update after promotion completes", func() {
		By("pushing component to source registry")
		ref := sourceRef("konfidence.io/promo/ttl-patch:v1.0.0")
		pushComponent(ctx, ref, new("latest"))

		By("creating VectorPromotionConfig")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/ttl-patch:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/ttl-patch:promoted", targetRegistryEndpoint)
		config := createConfig("ttl-patch-config", source, target)

		By("creating VectorPromotion without TTL")
		promotion := createPromotion("ttl-patch-promotion", config.Name)

		By("waiting for promotion to complete")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, galaxy.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		}, timeout, interval).Should(Succeed())

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
