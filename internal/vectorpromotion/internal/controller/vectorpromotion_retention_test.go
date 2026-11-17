package controller

import (
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
)

var _ = Describe("VectorPromotion self-retention", Ordered, Serial, func() {

	BeforeEach(func() { cleanupPromotions() })

	It("counts rejection as terminal and wakes older promotions to delete themselves", func() {
		config := createConfigWithRetention("retention-reject-config", 1)
		older := createPromotion("retention-reject-old", config.Name)
		setSucceededCondition(older, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, time.Now())
		newer := createPromotionRequiringApproval("retention-reject-new", config.Name)
		startWatchedExecutionManager()
		Eventually(func(g Gomega) {
			refreshPromotion(newer)
			g.Expect(promotion.IsWaiting(newer)).To(BeTrue())
			g.Expect(meta.FindStatusCondition(newer.Status.Conditions, konfidence.ConditionTypeGatesSatisfied)).NotTo(BeNil())
		}, timeout, interval).Should(Succeed())
		setOverride(newer, konfidence.PromotionOverrideTypeReject)
		// Less than the periodic fallback: deletion must be driven by fan-out
		// of the newer promotion's transition to terminal rejection.
		Eventually(func(g Gomega) {
			refreshPromotion(newer)
			g.Expect(observedState(newer)).To(Equal("Rejected"))
			g.Expect(apierrors.IsNotFound(k8sClient.Get(ctx, client.ObjectKeyFromObject(older), older))).To(BeTrue())
		}, 10*time.Second, interval).Should(Succeed())
	})

	It("fans out retention-limit changes to terminal promotions", func() {
		config := createConfigWithRetention("retention-config-watch", 3)
		var newest *konfidence.VectorPromotion
		for _, name := range []string{"retention-watch-a", "retention-watch-b", "retention-watch-c"} {
			newest = createPromotion(name, config.Name)
			setSucceededCondition(newest, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, time.Now())
		}
		startWatchedExecutionManager()
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(config), config)).To(Succeed())
			original := config.DeepCopy()
			keep := int32(1)
			config.Spec.KeepLastPromotions = &keep
			g.Expect(k8sClient.Patch(ctx, config, client.MergeFrom(original))).To(Succeed())
		}, timeout, interval).Should(Succeed())
		Eventually(func(g Gomega) {
			list := &konfidence.VectorPromotionList{}
			g.Expect(k8sClient.List(ctx, list, client.InNamespace(testNamespace),
				client.MatchingFields{promotionConfigNameField: config.Name})).To(Succeed())
			g.Expect(list.Items).To(HaveLen(1))
			g.Expect(list.Items[0].Name).To(Equal(newest.Name))
		}, 10*time.Second, interval).Should(Succeed())
	})

	It("retains terminal promotions within the count bound regardless of age", func() {
		promotion := createPromotion("retention-old-promotion", "retention-old-config")

		By("marking the promotion as succeeded")
		setSucceededCondition(promotion, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, time.Now().Add(-365*24*time.Hour))

		By("asserting the VectorPromotion persists")
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
		startWatchedExecutionManager()

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

	It("counts only eligible terminals and resumes cleanup after older active promotions finish", func() {
		config := createConfigWithRetention("retention-boundary-config", 1)
		oldest := createPromotion("retention-boundary-oldest", config.Name)
		eligible := createPromotion("retention-boundary-eligible", config.Name)
		active := createPromotionRequiringApproval("retention-boundary-active", config.Name)
		newest := createPromotion("retention-boundary-newest", config.Name)
		for _, p := range []*konfidence.VectorPromotion{oldest, eligible, newest} {
			setSucceededCondition(p, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, time.Now())
		}
		names := func(g Gomega) []string {
			list := &konfidence.VectorPromotionList{}
			g.Expect(k8sClient.List(ctx, list, client.InNamespace(testNamespace),
				client.MatchingFields{promotionConfigNameField: config.Name})).To(Succeed())
			result := make([]string, 0, len(list.Items))
			for _, p := range list.Items {
				result = append(result, p.Name)
			}
			return result
		}
		By("keeping one eligible terminal in addition to the deferred newer terminal")
		for _, p := range []*konfidence.VectorPromotion{oldest, eligible, newest} {
			reconcilePromotion(p.Name)
		}
		Eventually(func(g Gomega) {
			g.Expect(names(g)).To(ConsistOf(eligible.Name, active.Name, newest.Name))
		}, timeout, interval).Should(Succeed())
		By("letting the older active promotion supersede itself")
		startWatchedExecutionManager()
		By("applying the retention count to all terminals once none are active")
		Eventually(func(g Gomega) {
			g.Expect(names(g)).To(ConsistOf(newest.Name))
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

		By("newer promotions do not update the oldest promotion's lifecycle")
		reconcilePromotion(newest.Name)
		refreshPromotion(oldest)
		Expect(promotion.IsTerminal(oldest)).To(BeFalse())

		Eventually(func(g Gomega) {
			reconcilePromotion(oldest.Name)
			refreshPromotion(oldest)
			condition := meta.FindStatusCondition(oldest.Status.Conditions, konfidence.ConditionTypeSucceeded)
			g.Expect(observedState(oldest)).To(Equal("Superseded"))
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
			reconcilePromotion(oldest.Name)
			refreshPromotion(oldest)
			condition := meta.FindStatusCondition(oldest.Status.Conditions, konfidence.ConditionTypeSucceeded)
			g.Expect(observedState(oldest)).To(Equal("Superseded"))
			g.Expect(condition).NotTo(BeNil())
			g.Expect(condition.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(condition.Reason).To(Equal(konfidence.ReasonPromotionOutstandingLimitReached))
		}, timeout, interval).Should(Succeed())
	})

	It("should retain unlimited outstanding promotions when the bound is zero", func() {
		keep := int32(0)
		config := createConfig("outstanding-retention-unlimited-config",
			stageSource("missing-source-stage"), stageTarget("missing-target-stage"))
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, config)).To(Succeed())
			original := config.DeepCopy()
			config.Spec.KeepOutstandingPromotions = &keep
			g.Expect(k8sClient.Patch(ctx, config, client.MergeFrom(original))).To(Succeed())
		}, timeout, interval).Should(Succeed())

		promotions := []*konfidence.VectorPromotion{
			createPromotionRequiringApproval("outstanding-retention-unlimited-a", config.Name),
			createPromotionRequiringApproval("outstanding-retention-unlimited-b", config.Name),
			createPromotionRequiringApproval("outstanding-retention-unlimited-c", config.Name),
		}

		Consistently(func(g Gomega) {
			for _, vectorPromotion := range promotions {
				reconcilePromotion(vectorPromotion.Name)
				refreshPromotion(vectorPromotion)
				g.Expect(promotion.IsTerminal(vectorPromotion)).To(BeFalse())
			}
		}, 3*time.Second, interval).Should(Succeed())
	})

})
