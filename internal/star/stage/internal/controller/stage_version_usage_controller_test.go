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

	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	testutil "github.com/konfidence-project/landscape-stage-controller/internal/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

var _ = Describe("StageVersionUsage Controller", func() {
	const (
		StageVersionTest           = "stage-version-test"
		StageVersionTestUsage      = "stage-version-usage-test-usage"
		StageVersionTestMigration  = "stage-version-test-migration"
		StageVersionTest2          = "stage-version-test-2"
		StageVersionTest2Migration = "stage-version-test-2-migration"
		Namespace                  = "default"
		StageDev                   = "stage-dev"
		Vector001                  = "https://registry.kdenv.lab/ocm/vector//common.konfidence.cloud/example/vector:0.0.1"
		VectorName001              = "common.konfidence.cloud.example.vector-0.0.1"
		timeout                    = time.Second * 10
		interval                   = time.Millisecond * 250
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
			testutil.CreateStageVersion(ctx, k8sClient, StageVersionTest, Namespace, Vector001)

			// check that the stageVersion has been created and has valid properties
			stageVersion := &landscape.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersionTest, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Name).To(Equal(StageVersionTest))
			}, timeout, interval).Should(Succeed())

			testutil.CreateStageVersionUsage(ctx, k8sClient, StageVersionTestUsage, Namespace, StageVersionTest)

			// check that the stageVersionUsage has been created and has valid properties
			stageVersionUsage := &landscape.StageVersionUsage{}
			stageVersionUsageLookupKey := types.NamespacedName{Name: StageVersionTestUsage, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.Name).To(Equal(StageVersionTestUsage))
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions).To(HaveLen(1))
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions[0]).To(Equal(StageVersionTest))
				g.Expect(meta.IsStatusConditionTrue(stageVersionUsage.Status.Conditions, landscape.StageVersionReady)).To(BeFalse())
			}, timeout, interval).Should(Succeed())

			// check that the vectorDeployment has been created
			vectorDeployment := &landscape.VectorDeployment{}
			vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// mark vectorDeployment as deployed
			meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{Type: landscape.VectorDeployedCondition,
				Status: metav1.ConditionTrue, Reason: landscape.VectorDeployedCondition,
				Message: "Vector has been successfully deployed"})

			Expect(k8sClient.Status().Update(ctx, vectorDeployment)).To(Succeed())

			// check that the vectorMigration has been created
			vectorMigration := &landscape.VectorMigration{}
			vectorMigrationLookupKey := types.NamespacedName{Name: StageVersionTestMigration, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// mark vectorMigration as successful
			meta.SetStatusCondition(&vectorMigration.Status.Conditions, metav1.Condition{Type: landscape.VectorMigrationSucceeded,
				Status: metav1.ConditionTrue, Reason: landscape.VectorMigrationSucceeded,
				Message: "VectorMigration succeeded"})

			Expect(k8sClient.Status().Update(ctx, vectorMigration)).To(Succeed())

			// check that stageVersionUsage has condition StageVersionReady set to true
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions).To(HaveLen(1))
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions[0]).To(Equal(StageVersionTest))
				g.Expect(meta.IsStatusConditionTrue(stageVersionUsage.Status.Conditions, landscape.StageVersionReady)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

		})
		It("should set condition StageVersionNotFound when referenced stage version does not exist", func() {
			ctx := context.Background()
			testutil.CreateStageVersionUsage(ctx, k8sClient, StageVersionTestUsage, Namespace, StageVersionTest)

			// check that the stageVersionUsage has been created and has valid properties
			stageVersionUsage := &landscape.StageVersionUsage{}
			stageVersionUsageLookupKey := types.NamespacedName{Name: StageVersionTestUsage, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.Name).To(Equal(StageVersionTestUsage))
				g.Expect(meta.IsStatusConditionTrue(stageVersionUsage.Status.Conditions, landscape.StageVersionNotFound)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			testutil.CreateStageVersion(ctx, k8sClient, StageVersionTest, Namespace, Vector001)

			// check that the stageVersion has been created and has valid properties
			stageVersion := &landscape.StageVersion{}
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
				g.Expect(meta.FindStatusCondition(stageVersionUsage.Status.Conditions, landscape.StageVersionNotFound)).To(BeNil())
			}, timeout, interval).Should(Succeed())
		})
		It("should resolve multiple stageVersions by selector and mark stageVersionUsage only ready if all stageVersions are ready", func() {
			ctx := context.Background()
			testutil.CreateStageVersionUsageWithSelector(ctx, k8sClient, StageVersionTestUsage, Namespace, StageDev, VectorName001, false)

			// check that the stageVersionUsage has been created and has valid properties
			stageVersionUsage := &landscape.StageVersionUsage{}
			stageVersionUsageLookupKey := types.NamespacedName{Name: StageVersionTestUsage, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.Name).To(Equal(StageVersionTestUsage))
				g.Expect(meta.IsStatusConditionTrue(stageVersionUsage.Status.Conditions, landscape.StageVersionNotFound)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			testutil.CreateStageVersionWithLabels(ctx, k8sClient, StageVersionTest, Namespace, Vector001, StageDev, VectorName001)

			// check that the stageVersion has been created and has valid properties
			stageVersion := &landscape.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersionTest, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Name).To(Equal(StageVersionTest))
			}, timeout, interval).Should(Succeed())

			// check that the not found condition has been removed, the stageVersion has been resolved and that the usage is not yet ready
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.Name).To(Equal(StageVersionTestUsage))
				g.Expect(meta.FindStatusCondition(stageVersionUsage.Status.Conditions, landscape.StageVersionNotFound)).To(BeNil())
				g.Expect(meta.IsStatusConditionFalse(stageVersionUsage.Status.Conditions, landscape.StageVersionReady)).To(BeTrue())
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions).To(HaveLen(1))
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions[0]).To(Equal(StageVersionTest))
			}, timeout, interval).Should(Succeed())

			testutil.CreateStageVersionWithLabels(ctx, k8sClient, StageVersionTest2, Namespace, Vector001, StageDev, VectorName001)

			// check that the stageVersion has been created and has valid properties
			stageVersion2 := &landscape.StageVersion{}
			stageVersionLookupKey2 := types.NamespacedName{Name: StageVersionTest2, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey2, stageVersion2)).To(Succeed())
				g.Expect(stageVersion2.Name).To(Equal(StageVersionTest2))
			}, timeout, interval).Should(Succeed())

			// check that both stageVersions have been resolved and that the usage is marked as ready
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.Name).To(Equal(StageVersionTestUsage))
				g.Expect(meta.IsStatusConditionFalse(stageVersionUsage.Status.Conditions, landscape.StageVersionReady)).To(BeTrue())
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions).To(HaveLen(2))
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions[0]).To(Equal(StageVersionTest))
				g.Expect(stageVersionUsage.Status.ResolvedStageVersions[1]).To(Equal(StageVersionTest2))
			}, timeout, interval).Should(Succeed())

			// check that the vectorDeployment has been created
			vectorDeployment := &landscape.VectorDeployment{}
			vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
					err := k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)
					if err != nil {
						return err
					}

					// mark vectorDeployment as deployed
					meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{Type: landscape.VectorDeployedCondition,
						Status: metav1.ConditionTrue, Reason: landscape.VectorDeployedCondition,
						Message: "Vector has been successfully deployed"})

					return k8sClient.Status().Update(ctx, vectorDeployment)
				})).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// check that both vectorMigrations have been created
			vectorMigration := &landscape.VectorMigration{}
			vectorMigrationLookupKey := types.NamespacedName{Name: StageVersionTestMigration, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			vectorMigration2 := &landscape.VectorMigration{}
			vectorMigrationLookupKey2 := types.NamespacedName{Name: StageVersionTest2Migration, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey2, vectorMigration2)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// mark vectorMigrations as successful
			meta.SetStatusCondition(&vectorMigration.Status.Conditions, metav1.Condition{Type: landscape.VectorMigrationSucceeded,
				Status: metav1.ConditionTrue, Reason: landscape.VectorMigrationSucceeded,
				Message: "VectorMigration succeeded"})
			Expect(k8sClient.Status().Update(ctx, vectorMigration)).To(Succeed())
			meta.SetStatusCondition(&vectorMigration2.Status.Conditions, metav1.Condition{Type: landscape.VectorMigrationSucceeded,
				Status: metav1.ConditionTrue, Reason: landscape.VectorMigrationSucceeded,
				Message: "VectorMigration succeeded"})
			Expect(k8sClient.Status().Update(ctx, vectorMigration2)).To(Succeed())

			// check that usage has been marked as ready
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.Name).To(Equal(StageVersionTestUsage))
				g.Expect(meta.IsStatusConditionTrue(stageVersionUsage.Status.Conditions, landscape.StageVersionReady)).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})
})
