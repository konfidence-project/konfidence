/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller_test

import (
	"context"
	"time"

	landscape "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/star/stage/internal/controller"
	testutil "github.com/konfidence-project/konfidence/internal/star/stage/test/utils"
	pkgCtrl "github.com/konfidence-project/konfidence/pkg/controller"
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
		Vector001       = "https://registry.kdenv.lab/ocm/vector//landscape.konfidence.cloud/example/vector:0.0.1"
		VectorName001   = "star.konfidence.cloud.example.vector-0.0.1"
		timeout         = time.Second * 10
		interval        = time.Millisecond * 250
	)

	BeforeEach(func() {
		testutil.CleanupResources(k8sClient)
	})

	AfterEach(func() {
		testutil.CleanupResources(k8sClient)
	})

	Context("When reconciling a stageVersion", func() {
		It("should successfully reconcile the stageVersion", func() {
			ctx := context.Background()
			testutil.CreateStageVersion(ctx, k8sClient, StageDev, StageVersionDev, Namespace, Vector001, VectorName001)

			// check that the stageVersion has been created and has valid properties
			stageVersion := &landscape.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersionDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Name).To(Equal(StageVersionDev))
				g.Expect(stageVersion.Status.Conditions).To(HaveLen(1))
				g.Expect(meta.IsStatusConditionTrue(stageVersion.Status.Conditions, landscape.VectorDeploymentCreatedCondition)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// check that the vectorDeployment has been created and has valid properties
			vectorDeployment := &landscape.VectorDeployment{}
			vectorDeploymentLookupKey := types.NamespacedName{Name: StageVersionDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(stageVersion.Spec.Vector))
				g.Expect(vectorDeployment.Labels[pkgCtrl.StageVersionNameLabel]).To(Equal(StageVersionDev))
				g.Expect(vectorDeployment.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.HasOwnerReference(vectorDeployment.GetOwnerReferences(), metav1.OwnerReference{
					Kind: landscape.StageVersionKind,
					Name: StageVersionDev,
				})).To(BeTrue())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// mark vectorDeployment as deployed
			meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
				Type:               landscape.VectorDeployedCondition,
				Status:             metav1.ConditionTrue,
				Reason:             landscape.VectorDeployedCondition,
				Message:            "Vector has been successfully deployed",
				ObservedGeneration: vectorDeployment.Generation,
				LastTransitionTime: metav1.Now(),
			})

			Expect(k8sClient.Status().Update(ctx, vectorDeployment)).To(Succeed())

			// check that the vectorMigration has been created and has valid properties
			vectorMigration := &landscape.VectorMigration{}
			vectorMigrationLookupKey := types.NamespacedName{Name: StageVersionDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
				g.Expect(vectorMigration.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.HasOwnerReference(vectorMigration.GetOwnerReferences(), metav1.OwnerReference{
					Kind: landscape.StageVersionKind,
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
				g.Expect(meta.IsStatusConditionTrue(stageVersion.Status.Conditions, landscape.VectorMigrationCreatedCondition)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// mark vectorMigration as successful
			meta.SetStatusCondition(&vectorMigration.Status.Conditions, metav1.Condition{
				Type:               landscape.VectorMigrationSucceeded,
				Status:             metav1.ConditionTrue,
				Reason:             landscape.VectorMigrationSucceeded,
				Message:            "VectorMigration succeeded",
				ObservedGeneration: vectorMigration.Generation,
				LastTransitionTime: metav1.Now()})

			Expect(k8sClient.Status().Update(ctx, vectorMigration)).To(Succeed())

			// check that the vectorActivation has been created and has valid properties
			vectorActivation := &landscape.VectorActivation{}
			vectorActivationLookupKey := types.NamespacedName{Name: StageVersionDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorActivationLookupKey, vectorActivation)).To(Succeed())
				g.Expect(vectorActivation.Spec.Stage).To(Equal(StageDev))
				g.Expect(vectorActivation.Spec.StageVersion).To(Equal(StageVersionDev))
				g.Expect(vectorActivation.Spec.Vector).To(Equal(Vector001))
				g.Expect(vectorActivation.Spec.VectorDeployment).To(Equal(StageVersionDev))
				g.Expect(vectorActivation.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.HasOwnerReference(vectorActivation.GetOwnerReferences(), metav1.OwnerReference{
					Kind: landscape.StageVersionKind,
					Name: StageVersionDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// check that the stageVersion has status vectorMigrated, vectorActivationCreated and stageVersionReady
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Name).To(Equal(StageVersionDev))
				g.Expect(stageVersion.Status.Conditions).To(HaveLen(5))
				g.Expect(meta.IsStatusConditionTrue(stageVersion.Status.Conditions, landscape.VectorMigratedCondition)).To(BeTrue())
				g.Expect(meta.IsStatusConditionTrue(stageVersion.Status.Conditions, landscape.VectorActivationCreatedCondition)).To(BeTrue())
				g.Expect(meta.IsStatusConditionTrue(stageVersion.Status.Conditions, landscape.StageVersionReady)).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})
})
