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

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	"github.com/konfidence-project/landscape-stage-controller/internal/controller"
	testutil "github.com/konfidence-project/landscape-stage-controller/internal/utils"
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
		StageDev                    = "stage-dev"
		StageDevSpecName            = "dev"
		StageVersion                = "stage-version-z3q9efdlno78"
		StageVersionUpdated         = "stage-version-u67xcg4acsv0"
		Namespace                   = "default"
		Vector001                   = "https://registry.kdenv.lab/ocm/vector//common.konfidence.tools.cloud/example/vector:0.0.1"
		Vector002                   = "https://registry.kdenv.lab/ocm/vector//common.konfidence.tools.cloud/example/vector:0.0.2"
		VectorName001               = "common.konfidence.cloud.example.vector-0.0.1"
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
			testutil.CreateStage(ctx, k8sClient, StageDev, Namespace, StageDevSpecName, Vector001)

			// check that the stage has been created and has valid properties
			stage := &common.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Name).To(Equal(StageDevSpecName))
			}, timeout, interval).Should(Succeed())

			adaptedVectorName, err := testutil.AdaptVectorName(stage.Spec.Vector)
			Expect(err).ToNot(HaveOccurred(), "Failed to get adapted vector name")

			// check that the target stageVersionUsage has been created and has valid properties
			stageVersionUsages := &landscape.StageVersionUsageList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, stageVersionUsages, client.InNamespace(Namespace))).To(Succeed())
				g.Expect(stageVersionUsages.Items).To(HaveLen(1))
				g.Expect(stageVersionUsages.Items[0].Labels[controller.StageVersionUsageTarget]).To(Equal(stage.Name))
				g.Expect(stageVersionUsages.Items[0].GetOwnerReferences()).To(HaveLen(1))
				g.Expect(stageVersionUsages.Items[0].Spec.Reason).To(Equal(controller.StageVersionUsageTargetType))
				g.Expect(stageVersionUsages.Items[0].Spec.StageVersionSelector.MatchLabels[controller.StageNameLabel]).To(Equal(stage.Name))
				g.Expect(stageVersionUsages.Items[0].Spec.StageVersionSelector.MatchLabels[controller.VectorReferenceLabel]).To(Equal(adaptedVectorName))
				g.Expect(testutil.HasOwnerReference(stageVersionUsages.Items[0].GetOwnerReferences(), metav1.OwnerReference{
					Kind: common.StageKind,
					Name: StageDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// check that the stageVersion has been created and has valid properties
			stageVersion := &landscape.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersion, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Spec.Vector).To(Equal(stage.Spec.Vector))
				g.Expect(stageVersion.Spec.StageGeneration).To(Equal(stage.Generation))
				g.Expect(stageVersion.Labels[controller.StageNameLabel]).To(Equal(stage.Name))
				g.Expect(stageVersion.Labels[controller.VectorReferenceLabel]).To(Equal(adaptedVectorName))
				g.Expect(stageVersion.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.HasOwnerReference(stageVersion.GetOwnerReferences(), metav1.OwnerReference{
					Kind: common.StageKind,
					Name: StageDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Name).To(Equal(StageDevSpecName))
				g.Expect(meta.IsStatusConditionTrue(stage.Status.Conditions, common.StageReady)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

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

			testutil.CreateStage(ctx, k8sClient, StageDev, Namespace, StageDevSpecName, Vector001)

			// check that the stage has been created and has valid properties
			stage := &common.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Name).To(Equal(StageDevSpecName))
				g.Expect(meta.IsStatusConditionTrue(stage.Status.Conditions, common.StageReady)).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
		It("should update existing target stageVersionUsage with new stage vector", func() {
			ctx := context.Background()
			testutil.CreateStage(ctx, k8sClient, StageDev, Namespace, StageDevSpecName, Vector001)

			// check that the stage has been created and has valid properties
			stage := &common.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Name).To(Equal(StageDevSpecName))
			}, timeout, interval).Should(Succeed())

			adaptedVectorName, err := testutil.AdaptVectorName(stage.Spec.Vector)
			Expect(err).ToNot(HaveOccurred(), "Failed to get adapted vector name")

			// check that the target stageVersionUsage has been created and has valid properties
			stageVersionUsages := &landscape.StageVersionUsageList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, stageVersionUsages, client.InNamespace(Namespace))).To(Succeed())
				g.Expect(stageVersionUsages.Items).To(HaveLen(1))
				g.Expect(stageVersionUsages.Items[0].Labels[controller.StageVersionUsageTarget]).To(Equal(stage.Name))
				g.Expect(stageVersionUsages.Items[0].GetOwnerReferences()).To(HaveLen(1))
				g.Expect(stageVersionUsages.Items[0].Spec.Reason).To(Equal(controller.StageVersionUsageTargetType))
				g.Expect(stageVersionUsages.Items[0].Spec.StageVersionSelector.MatchLabels[controller.StageNameLabel]).To(Equal(stage.Name))
				g.Expect(stageVersionUsages.Items[0].Spec.StageVersionSelector.MatchLabels[controller.VectorReferenceLabel]).To(Equal(adaptedVectorName))
				g.Expect(testutil.HasOwnerReference(stageVersionUsages.Items[0].GetOwnerReferences(), metav1.OwnerReference{
					Kind: common.StageKind,
					Name: StageDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// update stage with new vector
			stage.Spec.Vector = Vector002
			Expect(k8sClient.Update(ctx, stage)).To(Succeed())

			adaptedVectorName, err = testutil.AdaptVectorName(stage.Spec.Vector)
			Expect(err).ToNot(HaveOccurred(), "Failed to get new adapted vector name")

			// check that the existing target stageVersionUsage has been updated
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, stageVersionUsages, client.InNamespace(Namespace))).To(Succeed())
				g.Expect(stageVersionUsages.Items).To(HaveLen(1))
				g.Expect(stageVersionUsages.Items[0].Labels[controller.StageVersionUsageTarget]).To(Equal(stage.Name))
				g.Expect(stageVersionUsages.Items[0].GetOwnerReferences()).To(HaveLen(1))
				g.Expect(stageVersionUsages.Items[0].Spec.Reason).To(Equal(controller.StageVersionUsageTargetType))
				g.Expect(stageVersionUsages.Items[0].Spec.StageVersionSelector.MatchLabels[controller.StageNameLabel]).To(Equal(stage.Name))
				g.Expect(stageVersionUsages.Items[0].Spec.StageVersionSelector.MatchLabels[controller.VectorReferenceLabel]).To(Equal(adaptedVectorName))
				g.Expect(testutil.HasOwnerReference(stageVersionUsages.Items[0].GetOwnerReferences(), metav1.OwnerReference{
					Kind: common.StageKind,
					Name: StageDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// check that the new stageVersion has been created and has valid properties
			stageVersion := &landscape.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersionUpdated, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Spec.Vector).To(Equal(stage.Spec.Vector))
				g.Expect(stageVersion.Spec.StageGeneration).To(Equal(stage.Generation))
				g.Expect(stageVersion.Labels[controller.StageNameLabel]).To(Equal(stage.Name))
				g.Expect(stageVersion.Labels[controller.VectorReferenceLabel]).To(Equal(adaptedVectorName))
				g.Expect(stageVersion.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.HasOwnerReference(stageVersion.GetOwnerReferences(), metav1.OwnerReference{
					Kind: common.StageKind,
					Name: StageDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
		It("should delete manually created target stageVersionUsages", func() {
			ctx := context.Background()
			adaptedVectorName, err := testutil.AdaptVectorName(Vector001)
			Expect(err).ToNot(HaveOccurred(), "Failed to get adapted vector name")

			testutil.CreateStageVersionUsageWithSelector(ctx, k8sClient, StageVersionManuallyCreated, Namespace, StageDev, VectorName001, true)
			testutil.CreateStage(ctx, k8sClient, StageDev, Namespace, StageDevSpecName, Vector001)

			// check that the stage has been created and has valid properties
			stage := &common.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Name).To(Equal(StageDevSpecName))
			}, timeout, interval).Should(Succeed())

			// check that only one targetUsage remains and that it has valid properties
			stageVersionUsages := &landscape.StageVersionUsageList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, stageVersionUsages, client.InNamespace(Namespace))).To(Succeed())
				g.Expect(stageVersionUsages.Items).To(HaveLen(1))
				g.Expect(stageVersionUsages.Items[0].Labels[controller.StageVersionUsageTarget]).To(Equal(stage.Name))
				g.Expect(stageVersionUsages.Items[0].GetOwnerReferences()).To(HaveLen(1))
				g.Expect(stageVersionUsages.Items[0].Spec.Reason).To(Equal(controller.StageVersionUsageTargetType))
				g.Expect(stageVersionUsages.Items[0].Spec.StageVersionSelector.MatchLabels[controller.StageNameLabel]).To(Equal(stage.Name))
				g.Expect(stageVersionUsages.Items[0].Spec.StageVersionSelector.MatchLabels[controller.VectorReferenceLabel]).To(Equal(adaptedVectorName))
				g.Expect(testutil.HasOwnerReference(stageVersionUsages.Items[0].GetOwnerReferences(), metav1.OwnerReference{
					Kind: common.StageKind,
					Name: StageDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})
})
