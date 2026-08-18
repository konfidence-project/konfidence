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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("StageVersion Controller", Ordered, func() {
	var (
		k8sClient client.Client
		cancel    context.CancelFunc
	)

	BeforeAll(func() {
		k8sClient, cancel = StartTestManagerWithReconciler(func(mgr ctrl.Manager) error {
			return (&controller.StageVersionReconciler{
				Client:   mgr.GetClient(),
				Scheme:   mgr.GetScheme(),
				Recorder: mgr.GetEventRecorder(controller.StageVersionControllerName),
			}).SetupWithManager(mgr)
		},
		)
	})

	AfterAll(func() {
		cancel()
	})

	const (
		StageDev        = "stage-dev"
		StageVersionDev = "stage-version-dev"
		Namespace       = "default"
		Vector001       = "https://registry.kdenv.lab/ocm/vector//example.konfidence.cloud/example/vector:0.0.1"
		VectorName001   = "example.konfidence.cloud.example.vector-0.0.1"
		timeout         = time.Second * 10
		interval        = time.Millisecond * 250
	)

	BeforeEach(func() {
		controller.CleanupResources(k8sClient)
	})

	AfterEach(func() {
		controller.CleanupResources(k8sClient)
	})

	Context("When reconciling a stageVersion", func() {
		It("should successfully reconcile the stageVersion", func() {
			ctx := context.Background()
			controller.CreateStageVersion(ctx, k8sClient, StageDev, StageVersionDev, Namespace, Vector001, VectorName001)

			// check that the stageVersion has been created and has valid properties
			stageVersion := &konfidence.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersionDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Name).To(Equal(StageVersionDev))
				g.Expect(stageVersion.Status.Conditions).To(HaveLen(1))
				g.Expect(meta.IsStatusConditionTrue(stageVersion.Status.Conditions, konfidence.VectorDeploymentCreatedCondition)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// check that the vectorDeployment has been created and has valid properties
			vectorDeployment := &konfidence.VectorDeployment{}
			vectorDeploymentLookupKey := types.NamespacedName{Name: StageVersionDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(stageVersion.Spec.Vector))
				g.Expect(vectorDeployment.Labels[pkgctrl.StageVersionNameLabel]).To(Equal(StageVersionDev))
				g.Expect(vectorDeployment.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(controller.HasOwnerReference(vectorDeployment.GetOwnerReferences(), metav1.OwnerReference{
					Kind: konfidence.StageVersionKind,
					Name: StageVersionDev,
				})).To(BeTrue())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// mark vectorDeployment as ready
			meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
				Type:               konfidence.VectorReadyCondition,
				Status:             metav1.ConditionTrue,
				Reason:             konfidence.VectorReadyCondition,
				Message:            "Vector deployment is ready",
				ObservedGeneration: vectorDeployment.Generation,
				LastTransitionTime: metav1.Now(),
			})

			Expect(k8sClient.Status().Update(ctx, vectorDeployment)).To(Succeed())

			// check that the vectorMigration has been created and has valid properties
			vectorMigration := &konfidence.VectorMigration{}
			vectorMigrationLookupKey := types.NamespacedName{Name: StageVersionDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
				g.Expect(vectorMigration.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(controller.HasOwnerReference(vectorMigration.GetOwnerReferences(), metav1.OwnerReference{
					Kind: konfidence.StageVersionKind,
					Name: StageVersionDev,
				})).To(BeTrue())
				g.Expect(vectorMigration.Spec.Vector).To(Equal(Vector001))
				g.Expect(vectorMigration.Spec.StageVersion).To(Equal(StageVersionDev))
			}, timeout, interval).Should(Succeed())

			// check that the stageVersion has status vectorMigrationCreated
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Name).To(Equal(StageVersionDev))
				g.Expect(stageVersion.Status.Conditions).To(HaveLen(2))
				g.Expect(meta.IsStatusConditionTrue(stageVersion.Status.Conditions, konfidence.VectorMigrationCreatedCondition)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// mark vectorMigration as successful
			meta.SetStatusCondition(&vectorMigration.Status.Conditions, metav1.Condition{
				Type:               konfidence.VectorMigrationSucceeded,
				Status:             metav1.ConditionTrue,
				Reason:             konfidence.VectorMigrationSucceeded,
				Message:            "VectorMigration succeeded",
				ObservedGeneration: vectorMigration.Generation,
				LastTransitionTime: metav1.Now()})

			Expect(k8sClient.Status().Update(ctx, vectorMigration)).To(Succeed())

			// check that the vectorActivation has been created and has valid properties
			vectorActivation := &konfidence.VectorActivation{}
			vectorActivationLookupKey := types.NamespacedName{Name: StageVersionDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorActivationLookupKey, vectorActivation)).To(Succeed())
				g.Expect(vectorActivation.Spec.Stage).To(Equal(StageDev))
				g.Expect(vectorActivation.Spec.StageVersion).To(Equal(StageVersionDev))
				g.Expect(vectorActivation.Spec.Vector).To(Equal(Vector001))
				g.Expect(vectorActivation.Spec.VectorDeployment).To(Equal(StageVersionDev))
				g.Expect(vectorActivation.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(controller.HasOwnerReference(vectorActivation.GetOwnerReferences(), metav1.OwnerReference{
					Kind: konfidence.StageVersionKind,
					Name: StageVersionDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// check that the stageVersion has status vectorMigrated, vectorActivationCreated and stageVersionReady
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Name).To(Equal(StageVersionDev))
				g.Expect(stageVersion.Status.Conditions).To(HaveLen(5))
				g.Expect(meta.IsStatusConditionTrue(stageVersion.Status.Conditions, konfidence.VectorMigratedCondition)).To(BeTrue())
				g.Expect(meta.IsStatusConditionTrue(stageVersion.Status.Conditions, konfidence.VectorActivationCreatedCondition)).To(BeTrue())
				g.Expect(meta.IsStatusConditionTrue(stageVersion.Status.Conditions, konfidence.StageVersionReady)).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})

		It("should not create the vectorMigration while the vectorDeployment is only deployed but not ready", func() {
			ctx := context.Background()
			controller.CreateStageVersion(ctx, k8sClient, StageDev, StageVersionDev, Namespace, Vector001, VectorName001)

			vectorDeployment := &konfidence.VectorDeployment{}
			vectorDeploymentLookupKey := types.NamespacedName{Name: StageVersionDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// mark vectorDeployment as deployed only
			meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
				Type:               konfidence.VectorDeployedCondition,
				Status:             metav1.ConditionTrue,
				Reason:             konfidence.VectorDeployedCondition,
				Message:            "Vector has been successfully deployed",
				ObservedGeneration: vectorDeployment.Generation,
				LastTransitionTime: metav1.Now(),
			})
			Expect(k8sClient.Status().Update(ctx, vectorDeployment)).To(Succeed())

			// the vectorMigration must not be created until VectorReady is set
			vectorMigrationLookupKey := types.NamespacedName{Name: StageVersionDev, Namespace: Namespace}
			Consistently(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, &konfidence.VectorMigration{})).ToNot(Succeed())
			}, time.Second*2, interval).Should(Succeed())
		})
	})
})
