package controller

import (
	"context"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/pkg/promotiongate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// The shared manager deliberately omits execution for manually-driven tests.
// These specs start it separately to exercise the real informer/queue wiring.
func startWatchedExecutionManager() {
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme, Metrics: metricsserver.Options{BindAddress: "0"},
		// Specs restart the same controller sequentially within this process.
		Controller: config.Controller{SkipNameValidation: ptr.To(true)},
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(RegisterFieldIndexes(ctx, mgr)).To(Succeed())
	Expect(NewVectorPromotionReconciler(mgr).SetupWithManager(mgr)).To(Succeed())
	managerCtx, stop := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- mgr.Start(managerCtx) }()
	DeferCleanup(func() {
		stop()
		Eventually(done, timeout).Should(Receive(BeNil()))
	})
	Expect(mgr.GetCache().WaitForCacheSync(managerCtx)).To(BeTrue())
}

var _ = Describe("VectorPromotion sibling watches", Ordered, Serial, func() {
	BeforeEach(func() { cleanupPromotions() })

	It("wakes older waiting promotions when a newer sibling executes", func() {
		createLandscapeWithNamespace("watch-exec-landscape", "kden-l-watch-exec")
		createStage("kden-l-watch-exec", "target", "registry.example//konfidence.io/promo/app:0.9.0")
		config := createConfig("watch-exec-config", stageSource("missing-source"), stageTargetInLandscape("target", "watch-exec-landscape"))
		older := createPromotionTargeting("watch-exec-old", config.Name, config.Spec.Target, true)
		startWatchedExecutionManager()
		Eventually(func(g Gomega) {
			refreshPromotion(older)
			g.Expect(meta.FindStatusCondition(older.Status.Conditions, konfidence.ConditionTypeGatesSatisfied)).NotTo(BeNil())
		}, timeout, interval).Should(Succeed())

		newer := createPromotionTargeting("watch-exec-new", config.Name, config.Spec.Target, true)
		Eventually(func(g Gomega) {
			g.Expect(promotiongate.Set(ctx, k8sClient, client.ObjectKeyFromObject(newer), promotiongate.Update{
				Type: konfidence.GateTypeExplicitApproval, State: konfidence.PromotionGateStateSatisfied,
				Reason: konfidence.ReasonGateApprovedByUser, SetBy: "test",
			})).To(Succeed())
		}, timeout, interval).Should(Succeed())
		// Less than the periodic resync: this must be delivered by sibling events.
		Eventually(func(g Gomega) {
			refreshPromotion(newer)
			refreshPromotion(older)
			g.Expect(observedState(newer)).To(Equal("Succeeded"))
			g.Expect(observedState(older)).To(Equal("Superseded"))
		}, 10*time.Second, interval).Should(Succeed())
	})

	It("fans out config limit changes to every affected waiting promotion", func() {
		config := createConfig("watch-limit-config", stageSource("missing-source"), stageTarget("missing-target"))
		oldest := createPromotionRequiringApproval("watch-limit-old", config.Name)
		middle := createPromotionRequiringApproval("watch-limit-middle", config.Name)
		newest := createPromotionRequiringApproval("watch-limit-new", config.Name)
		startWatchedExecutionManager()
		Eventually(func(g Gomega) {
			for _, p := range []*konfidence.VectorPromotion{oldest, middle, newest} {
				refreshPromotion(p)
				g.Expect(meta.FindStatusCondition(p.Status.Conditions, konfidence.ConditionTypeGatesSatisfied)).NotTo(BeNil())
			}
		}, timeout, interval).Should(Succeed())

		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(config), config)).To(Succeed())
			original := config.DeepCopy()
			keep := int32(1)
			config.Spec.KeepOutstandingPromotions = &keep
			g.Expect(k8sClient.Patch(ctx, config, client.MergeFrom(original))).To(Succeed())
		}, timeout, interval).Should(Succeed())
		Eventually(func(g Gomega) {
			for _, p := range []*konfidence.VectorPromotion{oldest, middle} {
				refreshPromotion(p)
				condition := meta.FindStatusCondition(p.Status.Conditions, konfidence.ConditionTypeSucceeded)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Reason).To(Equal(konfidence.ReasonPromotionOutstandingLimitReached))
			}
			refreshPromotion(newest)
			g.Expect(observedState(newest)).To(Equal("Waiting"))
		}, 10*time.Second, interval).Should(Succeed())
	})

	It("wakes older promotions when creation exceeds the outstanding limit", func() {
		config := createConfig("watch-create-config", stageSource("missing-source"), stageTarget("missing-target"))
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(config), config)).To(Succeed())
			original := config.DeepCopy()
			keep := int32(1)
			config.Spec.KeepOutstandingPromotions = &keep
			g.Expect(k8sClient.Patch(ctx, config, client.MergeFrom(original))).To(Succeed())
		}, timeout, interval).Should(Succeed())
		older := createPromotionRequiringApproval("watch-create-old", config.Name)
		startWatchedExecutionManager()
		Eventually(func(g Gomega) {
			refreshPromotion(older)
			g.Expect(meta.FindStatusCondition(older.Status.Conditions, konfidence.ConditionTypeGatesSatisfied)).NotTo(BeNil())
		}, timeout, interval).Should(Succeed())
		newer := createPromotionRequiringApproval("watch-create-new", config.Name)
		Eventually(func(g Gomega) {
			refreshPromotion(older)
			refreshPromotion(newer)
			condition := meta.FindStatusCondition(older.Status.Conditions, konfidence.ConditionTypeSucceeded)
			g.Expect(condition).NotTo(BeNil())
			g.Expect(condition.Reason).To(Equal(konfidence.ReasonPromotionOutstandingLimitReached))
			g.Expect(observedState(newer)).To(Equal("Waiting"))
		}, 10*time.Second, interval).Should(Succeed())
	})
})
