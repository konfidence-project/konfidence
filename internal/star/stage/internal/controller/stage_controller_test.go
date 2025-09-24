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

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	testutil "github.com/konfidence-project/landscape-stage-controller/internal/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("Stage Controller", func() {
	const (
		StageDev          = "stage-dev"
		StageDevSpecName  = "dev"
		StageDev2SpecName = "dev2"
		StageVersionUsage = "stage-dev-usage"
		Namespace         = "default"
		Vector001         = "https://registry.kdenv.lab/ocm/vector//common.konfidence.tools.cloud/example/vector:0.0.1"
		timeout           = time.Second * 10
		interval          = time.Millisecond * 250
	)

	BeforeEach(func() {
		testutil.CleanupStage(k8sClient, StageDev, Namespace)
		testutil.CleanupStageVersionUsage(k8sClient, StageVersionUsage, Namespace)
	})

	AfterEach(func() {
		testutil.CleanupStage(k8sClient, StageDev, Namespace)
		testutil.CleanupStageVersionUsage(k8sClient, StageVersionUsage, Namespace)
	})

	Context("When reconciling a stage", func() {
		It("should successfully reconcile the stage", func() {
			ctx := context.Background()
			testutil.CreateStage(ctx, k8sClient, StageDev, Namespace, StageDevSpecName, Vector001)

			// check that the stage has been created and has valid properties
			stage := &common.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Name).To(Equal(StageDevSpecName))
			}, timeout, interval).Should(Succeed())

			// check that the stageVersion has been created and has valid properties
			stageVersions := &landscape.StageVersionList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, stageVersions, client.InNamespace(Namespace))).To(Succeed())
				g.Expect(stageVersions.Items).To(HaveLen(1))
				g.Expect(stageVersions.Items[0].Spec.Vector).To(Equal(stage.Spec.Vector))
				g.Expect(stageVersions.Items[0].Spec.StageGeneration).To(Equal(stage.Generation))
				g.Expect(stageVersions.Items[0].GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.HasOwnerReference(stageVersions.Items[0].GetOwnerReferences(), metav1.OwnerReference{
					Kind: common.StageKind,
					Name: StageDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

		})
		It("should delete old stageVersion if after a stage update the stage owner reference is the last one remaining", func() {
			ctx := context.Background()
			testutil.CreateStage(ctx, k8sClient, StageDev, Namespace, StageDevSpecName, Vector001)

			// check that the stage has been created and has valid properties
			stage := &common.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Name).To(Equal(StageDevSpecName))
			}, timeout, interval).Should(Succeed())

			// check that the stageVersion has been created and has valid properties
			stageVersions := &landscape.StageVersionList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, stageVersions, client.InNamespace(Namespace))).To(Succeed())
				g.Expect(stageVersions.Items).To(HaveLen(1))
				g.Expect(stageVersions.Items[0].Spec.Vector).To(Equal(stage.Spec.Vector))
				g.Expect(stageVersions.Items[0].Spec.StageGeneration).To(Equal(stage.Generation))
				g.Expect(stageVersions.Items[0].GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.HasOwnerReference(stageVersions.Items[0].GetOwnerReferences(), metav1.OwnerReference{
					Kind: common.StageKind,
					Name: StageDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			oldUid := stageVersions.Items[0].UID

			// update stage spec name
			stage.Spec.Name = StageDev2SpecName
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Update(ctx, stage)).To(Succeed())
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Name).To(Equal(StageDev2SpecName))
			}, timeout, interval).Should(Succeed())

			// check that the old stageVersion has been deleted and a new one has been created instead
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, stageVersions, client.InNamespace(Namespace))).To(Succeed())
				g.Expect(stageVersions.Items).To(HaveLen(1))
				g.Expect(stageVersions.Items[0].Spec.Vector).To(Equal(stage.Spec.Vector))
				g.Expect(stageVersions.Items[0].Spec.StageGeneration).To(Equal(stage.Generation))
				g.Expect(stageVersions.Items[0].UID).To(Not(Equal(oldUid)))
				g.Expect(stageVersions.Items[0].GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.HasOwnerReference(stageVersions.Items[0].GetOwnerReferences(), metav1.OwnerReference{
					Kind: common.StageKind,
					Name: StageDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

		})
		It("should not delete old stageVersion if the stageVersion has multiple owner references", func() {
			ctx := context.Background()
			testutil.CreateStage(ctx, k8sClient, StageDev, Namespace, StageDevSpecName, Vector001)

			// check that the stage has been created and has valid properties
			stage := &common.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Name).To(Equal(StageDevSpecName))
			}, timeout, interval).Should(Succeed())

			// check that the stageVersion has been created and has valid properties
			stageVersions := &landscape.StageVersionList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, stageVersions, client.InNamespace(Namespace))).To(Succeed())
				g.Expect(stageVersions.Items).To(HaveLen(1))
				g.Expect(stageVersions.Items[0].Spec.Vector).To(Equal(stage.Spec.Vector))
				g.Expect(stageVersions.Items[0].Spec.StageGeneration).To(Equal(stage.Generation))
				g.Expect(stageVersions.Items[0].GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.HasOwnerReference(stageVersions.Items[0].GetOwnerReferences(), metav1.OwnerReference{
					Kind: common.StageKind,
					Name: StageDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// create a stageVersionUsage
			oldUid := stageVersions.Items[0].UID
			stageVersionName := stageVersions.Items[0].Name
			testutil.CreateStageVersionUsage(ctx, k8sClient, StageVersionUsage, Namespace)

			// check that the stageVersionUsage has been created and has valid properties
			stageVersionUsage := &landscape.StageVersionUsage{}
			stageVersionUsageLookupKey := types.NamespacedName{Name: StageVersionUsage, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.Name).To(Equal(StageVersionUsage))
			}, timeout, interval).Should(Succeed())

			// set stageVersionUsage as owner of the stageVersion
			stageVersion := &landscape.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: stageVersionName, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(controllerutil.SetOwnerReference(stageVersionUsage, stageVersion, reconcileScheme)).To(Succeed())
				g.Expect(k8sClient.Update(ctx, stageVersion)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// check that the stageVersion now has two owner references, one for the stage and one for the usage
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, stageVersions, client.InNamespace(Namespace))).To(Succeed())
				g.Expect(stageVersions.Items).To(HaveLen(1))
				g.Expect(stageVersions.Items[0].Spec.Vector).To(Equal(stage.Spec.Vector))
				g.Expect(stageVersions.Items[0].Spec.StageGeneration).To(Equal(stage.Generation))
				g.Expect(stageVersions.Items[0].GetOwnerReferences()).To(HaveLen(2))
				g.Expect(testutil.HasOwnerReference(stageVersions.Items[0].GetOwnerReferences(), metav1.OwnerReference{
					Kind: common.StageKind,
					Name: StageDev,
				})).To(BeTrue())
				g.Expect(testutil.HasOwnerReference(stageVersions.Items[0].GetOwnerReferences(), metav1.OwnerReference{
					Kind: landscape.StageVersionUsageKind,
					Name: StageVersionUsage,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// update the stage spec name, this changes the stage object generation value
			stage.Spec.Name = StageDev2SpecName
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Update(ctx, stage)).To(Succeed())
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Name).To(Equal(StageDev2SpecName))
			}, timeout, interval).Should(Succeed())

			// check that the old stageVersion has not been deleted and has one remaining owner reference
			// the stage reference should have been removed and added to the new stageVersion object
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, stageVersions, client.InNamespace(Namespace))).To(Succeed())
				g.Expect(stageVersions.Items).To(HaveLen(2))

				var oldStageVersion landscape.StageVersion
				var newStageVersion landscape.StageVersion

				if stageVersions.Items[0].UID == oldUid {
					oldStageVersion = stageVersions.Items[0]
					newStageVersion = stageVersions.Items[1]
				} else {
					oldStageVersion = stageVersions.Items[1]
					newStageVersion = stageVersions.Items[0]
				}

				g.Expect(oldStageVersion.Spec.Vector).To(Equal(stage.Spec.Vector))
				g.Expect(oldStageVersion.Spec.StageGeneration).To(Equal(stage.Generation - 1))
				g.Expect(oldStageVersion.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.HasOwnerReference(oldStageVersion.GetOwnerReferences(), metav1.OwnerReference{
					Kind: landscape.StageVersionUsageKind,
					Name: StageVersionUsage,
				})).To(BeTrue())
				g.Expect(newStageVersion.Spec.Vector).To(Equal(stage.Spec.Vector))
				g.Expect(newStageVersion.Spec.StageGeneration).To(Equal(stage.Generation))
				g.Expect(newStageVersion.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.HasOwnerReference(newStageVersion.GetOwnerReferences(), metav1.OwnerReference{
					Kind: common.StageKind,
					Name: StageDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})
})
