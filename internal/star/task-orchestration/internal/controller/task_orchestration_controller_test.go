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
	testutil "github.com/konfidence-project/landscape-task-orchestration-controller/internal/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Task Orchestration Controller", func() {
	const (
		StageVersion      = "stage-version-stage-dev"
		VectorMigration   = "stage-version-stage-dev-migration"
		StageVersionUsage = "stage-version-stage-dev-usage"
		Namespace         = "default"
		Vector001         = "https://registry.kdenv.lab/ocm/vector//common.konfidence.tools.cloud/example/vector:0.0.1"
		timeout           = time.Second * 10
		interval          = time.Millisecond * 250
	)

	BeforeEach(func() {
		testutil.CleanupStageVersion(k8sClient, StageVersion, Namespace)
		testutil.CleanupStageVersionUsage(k8sClient, StageVersionUsage, Namespace)
		testutil.CleanupVectorMigration(k8sClient, VectorMigration, Namespace)
	})

	AfterEach(func() {
		testutil.CleanupStageVersion(k8sClient, StageVersion, Namespace)
		testutil.CleanupStageVersionUsage(k8sClient, StageVersionUsage, Namespace)
		testutil.CleanupVectorMigration(k8sClient, VectorMigration, Namespace)
	})

	Context("When reconciling a vectorMigration", func() {
		It("should successfully reconcile the vectorMigration", func() {
			ctx := context.Background()
			testutil.CreateStageVersion(ctx, k8sClient, StageVersion, Namespace, Vector001)

			// check that the stageVersion has been created and has valid properties
			stageVersion := &landscape.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersion, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Spec.Vector).To(Equal(Vector001))
				g.Expect(stageVersion.Spec.StageGeneration).To(Equal(int64(1)))
			}, timeout, interval).Should(Succeed())

			// create the vectorMigration
			testutil.CreateVectorMigration(ctx, k8sClient, VectorMigration, Namespace, StageVersion, Vector001)

			// check that the vectorMigration has been created and has valid properties
			vectorMigration := &landscape.VectorMigration{}
			vectorMigrationLookupKey := types.NamespacedName{Name: VectorMigration, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
				g.Expect(vectorMigration.Spec.StageVersion).To(Equal(StageVersion))
				g.Expect(vectorMigration.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// check that a new stageVersionUsage has been created
			stageVersionUsage := &landscape.StageVersionUsage{}
			stageVersionUsageLookupKey := types.NamespacedName{Name: StageVersionUsage, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.ContainsReference(stageVersionUsage.GetOwnerReferences(), VectorMigration, landscape.VectorMigrationKind)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// check that the stageVersionUsage has been set as owner of the stageVersion
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(testutil.ContainsReference(stageVersion.GetOwnerReferences(), StageVersionUsage, landscape.StageVersionUsageKind)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

		})
	})

	Context("When deleting a vectorMigration", func() {
		It("it should delete the stageVersionUsage and the stageVersion if no other owner reference exists", func() {
			ctx := context.Background()
			testutil.CreateStageVersion(ctx, k8sClient, StageVersion, Namespace, Vector001)

			// check that the stageVersion has been created and has valid properties
			stageVersion := &landscape.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersion, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Spec.Vector).To(Equal(Vector001))
				g.Expect(stageVersion.Spec.StageGeneration).To(Equal(int64(1)))
			}, timeout, interval).Should(Succeed())

			// create the vectorMigration
			testutil.CreateVectorMigration(ctx, k8sClient, VectorMigration, Namespace, StageVersion, Vector001)

			// check that the vectorMigration has been created and has valid properties
			vectorMigration := &landscape.VectorMigration{}
			vectorMigrationLookupKey := types.NamespacedName{Name: VectorMigration, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
				g.Expect(vectorMigration.Spec.StageVersion).To(Equal(StageVersion))
				g.Expect(vectorMigration.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// check that a new stageVersionUsage has been created
			stageVersionUsage := &landscape.StageVersionUsage{}
			stageVersionUsageLookupKey := types.NamespacedName{Name: StageVersionUsage, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.ContainsReference(stageVersionUsage.GetOwnerReferences(), VectorMigration, landscape.VectorMigrationKind)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// check that the stageVersionUsage has been set as owner of the stageVersion
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(testutil.ContainsReference(stageVersion.GetOwnerReferences(), StageVersionUsage, landscape.StageVersionUsageKind)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// delete the vectorMigration
			testutil.DeleteVectorMigration(ctx, k8sClient, vectorMigration)

			// check that vectorMigration has been deleted
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError(errors.IsNotFound, "Should be a not found error"))
			}, timeout, interval).Should(Succeed())

			// check that stageVersionUsage has been deleted
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError(errors.IsNotFound, "Should be a not found error"))
			}, timeout, interval).Should(Succeed())

			// check that stageVersion has been deleted
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError(errors.IsNotFound, "Should be a not found error"))
			}, timeout, interval).Should(Succeed())
		})
	})
})
