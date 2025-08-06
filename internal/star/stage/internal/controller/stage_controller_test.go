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
	"fmt"
	"time"

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Stage Controller", func() {
	const (
		StageDev          = "stage-dev"
		StageDevSpecName  = "dev"
		StageTest         = "stage-test"
		StageTestSpecName = "test"
		Namespace         = "default"
		Vector001         = "https://registry.kdenv.lab/ocm/vector//common.konfidence.konfidence.cloud/example/vector:0.0.1"
		VectorName001     = "common.konfidence.konfidence.cloud.example.vector-0.0.1"
		Vector002         = "https://registry.kdenv.lab/ocm/vector//common.konfidence.konfidence.cloud/example/vector:0.0.2"
		VectorName002     = "common.konfidence.konfidence.cloud.example.vector-0.0.2"
		timeout           = time.Second * 10
		interval          = time.Millisecond * 250
	)

	BeforeEach(func() {
		cleanupStage(k8sClient, StageDev, Namespace)
		cleanupStage(k8sClient, StageTest, Namespace)
	})

	AfterEach(func() {
		cleanupStage(k8sClient, StageDev, Namespace)
		cleanupStage(k8sClient, StageTest, Namespace)
	})

	Context("When reconciling a stage", func() {
		It("should successfully reconcile the stage", func() {
			ctx := context.Background()
			createStage(ctx, k8sClient, StageDev, Namespace, StageDevSpecName, Vector001)

			// check that the stage has been created and has valid properties
			stage := &common.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Name).To(Equal(StageDevSpecName))
				g.Expect(len(stage.Status.Conditions)).To(Equal(1))
				g.Expect(stage.Status.Conditions[0].Reason).To(Equal(common.VectorDeploymentCreatedCondition))
				g.Expect(stage.Status.Conditions[0].Type).To(Equal(common.VectorDeploymentCreatedCondition))
				g.Expect(stage.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			}, timeout, interval).Should(Succeed())

			// check that the vectorDeploymentUsage has been created and has valid properties
			vectorDeploymentUsages := &landscape.VectorDeploymentUsageList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, vectorDeploymentUsages, client.InNamespace(Namespace))).To(Succeed())
				g.Expect(len(vectorDeploymentUsages.Items)).To(Equal(1))
				g.Expect(vectorDeploymentUsages.Items[0].Spec.Vector).To(Equal(stage.Spec.Vector))
				g.Expect(len(vectorDeploymentUsages.Items[0].GetOwnerReferences())).To(Equal(1))
				g.Expect(vectorDeploymentUsages.Items[0].GetOwnerReferences()[0].Kind).To(Equal(common.StageKind))
				g.Expect(vectorDeploymentUsages.Items[0].GetOwnerReferences()[0].Name).To(Equal(StageDev))
			}, timeout, interval).Should(Succeed())

			// check that the vectorDeployment has been created and has valid properties
			vectorDeployment := &landscape.VectorDeployment{}
			vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(stage.Spec.Vector))
				g.Expect(len(vectorDeployment.GetOwnerReferences())).To(Equal(2))
				g.Expect(containsReference(vectorDeployment.GetOwnerReferences(), StageDev, common.StageKind)).To(BeTrue())
				g.Expect(containsReference(vectorDeployment.GetOwnerReferences(), vectorDeploymentUsages.Items[0].Name, landscape.VectorDeploymentUsageKind)).To(BeTrue())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// mark vectorDeployment as deployed
			meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{Type: common.VectorDeployedCondition,
				Status: metav1.ConditionTrue, Reason: landscape.VectorDeployedCondition,
				Message: fmt.Sprintf("Vector has been successfully deployed")})

			Expect(k8sClient.Status().Update(ctx, vectorDeployment)).To(Succeed())

			// check that the stage has status ready
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Name).To(Equal(StageDevSpecName))
				g.Expect(len(stage.Status.Conditions)).To(Equal(2))
				g.Expect(stage.Status.Conditions[1].Reason).To(Equal(common.StageReady))
				g.Expect(stage.Status.Conditions[1].Type).To(Equal(common.StageReady))
				g.Expect(stage.Status.Conditions[1].Status).To(Equal(metav1.ConditionTrue))
			}, timeout, interval).Should(Succeed())

		})
		It("should reuse vectorDeployment if a second stage references the same vector", func() {
			ctx := context.Background()
			// create dev stage
			createStage(ctx, k8sClient, StageDev, Namespace, StageDevSpecName, Vector001)

			// check that the stage has been created
			stageDev := &common.Stage{}
			stageDevLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageDevLookupKey, stageDev)).To(Succeed())
				g.Expect(stageDev.Spec.Name).To(Equal(StageDevSpecName))
			}, timeout, interval).Should(Succeed())

			// create test stage
			createStage(ctx, k8sClient, StageTest, Namespace, StageTestSpecName, Vector001)

			// check that the stage has been created
			stageTest := &common.Stage{}
			stageTestLookupKey := types.NamespacedName{Name: StageTest, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageTestLookupKey, stageTest)).To(Succeed())
				g.Expect(stageTest.Spec.Name).To(Equal(StageTestSpecName))
			}, timeout, interval).Should(Succeed())

			// check that two vectorDeploymentUsages have been created
			vectorDeploymentUsages := &landscape.VectorDeploymentUsageList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, vectorDeploymentUsages, client.InNamespace(Namespace))).To(Succeed())
				g.Expect(len(vectorDeploymentUsages.Items)).To(Equal(2))
				g.Expect(containsVectorDeploymentUsage(vectorDeploymentUsages.Items, StageDev, Vector001)).To(BeTrue())
				g.Expect(containsVectorDeploymentUsage(vectorDeploymentUsages.Items, StageTest, Vector001)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// check that the vectorDeployment has both stages and both vectorDeploymentUsages as owners
			vectorDeployment := &landscape.VectorDeployment{}
			vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(stageDev.Spec.Vector))
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
				g.Expect(len(vectorDeployment.GetOwnerReferences())).To(Equal(4))
				g.Expect(containsReference(vectorDeployment.GetOwnerReferences(), StageDev, common.StageKind)).To(BeTrue())
				g.Expect(containsReference(vectorDeployment.GetOwnerReferences(), StageTest, common.StageKind)).To(BeTrue())
				g.Expect(containsReference(vectorDeployment.GetOwnerReferences(), vectorDeploymentUsages.Items[0].Name, landscape.VectorDeploymentUsageKind)).To(BeTrue())
				g.Expect(containsReference(vectorDeployment.GetOwnerReferences(), vectorDeploymentUsages.Items[1].Name, landscape.VectorDeploymentUsageKind)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// mark vectorDeployment as deployed
			meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{Type: common.VectorDeployedCondition,
				Status: metav1.ConditionTrue, Reason: landscape.VectorDeployedCondition,
				Message: fmt.Sprintf("Vector has been successfully deployed")})

			Expect(k8sClient.Status().Update(ctx, vectorDeployment)).To(Succeed())

			// check that both stages have status ready
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageDevLookupKey, stageDev)).To(Succeed())
				g.Expect(len(stageDev.Status.Conditions)).To(Equal(2))
				g.Expect(stageDev.Status.Conditions[1].Reason).To(Equal(common.StageReady))
				g.Expect(stageDev.Status.Conditions[1].Type).To(Equal(common.StageReady))
				g.Expect(stageDev.Status.Conditions[1].Status).To(Equal(metav1.ConditionTrue))
				g.Expect(k8sClient.Get(ctx, stageTestLookupKey, stageTest)).To(Succeed())
				g.Expect(len(stageTest.Status.Conditions)).To(Equal(1))
				g.Expect(stageTest.Status.Conditions[0].Reason).To(Equal(common.StageReady))
				g.Expect(stageTest.Status.Conditions[0].Type).To(Equal(common.StageReady))
				g.Expect(stageTest.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			}, timeout, interval).Should(Succeed())
		})
		It("should delete old vectorDeploymentUsage and vectorDeployment if the stage vector has been changed", func() {
			ctx := context.Background()

			// create dev stage
			createStage(ctx, k8sClient, StageDev, Namespace, StageDevSpecName, Vector001)

			// check that the stage has been created
			stage := &common.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Name).To(Equal(StageDevSpecName))
			}, timeout, interval).Should(Succeed())

			// check that the vectorDeploymentUsage has been created and has valid properties
			vectorDeploymentUsages := &landscape.VectorDeploymentUsageList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, vectorDeploymentUsages, client.InNamespace(Namespace))).To(Succeed())
				g.Expect(len(vectorDeploymentUsages.Items)).To(Equal(1))
				g.Expect(vectorDeploymentUsages.Items[0].Spec.Vector).To(Equal(stage.Spec.Vector))
				g.Expect(len(vectorDeploymentUsages.Items[0].GetOwnerReferences())).To(Equal(1))
				g.Expect(vectorDeploymentUsages.Items[0].GetOwnerReferences()[0].Kind).To(Equal(common.StageKind))
				g.Expect(vectorDeploymentUsages.Items[0].GetOwnerReferences()[0].Name).To(Equal(StageDev))
			}, timeout, interval).Should(Succeed())

			// store the name for later
			oldVectorDeploymentUsageName := vectorDeploymentUsages.Items[0].Name

			// check that the vectorDeployment has been created and has valid properties
			vectorDeployment := &landscape.VectorDeployment{}
			vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(stage.Spec.Vector))
				g.Expect(len(vectorDeployment.GetOwnerReferences())).To(Equal(2))
				g.Expect(containsReference(vectorDeployment.GetOwnerReferences(), StageDev, common.StageKind)).To(BeTrue())
				g.Expect(containsReference(vectorDeployment.GetOwnerReferences(), vectorDeploymentUsages.Items[0].Name, landscape.VectorDeploymentUsageKind)).To(BeTrue())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			oldVectorDeploymentUid := vectorDeployment.UID

			// reload stage
			Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())

			// update stage with new vector
			stage.Spec.Vector = Vector002
			Expect(k8sClient.Update(ctx, stage)).To(Succeed())

			// and reload stage
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Name).To(Equal(StageDevSpecName))
				g.Expect(stage.Spec.Vector).To(Equal(Vector002))
			}, timeout, interval).Should(Succeed())

			// check that the old vectorDeploymentUsage has been deleted and a new one with the updated vector has been created instead
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, vectorDeploymentUsages, client.InNamespace(Namespace))).To(Succeed())
				g.Expect(len(vectorDeploymentUsages.Items)).To(Equal(1))
				g.Expect(vectorDeploymentUsages.Items[0].Spec.Vector).To(Equal(stage.Spec.Vector))
				g.Expect(vectorDeploymentUsages.Items[0].Spec.Vector).To(Equal(Vector002))
				g.Expect(vectorDeploymentUsages.Items[0].Name).ToNot(Equal(oldVectorDeploymentUsageName))
				g.Expect(len(vectorDeploymentUsages.Items[0].GetOwnerReferences())).To(Equal(1))
				g.Expect(vectorDeploymentUsages.Items[0].GetOwnerReferences()[0].Kind).To(Equal(common.StageKind))
				g.Expect(vectorDeploymentUsages.Items[0].GetOwnerReferences()[0].Name).To(Equal(StageDev))
			}, timeout, interval).Should(Succeed())

			// check that a new vectorDeployment with correct owner references has been created
			vectorDeploymentLookupKey = types.NamespacedName{Name: VectorName002, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(stage.Spec.Vector))
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector002))
				g.Expect(vectorDeployment.UID).ToNot(Equal(oldVectorDeploymentUid))
				g.Expect(len(vectorDeployment.GetOwnerReferences())).To(Equal(2))
				g.Expect(containsReference(vectorDeployment.GetOwnerReferences(), StageDev, common.StageKind)).To(BeTrue())
				g.Expect(containsReference(vectorDeployment.GetOwnerReferences(), vectorDeploymentUsages.Items[0].Name, landscape.VectorDeploymentUsageKind)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// check that the old vectorDeployment has been deleted
			vectorDeploymentLookupKey = types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError(errors.IsNotFound, "Should be a not found error"))
			}, timeout, interval).Should(Succeed())
		})

	})

	Context("When deleting a stage", func() {
		It("should delete vectorDeploymentUsage and vectorDeployment if no other owners exist", func() {
			ctx := context.Background()
			createStage(ctx, k8sClient, StageDev, Namespace, StageDevSpecName, Vector001)

			// check that the stage has been created and has valid properties
			stage := &common.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Name).To(Equal(StageDevSpecName))
				g.Expect(len(stage.Status.Conditions)).To(Equal(1))
				g.Expect(stage.Status.Conditions[0].Reason).To(Equal(common.VectorDeploymentCreatedCondition))
				g.Expect(stage.Status.Conditions[0].Type).To(Equal(common.VectorDeploymentCreatedCondition))
				g.Expect(stage.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			}, timeout, interval).Should(Succeed())

			// delete the stage
			deleteStage(ctx, k8sClient, stage)

			// check that vectorDeploymentUsage has been deleted
			vectorDeploymentUsages := &landscape.VectorDeploymentUsageList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, vectorDeploymentUsages, client.InNamespace(Namespace))).To(Succeed())
				g.Expect(len(vectorDeploymentUsages.Items)).To(BeZero())
			}, timeout, interval).Should(Succeed())

			// check that vectorDeployment has been deleted
			vectorDeployment := &landscape.VectorDeployment{}
			vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError(errors.IsNotFound, "Should be a not found error"))
			}, timeout, interval).Should(Succeed())
		})
	})
	It("should only delete vectorDeploymentUsage and owner references of deleted stage", func() {
		ctx := context.Background()
		// create dev stage
		createStage(ctx, k8sClient, StageDev, Namespace, StageDevSpecName, Vector001)

		// check that the stage has been created
		stageDev := &common.Stage{}
		stageDevLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, stageDevLookupKey, stageDev)).To(Succeed())
			g.Expect(stageDev.Spec.Name).To(Equal(StageDevSpecName))
		}, timeout, interval).Should(Succeed())

		// create test stage
		createStage(ctx, k8sClient, StageTest, Namespace, StageTestSpecName, Vector001)

		// check that the stage has been created
		stageTest := &common.Stage{}
		stageTestLookupKey := types.NamespacedName{Name: StageTest, Namespace: Namespace}
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, stageTestLookupKey, stageTest)).To(Succeed())
			g.Expect(stageTest.Spec.Name).To(Equal(StageTestSpecName))
		}, timeout, interval).Should(Succeed())

		// check that two vectorDeploymentUsages have been created
		vectorDeploymentUsages := &landscape.VectorDeploymentUsageList{}
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.List(ctx, vectorDeploymentUsages, client.InNamespace(Namespace))).To(Succeed())
			g.Expect(len(vectorDeploymentUsages.Items)).To(Equal(2))
			g.Expect(containsVectorDeploymentUsage(vectorDeploymentUsages.Items, StageDev, Vector001)).To(BeTrue())
			g.Expect(containsVectorDeploymentUsage(vectorDeploymentUsages.Items, StageTest, Vector001)).To(BeTrue())
		}, timeout, interval).Should(Succeed())

		// check that the vectorDeployment has both stages and both vectorDeploymentUsages as owners
		vectorDeployment := &landscape.VectorDeployment{}
		vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
			g.Expect(vectorDeployment.Spec.Vector).To(Equal(stageDev.Spec.Vector))
			g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
			g.Expect(len(vectorDeployment.GetOwnerReferences())).To(Equal(4))
			g.Expect(containsReference(vectorDeployment.GetOwnerReferences(), StageDev, common.StageKind)).To(BeTrue())
			g.Expect(containsReference(vectorDeployment.GetOwnerReferences(), StageTest, common.StageKind)).To(BeTrue())
			g.Expect(containsReference(vectorDeployment.GetOwnerReferences(), vectorDeploymentUsages.Items[0].Name, landscape.VectorDeploymentUsageKind)).To(BeTrue())
			g.Expect(containsReference(vectorDeployment.GetOwnerReferences(), vectorDeploymentUsages.Items[1].Name, landscape.VectorDeploymentUsageKind)).To(BeTrue())
		}, timeout, interval).Should(Succeed())

		// delete the stage
		deleteStage(ctx, k8sClient, stageDev)

		// check that vectorDeploymentUsage of the deleted stage has also been deleted
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.List(ctx, vectorDeploymentUsages, client.InNamespace(Namespace))).To(Succeed())
			g.Expect(len(vectorDeploymentUsages.Items)).To(Equal(1))
			g.Expect(containsVectorDeploymentUsage(vectorDeploymentUsages.Items, StageTest, Vector001)).To(BeTrue())
		}, timeout, interval).Should(Succeed())

		// check that vectorDeployment has removed owner references of deleted stage and vectorDeploymentUsage
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
			g.Expect(vectorDeployment.Spec.Vector).To(Equal(stageTest.Spec.Vector))
			g.Expect(len(vectorDeployment.GetOwnerReferences())).To(Equal(2))
			g.Expect(containsReference(vectorDeployment.GetOwnerReferences(), StageTest, common.StageKind)).To(BeTrue())
			g.Expect(containsReference(vectorDeployment.GetOwnerReferences(), vectorDeploymentUsages.Items[0].Name, landscape.VectorDeploymentUsageKind)).To(BeTrue())
		}, timeout, interval).Should(Succeed())
	})
})

func createStage(ctx context.Context, k8sClient client.Client, name string, namespace string, specName string, vectorName string) {
	stage := &common.Stage{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "common.konfidence.cloud/v1alpha1",
			Kind:       "Stage",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: common.StageSpec{
			Name:   specName,
			Vector: vectorName,
		},
	}

	Expect(k8sClient.Create(ctx, stage)).To(Succeed())
}

func getStage(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *common.Stage {
	stage := &common.Stage{}
	stageLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, stageLookupKey, stage)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch stage: %s", name)
	return stage
}

func deleteStage(ctx context.Context, k8sClient client.Client, stage *common.Stage) {
	err := k8sClient.Delete(ctx, stage)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete stage: %s", stage.Name)
}

func cleanupStage(k8sClient client.Client, stageName string, namespace string) {
	ctx := context.Background()
	stage := getStage(ctx, k8sClient, stageName, namespace, true)

	if stage != nil {
		deleteStage(ctx, k8sClient, stage)
	}
}

func containsReference(references []metav1.OwnerReference, name string, kind string) bool {
	for _, ref := range references {
		if ref.Kind == kind && ref.Name == name {
			return true
		}
	}

	return false
}

func containsVectorDeploymentUsage(items []landscape.VectorDeploymentUsage, stageName string, vectorName string) bool {
	for _, usage := range items {
		if usage.Spec.Vector == vectorName &&
			len(usage.GetOwnerReferences()) == 1 &&
			usage.GetOwnerReferences()[0].Kind == common.StageKind &&
			usage.GetOwnerReferences()[0].Name == stageName {
			return true
		}
	}

	return false
}
