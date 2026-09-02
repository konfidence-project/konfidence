package controller_test

import (
	"context"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/stage/internal/controller"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
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
		StageVersion                = "stage-dev-789jgf975c4cr"
		StageVersionUpdated         = "stage-dev-44d75n8wqlcg6"
		Namespace                   = "default"
		Vector001                   = "https://registry.kdenv.lab/ocm/vector//landscape.konfidence.tools.cloud/example/vector:0.0.1"
		Vector002                   = "https://registry.kdenv.lab/ocm/vector//landscape.konfidence.tools.cloud/example/vector:0.0.2"
		VectorName001               = "example.konfidence.cloud.example.vector-0.0.1"
		Vector001Digest             = "dz6hwnwzg5vlpgwnp67f9d4zd"
		Vector002Digest             = "dz6hwnw2kxclb7j4757cbsf7m"
		StageVersionManuallyCreated = "stage-version-usage-manually-created"
		timeout                     = time.Second * 10
		interval                    = time.Millisecond * 250
	)

	BeforeEach(func() {
		controller.CleanupResources(k8sClient)
	})

	AfterEach(func() {
		controller.CleanupResources(k8sClient)
	})

	Context("When reconciling a stage", func() {
		It("should successfully reconcile the stage", func() {
			ctx := context.Background()
			controller.CreateStage(ctx, k8sClient, StageDev, Namespace, Vector001)

			// check that the stage has been created and has valid properties
			stage := &konfidence.Stage{}
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
			controller.CreateStageVersion(ctx, k8sClient, StageDev, StageVersion, Namespace, Vector001, VectorName001)

			// check that the stageVersion has been created
			stageVersion := &konfidence.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersion, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			controller.CreateStage(ctx, k8sClient, StageDev, Namespace, Vector001)
			verifyStageReady(ctx, k8sClient, StageDev, Namespace, timeout, interval)
		})
		It("should update existing target stageVersionUsage with new stage vector", func() {
			ctx := context.Background()
			controller.CreateStage(ctx, k8sClient, StageDev, Namespace, Vector001)

			// check that the stage has been created and has valid properties
			stage := &konfidence.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			verifyStageVersionUsage(ctx, k8sClient, Namespace, stage, Vector001Digest, timeout, interval)

			// update stage with new vector
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				stage.Spec.Vector = Vector002
				g.Expect(k8sClient.Update(ctx, stage)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// check that the existing target stageVersionUsage has been updated
			verifyStageVersionUsage(ctx, k8sClient, Namespace, stage, Vector002Digest, timeout, interval)

			// check that the new stageVersion has been created and has valid properties
			verifyStageVersion(ctx, k8sClient, StageVersionUpdated, Namespace, stage, Vector002Digest, timeout, interval)
		})
		It("should delete manually created target stageVersionUsages", func() {
			ctx := context.Background()
			controller.CreateStageVersionUsageWithSelector(ctx, k8sClient, StageVersionManuallyCreated, Namespace, StageDev, Vector001Digest, true)
			controller.CreateStage(ctx, k8sClient, StageDev, Namespace, Vector001)

			// check that the stage has been created and has valid properties
			stage := &konfidence.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// check that only one targetUsage remains and that it has valid properties
			verifyStageVersionUsage(ctx, k8sClient, Namespace, stage, Vector001Digest, timeout, interval)
		})
	})

	Context("When mirroring the active stageVersion into the stage status", func() {
		It("should leave the active stageVersion reference unset if no active stageVersionUsage exists", func() {
			ctx := context.Background()
			controller.CreateStage(ctx, k8sClient, StageDev, Namespace, Vector001)

			verifyStageReady(ctx, k8sClient, StageDev, Namespace, timeout, interval)
			verifyActiveStageVersion(ctx, k8sClient, StageDev, Namespace, nil, timeout, interval)
		})

		It("should mirror, update and clear the active stageVersion reference", func() {
			ctx := context.Background()
			controller.CreateStage(ctx, k8sClient, StageDev, Namespace, Vector001)
			verifyStageReady(ctx, k8sClient, StageDev, Namespace, timeout, interval)

			stage := controller.GetStage(ctx, k8sClient, StageDev, Namespace, false)

			// creating the active stageVersionUsage is mirrored into the stage status
			controller.CreateActiveStageVersionUsage(ctx, k8sClient, stage, StageVersion)
			verifyActiveStageVersion(ctx, k8sClient, StageDev, Namespace, ptr.To(StageVersion), timeout, interval)

			// updating the reference bumps the usage generation and is mirrored as well
			controller.UpdateActiveStageVersionUsageRef(ctx, k8sClient, stage, StageVersionUpdated)
			verifyActiveStageVersion(ctx, k8sClient, StageDev, Namespace, ptr.To(StageVersionUpdated), timeout, interval)

			// deleting the active stageVersionUsage clears the reference again
			activeUsage := controller.GetStageVersionUsage(ctx, k8sClient, konfidence.ActiveStageVersionUsageName(StageDev), Namespace, false)
			controller.DeleteStageVersionUsage(ctx, k8sClient, activeUsage)
			verifyActiveStageVersion(ctx, k8sClient, StageDev, Namespace, nil, timeout, interval)
		})

		It("should leave the active stageVersion reference unset if the active stageVersionUsage references no stageVersion", func() {
			ctx := context.Background()
			controller.CreateStage(ctx, k8sClient, StageDev, Namespace, Vector001)
			verifyStageReady(ctx, k8sClient, StageDev, Namespace, timeout, interval)

			stage := controller.GetStage(ctx, k8sClient, StageDev, Namespace, false)

			// an active stageVersionUsage that resolves its stageVersion by selector carries
			// no reference to mirror
			controller.CreateActiveStageVersionUsageWithSelector(ctx, k8sClient, stage)
			verifyActiveStageVersionStaysUnset(ctx, k8sClient, StageDev, Namespace, interval)

			// setting a reference on that very usage is mirrored ...
			controller.UpdateActiveStageVersionUsageRef(ctx, k8sClient, stage, StageVersion)
			verifyActiveStageVersion(ctx, k8sClient, StageDev, Namespace, ptr.To(StageVersion), timeout, interval)

			// ... and dropping it again clears the mirrored reference
			controller.ClearActiveStageVersionUsageRef(ctx, k8sClient, stage)
			verifyActiveStageVersion(ctx, k8sClient, StageDev, Namespace, nil, timeout, interval)
		})
	})
})

// check that the stage status keeps mirroring no active stageVersion at all
func verifyActiveStageVersionStaysUnset(ctx context.Context, k8sClient client.Client, stageName string,
	namespace string, interval time.Duration) {
	stage := &konfidence.Stage{}
	stageLookupKey := types.NamespacedName{Name: stageName, Namespace: namespace}
	Consistently(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
		g.Expect(stage.Status.ActiveStageVersion).To(BeNil())
	}, time.Second*2, interval).Should(Succeed())
}

// check that the stage status mirrors the expected active stageVersion, nil meaning no reference at all
//
//nolint:unparam
func verifyActiveStageVersion(ctx context.Context, k8sClient client.Client, stageName string, namespace string,
	expected *string, timeout time.Duration, interval time.Duration) {
	stage := &konfidence.Stage{}
	stageLookupKey := types.NamespacedName{Name: stageName, Namespace: namespace}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
		if expected == nil {
			g.Expect(stage.Status.ActiveStageVersion).To(BeNil())
			return
		}
		g.Expect(stage.Status.ActiveStageVersion).ToNot(BeNil())
		g.Expect(stage.Status.ActiveStageVersion.Name).To(Equal(*expected))
	}, timeout, interval).Should(Succeed())
}

// check that the stageVersion has been created and has valid properties
func verifyStageVersion(ctx context.Context, k8sClient client.Client, stageVersionName string, namespace string,
	stage *konfidence.Stage, vectorRef string, timeout time.Duration, interval time.Duration) {
	stageVersion := &konfidence.StageVersion{}
	stageVersionLookupKey := types.NamespacedName{Name: stageVersionName, Namespace: namespace}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
		g.Expect(stageVersion.Spec.Vector).To(Equal(stage.Spec.Vector))
		g.Expect(stageVersion.Spec.StageGeneration).To(Equal(stage.Generation))
		g.Expect(stageVersion.Labels[pkgctrl.StageNameLabel]).To(Equal(stage.Name))
		g.Expect(stageVersion.Labels[pkgctrl.VectorReferenceLabel]).To(Equal(vectorRef))
		g.Expect(stageVersion.GetOwnerReferences()).To(HaveLen(1))
		g.Expect(controller.HasOwnerReference(stageVersion.GetOwnerReferences(), metav1.OwnerReference{
			Kind: konfidence.StageKind,
			Name: stage.Name,
		})).To(BeTrue())
	}, timeout, interval).Should(Succeed())
}

// check that the target stageVersionUsage has been created and has valid properties
//
//nolint:unparam
func verifyStageVersionUsage(ctx context.Context, k8sClient client.Client, namespace string,
	stage *konfidence.Stage, vectorRef string, timeout time.Duration, interval time.Duration) {
	stageVersionUsages := &konfidence.StageVersionUsageList{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.List(ctx, stageVersionUsages, client.InNamespace(namespace))).To(Succeed())
		g.Expect(stageVersionUsages.Items).To(HaveLen(1))
		g.Expect(stageVersionUsages.Items[0].Labels[pkgctrl.StageVersionUsageTarget]).To(Equal(stage.Name))
		g.Expect(stageVersionUsages.Items[0].GetOwnerReferences()).To(HaveLen(1))
		g.Expect(stageVersionUsages.Items[0].Spec.Reason).To(Equal(controller.StageVersionUsageTargetType))
		g.Expect(stageVersionUsages.Items[0].Spec.StageVersionSelector.MatchLabels[pkgctrl.StageNameLabel]).To(Equal(stage.Name))
		g.Expect(stageVersionUsages.Items[0].Spec.StageVersionSelector.MatchLabels[pkgctrl.VectorReferenceLabel]).To(Equal(vectorRef))
		g.Expect(controller.HasOwnerReference(stageVersionUsages.Items[0].GetOwnerReferences(), metav1.OwnerReference{
			Kind: konfidence.StageKind,
			Name: stage.Name,
		})).To(BeTrue())
	}, timeout, interval).Should(Succeed())
}

//nolint:unparam
func verifyStageReady(ctx context.Context, k8sClient client.Client, stageName string, namespace string, timeout time.Duration, interval time.Duration) {
	stage := &konfidence.Stage{}
	stageLookupKey := types.NamespacedName{Name: stageName, Namespace: namespace}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
		g.Expect(meta.IsStatusConditionTrue(stage.Status.Conditions, konfidence.StageReady)).To(BeTrue())
	}, timeout, interval).Should(Succeed())
}
