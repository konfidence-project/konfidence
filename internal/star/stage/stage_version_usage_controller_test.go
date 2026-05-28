package stage_test

import (
	"context"
	"time"

	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/star/stage"
	testutil "github.com/konfidence-project/konfidence/internal/star/stage/internal/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("StageVersionUsage Controller", Ordered, func() {
	var (
		k8sClient client.Client
		cancel    context.CancelFunc
	)

	BeforeAll(func() {
		k8sClient, cancel = StartTestManagerWithReconciler(func(mgr ctrl.Manager) error {
			return (&stage.StageVersionUsageReconciler{
				Client: mgr.GetClient(),
				Scheme: mgr.GetScheme(),
			}).SetupWithManager(mgr)
		},
		)
	})

	AfterAll(func() {
		cancel()
	})

	const (
		StageVersionTest      = "stage-version-test"
		StageVersionTest2     = "stage-version-test-2"
		StageVersionTestUsage = "stage-version-usage-test-usage"
		Namespace             = "default"
		StageDev              = "stage-dev"
		Vector001             = "https://registry.kdenv.lab/ocm/vector//landscape.konfidence.cloud/example/vector:0.0.1"
		VectorName001         = "star.konfidence.cloud.example.vector-0.0.1"
		timeout               = time.Second * 10
		interval              = time.Millisecond * 250
	)

	BeforeEach(func() {
		testutil.CleanupResources(k8sClient)
	})

	AfterEach(func() {
		testutil.CleanupResources(k8sClient)
	})

	Context("When reconciling a stageVersionUsage", func() {
		It("should successfully reconcile the stageVersionUsage", func() {
			ctx := context.Background()
			testutil.CreateStageVersion(ctx, k8sClient, StageDev, StageVersionTest, Namespace, Vector001, VectorName001)

			// check that the stageVersion has been created and has valid properties
			stageVersion := &star.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersionTest, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Name).To(Equal(StageVersionTest))
			}, timeout, interval).Should(Succeed())

			testutil.CreateStageVersionUsage(ctx, k8sClient, StageVersionTestUsage, Namespace, StageVersionTest)

			// check that the stageVersionUsage has been created and has valid properties
			stageVersionUsage := &star.StageVersionUsage{}
			stageVersionUsageLookupKey := types.NamespacedName{Name: StageVersionTestUsage, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.Name).To(Equal(StageVersionTestUsage))
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions).To(HaveLen(1))
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions[0]).To(Equal(StageVersionTest))
				g.Expect(meta.IsStatusConditionTrue(stageVersionUsage.Status.Conditions, star.StageVersionUsageReady)).To(BeFalse())
			}, timeout, interval).Should(Succeed())

			// mark stageVersion as ready
			meta.SetStatusCondition(&stageVersion.Status.Conditions, metav1.Condition{
				Type:               star.StageVersionReady,
				Status:             metav1.ConditionTrue,
				Reason:             star.StageVersionReady,
				Message:            "StageVersion is ready",
				ObservedGeneration: stageVersion.Generation,
				LastTransitionTime: metav1.Now(),
			})

			Expect(k8sClient.Status().Update(ctx, stageVersion)).To(Succeed())

			// check that stageVersionUsage has condition StageVersionReady set to true
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions).To(HaveLen(1))
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions[0]).To(Equal(StageVersionTest))
				g.Expect(meta.IsStatusConditionTrue(stageVersionUsage.Status.Conditions, star.StageVersionUsageReady)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

		})
		It("should set condition StageVersionNotFound when referenced stage version does not exist", func() {
			ctx := context.Background()
			testutil.CreateStageVersionUsage(ctx, k8sClient, StageVersionTestUsage, Namespace, StageVersionTest)

			// check that the stageVersionUsage has been created and has valid properties
			stageVersionUsage := &star.StageVersionUsage{}
			stageVersionUsageLookupKey := types.NamespacedName{Name: StageVersionTestUsage, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.Name).To(Equal(StageVersionTestUsage))
				g.Expect(meta.IsStatusConditionTrue(stageVersionUsage.Status.Conditions, star.StageVersionNotFound)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			testutil.CreateStageVersion(ctx, k8sClient, StageDev, StageVersionTest, Namespace, Vector001, VectorName001)

			// check that the stageVersion has been created and has valid properties
			stageVersion := &star.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersionTest, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Name).To(Equal(StageVersionTest))
			}, timeout, interval).Should(Succeed())

			// check that StageVersionNotFound condition has been removed
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.Name).To(Equal(StageVersionTestUsage))
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions).To(HaveLen(1))
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions[0]).To(Equal(StageVersionTest))
				g.Expect(meta.FindStatusCondition(stageVersionUsage.Status.Conditions, star.StageVersionNotFound)).To(BeNil())
			}, timeout, interval).Should(Succeed())
		})
		It("should resolve multiple stageVersions by selector and mark stageVersionUsage only ready if all stageVersions are ready", func() {
			ctx := context.Background()
			testutil.CreateStageVersionUsageWithSelector(ctx, k8sClient, StageVersionTestUsage, Namespace, StageDev, VectorName001, false)

			// check that the stageVersionUsage has been created and has valid properties
			stageVersionUsage := &star.StageVersionUsage{}
			stageVersionUsageLookupKey := types.NamespacedName{Name: StageVersionTestUsage, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.Name).To(Equal(StageVersionTestUsage))
				g.Expect(meta.IsStatusConditionTrue(stageVersionUsage.Status.Conditions, star.StageVersionNotFound)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			testutil.CreateStageVersionWithLabels(ctx, k8sClient, StageVersionTest, Namespace, Vector001, StageDev, VectorName001)

			// check that the stageVersion has been created and has valid properties
			stageVersion := &star.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersionTest, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Name).To(Equal(StageVersionTest))
			}, timeout, interval).Should(Succeed())

			// check that the not found condition has been removed, the stageVersion has been resolved and that the usage is not yet ready
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.Name).To(Equal(StageVersionTestUsage))
				g.Expect(meta.FindStatusCondition(stageVersionUsage.Status.Conditions, star.StageVersionNotFound)).To(BeNil())
				g.Expect(meta.IsStatusConditionFalse(stageVersionUsage.Status.Conditions, star.StageVersionUsageReady)).To(BeTrue())
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions).To(HaveLen(1))
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions[0]).To(Equal(StageVersionTest))
			}, timeout, interval).Should(Succeed())

			testutil.CreateStageVersionWithLabels(ctx, k8sClient, StageVersionTest2, Namespace, Vector001, StageDev, VectorName001)

			// check that the stageVersion has been created and has valid properties
			stageVersion2 := &star.StageVersion{}
			stageVersionLookupKey2 := types.NamespacedName{Name: StageVersionTest2, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey2, stageVersion2)).To(Succeed())
				g.Expect(stageVersion2.Name).To(Equal(StageVersionTest2))
			}, timeout, interval).Should(Succeed())

			// check that both stageVersions have been resolved and that the usage is marked as not ready
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.Name).To(Equal(StageVersionTestUsage))
				g.Expect(meta.IsStatusConditionFalse(stageVersionUsage.Status.Conditions, star.StageVersionUsageReady)).To(BeTrue())
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions).To(HaveLen(2))
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions[0]).To(Equal(StageVersionTest))
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions[1]).To(Equal(StageVersionTest2))
			}, timeout, interval).Should(Succeed())

			// mark stageVersion1 as ready
			meta.SetStatusCondition(&stageVersion.Status.Conditions, metav1.Condition{
				Type:               star.StageVersionReady,
				Status:             metav1.ConditionTrue,
				Reason:             star.StageVersionReady,
				Message:            "StageVersion is ready",
				ObservedGeneration: stageVersion.Generation,
				LastTransitionTime: metav1.Now(),
			})

			Expect(k8sClient.Status().Update(ctx, stageVersion)).To(Succeed())

			// check that the usage is still not ready
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.Name).To(Equal(StageVersionTestUsage))
				g.Expect(meta.IsStatusConditionFalse(stageVersionUsage.Status.Conditions, star.StageVersionUsageReady)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// mark stageVersion2 as ready
			meta.SetStatusCondition(&stageVersion2.Status.Conditions, metav1.Condition{
				Type:               star.StageVersionReady,
				Status:             metav1.ConditionTrue,
				Reason:             star.StageVersionReady,
				Message:            "StageVersion is ready",
				ObservedGeneration: stageVersion2.Generation,
				LastTransitionTime: metav1.Now(),
			})

			Expect(k8sClient.Status().Update(ctx, stageVersion2)).To(Succeed())

			// check that now the usage has been marked as ready
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.Name).To(Equal(StageVersionTestUsage))
				g.Expect(meta.IsStatusConditionTrue(stageVersionUsage.Status.Conditions, star.StageVersionUsageReady)).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})
})
