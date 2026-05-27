package stage_test

import (
	"context"
	"time"

	landscape "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	controller "github.com/konfidence-project/konfidence/internal/star/stage"
	testutil "github.com/konfidence-project/konfidence/internal/star/stage/internal/utils"
	pkgCtrl "github.com/konfidence-project/konfidence/pkg/controller"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Stage Controller", Ordered, func() {
	var (
		k8sClient client.Client
		cancel    context.CancelFunc
	)

	BeforeAll(func() {
		k8sClient, cancel = StartTestManagerWithReconciler(func(mgr ctrl.Manager) error {
			return (&controller.StageReconciler{
				Client:   mgr.GetClient(),
				Scheme:   mgr.GetScheme(),
				Recorder: mgr.GetEventRecorder(controller.StageControllerName),
			}).SetupWithManager(mgr)
		},
		)
	})

	AfterAll(func() {
		cancel()
	})

	const (
		StageDev                    = "stage-dev"
		StageVersion                = "stage-dev-3j5rp95ig707y"
		StageVersionUpdated         = "stage-dev-n31v4bxt7p2"
		Namespace                   = "default"
		Vector001                   = "https://registry.kdenv.lab/ocm/vector//landscape.konfidence.tools.cloud/example/vector:0.0.1"
		Vector002                   = "https://registry.kdenv.lab/ocm/vector//landscape.konfidence.tools.cloud/example/vector:0.0.2"
		VectorName001               = "star.konfidence.cloud.example.vector-0.0.1"
		Vector001Digest             = "8dhqbvbdpgatwpbvwhio5n0dn"
		Vector002Digest             = "8dhqbvbesc7tl3rfig37lzoiu"
		StageVersionManuallyCreated = "stage-version-usage-manually-created"
		timeout                     = time.Second * 10
		interval                    = time.Millisecond * 250
	)

	BeforeEach(func() {
		testutil.CleanupResources(k8sClient)
	})

	AfterEach(func() {
		testutil.CleanupResources(k8sClient)
	})

	Context("When reconciling a stage", func() {
		It("should successfully reconcile the stage", func() {
			ctx := context.Background()
			testutil.CreateStage(ctx, k8sClient, StageDev, Namespace, Vector001)

			// check that the stage has been created and has valid properties
			stage := &landscape.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			verifyStageVersionUsage(ctx, k8sClient, Namespace, stage, Vector001Digest, timeout, interval)
			verifyStageVersion(ctx, k8sClient, StageVersion, Namespace, stage, Vector001Digest, timeout, interval)
			verifyStageReady(ctx, k8sClient, StageDev, Namespace, timeout, interval)
		})
		It("should successfully reconcile the stage if stageVersion already exists", func() {
			ctx := context.Background()
			testutil.CreateStageVersion(ctx, k8sClient, StageDev, StageVersion, Namespace, Vector001, VectorName001)

			// check that the stageVersion has been created
			stageVersion := &landscape.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersion, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			testutil.CreateStage(ctx, k8sClient, StageDev, Namespace, Vector001)
			verifyStageReady(ctx, k8sClient, StageDev, Namespace, timeout, interval)
		})
		It("should update existing target stageVersionUsage with new stage vector", func() {
			ctx := context.Background()
			testutil.CreateStage(ctx, k8sClient, StageDev, Namespace, Vector001)

			// check that the stage has been created and has valid properties
			stage := &landscape.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			verifyStageVersionUsage(ctx, k8sClient, Namespace, stage, Vector001Digest, timeout, interval)

			// update stage with new vector
			stage.Spec.Vector = Vector002
			Expect(k8sClient.Update(ctx, stage)).To(Succeed())

			// check that the existing target stageVersionUsage has been updated
			verifyStageVersionUsage(ctx, k8sClient, Namespace, stage, Vector002Digest, timeout, interval)

			// check that the new stageVersion has been created and has valid properties
			verifyStageVersion(ctx, k8sClient, StageVersionUpdated, Namespace, stage, Vector002Digest, timeout, interval)
		})
		It("should delete manually created target stageVersionUsages", func() {
			ctx := context.Background()
			testutil.CreateStageVersionUsageWithSelector(ctx, k8sClient, StageVersionManuallyCreated, Namespace, StageDev, Vector001Digest, true)
			testutil.CreateStage(ctx, k8sClient, StageDev, Namespace, Vector001)

			// check that the stage has been created and has valid properties
			stage := &landscape.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// check that only one targetUsage remains and that it has valid properties
			verifyStageVersionUsage(ctx, k8sClient, Namespace, stage, Vector001Digest, timeout, interval)
		})
	})
})

// check that the stageVersion has been created and has valid properties
func verifyStageVersion(ctx context.Context, k8sClient client.Client, stageVersionName string, namespace string,
	stage *landscape.Stage, vectorRef string, timeout time.Duration, interval time.Duration) {
	stageVersion := &landscape.StageVersion{}
	stageVersionLookupKey := types.NamespacedName{Name: stageVersionName, Namespace: namespace}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
		g.Expect(stageVersion.Spec.Vector).To(Equal(stage.Spec.Vector))
		g.Expect(stageVersion.Spec.StageGeneration).To(Equal(stage.Generation))
		g.Expect(stageVersion.Labels[pkgCtrl.StageNameLabel]).To(Equal(stage.Name))
		g.Expect(stageVersion.Labels[pkgCtrl.VectorReferenceLabel]).To(Equal(vectorRef))
		g.Expect(stageVersion.GetOwnerReferences()).To(HaveLen(1))
		g.Expect(testutil.HasOwnerReference(stageVersion.GetOwnerReferences(), metav1.OwnerReference{
			Kind: landscape.StageKind,
			Name: stage.Name,
		})).To(BeTrue())
	}, timeout, interval).Should(Succeed())
}

// check that the target stageVersionUsage has been created and has valid properties
//
//nolint:unparam
func verifyStageVersionUsage(ctx context.Context, k8sClient client.Client, namespace string,
	stage *landscape.Stage, vectorRef string, timeout time.Duration, interval time.Duration) {
	stageVersionUsages := &landscape.StageVersionUsageList{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.List(ctx, stageVersionUsages, client.InNamespace(namespace))).To(Succeed())
		g.Expect(stageVersionUsages.Items).To(HaveLen(1))
		g.Expect(stageVersionUsages.Items[0].Labels[pkgCtrl.StageVersionUsageTarget]).To(Equal(stage.Name))
		g.Expect(stageVersionUsages.Items[0].GetOwnerReferences()).To(HaveLen(1))
		g.Expect(stageVersionUsages.Items[0].Spec.Reason).To(Equal(controller.StageVersionUsageTargetType))
		g.Expect(stageVersionUsages.Items[0].Spec.StageVersionSelector.MatchLabels[pkgCtrl.StageNameLabel]).To(Equal(stage.Name))
		g.Expect(stageVersionUsages.Items[0].Spec.StageVersionSelector.MatchLabels[pkgCtrl.VectorReferenceLabel]).To(Equal(vectorRef))
		g.Expect(testutil.HasOwnerReference(stageVersionUsages.Items[0].GetOwnerReferences(), metav1.OwnerReference{
			Kind: landscape.StageKind,
			Name: stage.Name,
		})).To(BeTrue())
	}, timeout, interval).Should(Succeed())
}

func verifyStageReady(ctx context.Context, k8sClient client.Client, stageName string, namespace string, timeout time.Duration, interval time.Duration) {
	stage := &landscape.Stage{}
	stageLookupKey := types.NamespacedName{Name: stageName, Namespace: namespace}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
		g.Expect(meta.IsStatusConditionTrue(stage.Status.Conditions, landscape.StageReady)).To(BeTrue())
	}, timeout, interval).Should(Succeed())
}
