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

package controller

import (
	"context"
	"time"

	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	testutil "github.com/konfidence-project/landscape-stage-controller/internal/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("StageVersion Controller", func() {
	const (
		StageVersionDev          = "stage-version-dev"
		StageVersionTest         = "stage-version-test"
		Namespace                = "default"
		Vector001                = "https://registry.kdenv.lab/ocm/vector//common.konfidence.cloud/example/vector:0.0.1"
		VectorName001            = "common.konfidence.cloud.example.vector-0.0.1"
		StageVersionDevMigration = "stage-version-dev-migration"
		timeout                  = time.Second * 10
		interval                 = time.Millisecond * 250
	)

	BeforeEach(func() {
		testutil.CleanupStageVersion(k8sClient, StageVersionDev, Namespace)
		testutil.CleanupStageVersion(k8sClient, StageVersionTest, Namespace)
	})

	AfterEach(func() {
		testutil.CleanupStageVersion(k8sClient, StageVersionDev, Namespace)
		testutil.CleanupStageVersion(k8sClient, StageVersionTest, Namespace)
	})

	Context("When reconciling a stageVersion", func() {
		It("should successfully reconcile the stageVersion", func() {
			ctx := context.Background()
			testutil.CreateStageVersion(ctx, k8sClient, StageVersionDev, Namespace, Vector001)

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
			vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(stageVersion.Spec.Vector))
				g.Expect(vectorDeployment.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.HasOwnerReference(vectorDeployment.GetOwnerReferences(), metav1.OwnerReference{
					Kind: landscape.StageVersionKind,
					Name: StageVersionDev,
				})).To(BeTrue())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// mark vectorDeployment as deployed
			meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{Type: landscape.VectorDeployedCondition,
				Status: metav1.ConditionTrue, Reason: landscape.VectorDeployedCondition,
				Message: "Vector has been successfully deployed"})

			Expect(k8sClient.Status().Update(ctx, vectorDeployment)).To(Succeed())

			// check that the vectorMigration has been created and has valid properties
			vectorMigration := &landscape.VectorMigration{}
			vectorMigrationLookupKey := types.NamespacedName{Name: StageVersionDevMigration, Namespace: Namespace}
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

			// check that the stageVersion has status vectorMigrationCreated and ready
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Name).To(Equal(StageVersionDev))
				g.Expect(stageVersion.Status.Conditions).To(HaveLen(3))
				g.Expect(meta.IsStatusConditionTrue(stageVersion.Status.Conditions, landscape.VectorMigrationCreatedCondition)).To(BeTrue())
				g.Expect(meta.IsStatusConditionTrue(stageVersion.Status.Conditions, landscape.StageVersionReady)).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
		It("should re-use a vectorDeployment if another stageVersion references the same vector", func() {
			ctx := context.Background()
			testutil.CreateStageVersion(ctx, k8sClient, StageVersionDev, Namespace, Vector001)

			// check that the stageVersion has been created and has valid properties
			stageVersionDev := &landscape.StageVersion{}
			stageVersionDevLookupKey := types.NamespacedName{Name: StageVersionDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionDevLookupKey, stageVersionDev)).To(Succeed())
				g.Expect(stageVersionDev.Name).To(Equal(StageVersionDev))
				g.Expect(stageVersionDev.Status.Conditions).To(HaveLen(1))
				g.Expect(meta.IsStatusConditionTrue(stageVersionDev.Status.Conditions, landscape.VectorDeploymentCreatedCondition)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// create another stageVersion that references the same vector
			testutil.CreateStageVersion(ctx, k8sClient, StageVersionTest, Namespace, Vector001)

			// check that the stageVersion has been created and has valid properties
			stageVersionTest := &landscape.StageVersion{}
			stageVersionTestLookupKey := types.NamespacedName{Name: StageVersionDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionTestLookupKey, stageVersionTest)).To(Succeed())
				g.Expect(stageVersionTest.Name).To(Equal(StageVersionDev))
				g.Expect(stageVersionTest.Status.Conditions).To(HaveLen(1))
				g.Expect(meta.IsStatusConditionTrue(stageVersionTest.Status.Conditions, landscape.VectorDeploymentCreatedCondition)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// check that the vectorDeployment has been created and has both stageVersions set as owner references
			vectorDeployment := &landscape.VectorDeployment{}
			vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(stageVersionDev.Spec.Vector))
				g.Expect(vectorDeployment.GetOwnerReferences()).To(HaveLen(2))
				g.Expect(testutil.HasOwnerReference(vectorDeployment.GetOwnerReferences(), metav1.OwnerReference{
					Kind: landscape.StageVersionKind,
					Name: StageVersionDev,
				})).To(BeTrue())
				g.Expect(testutil.HasOwnerReference(vectorDeployment.GetOwnerReferences(), metav1.OwnerReference{
					Kind: landscape.StageVersionKind,
					Name: StageVersionTest,
				})).To(BeTrue())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())
		})

	})

	Context("When deleting a stageVersion", func() {
		It("it should delete the vectorDeployment if no other stageVersion references it", func() {
			ctx := context.Background()
			testutil.CreateStageVersion(ctx, k8sClient, StageVersionDev, Namespace, Vector001)

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
			vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(stageVersion.Spec.Vector))
				g.Expect(vectorDeployment.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.HasOwnerReference(vectorDeployment.GetOwnerReferences(), metav1.OwnerReference{
					Kind: landscape.StageVersionKind,
					Name: StageVersionDev,
				})).To(BeTrue())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// delete the stageVersion
			testutil.DeleteStageVersion(ctx, k8sClient, stageVersion)

			// check that vectorDeployment has been deleted
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError(errors.IsNotFound, "Should be a not found error"))
			}, timeout, interval).Should(Succeed())
		})

		It("it should delete the vectorMigration", func() {
			ctx := context.Background()
			testutil.CreateStageVersion(ctx, k8sClient, StageVersionDev, Namespace, Vector001)

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
			vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(stageVersion.Spec.Vector))
				g.Expect(vectorDeployment.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.HasOwnerReference(vectorDeployment.GetOwnerReferences(), metav1.OwnerReference{
					Kind: landscape.StageVersionKind,
					Name: StageVersionDev,
				})).To(BeTrue())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// mark vectorDeployment as deployed
			meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{Type: landscape.VectorDeployedCondition,
				Status: metav1.ConditionTrue, Reason: landscape.VectorDeployedCondition,
				Message: "Vector has been successfully deployed"})

			Expect(k8sClient.Status().Update(ctx, vectorDeployment)).To(Succeed())

			// check that the vectorMigration has been created and has valid properties
			vectorMigration := &landscape.VectorMigration{}
			vectorMigrationLookupKey := types.NamespacedName{Name: StageVersionDevMigration, Namespace: Namespace}
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

			// delete the stageVersion
			testutil.DeleteStageVersion(ctx, k8sClient, stageVersion)

			// check that vectorMigration has been deleted
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError(errors.IsNotFound, "Should be a not found error"))
			}, timeout, interval).Should(Succeed())
		})

	})
})
