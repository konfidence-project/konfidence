/*
Copyright 2026.

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
	"encoding/json"

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const testStageVector = "ocm.example.com/test-vector:1.0.0"

var _ = Describe("StageSync Controller", Ordered, func() {
	const (
		stageSyncName      = "test-stagesync"
		stageSyncNamespace = "default"
		stageName          = "test-stage"
		stageNamespace     = "default"
		stageVector        = testStageVector
	)

	Context("When reconciling a stageSync on the remote cluster", func() {
		var stageSync *global.StageSync

		BeforeEach(func() {
			By("creating the StageSync on the remote cluster")
			stageSync = buildStageSync(stageSyncName, stageSyncNamespace, stageName, stageNamespace, "common.konfidence.cloud/v1alpha1")
			Expect(remoteK8sClient.Create(ctx, stageSync)).To(Succeed())
		})

		AfterEach(func() {
			By("cleaning up the StageSync from the remote cluster")
			Expect(remoteK8sClient.Delete(ctx, stageSync)).To(Succeed())

			By("verifying the Stage is deleted from the local cluster after the StageSync is removed")
			Eventually(func(g Gomega) {
				err := localK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageName,
					Namespace: stageNamespace,
				}, &common.Stage{})
				g.Expect(client.IgnoreNotFound(err)).To(Succeed())
				g.Expect(err).To(MatchError(ContainSubstring("not found")))
			}).Should(Succeed())
		})

		It("should successfully create a stage on the local cluster", func() {
			By("verifying the Stage is created on the local cluster")
			createdStage := &common.Stage{}
			Eventually(func(g Gomega) {
				g.Expect(localK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageName,
					Namespace: stageNamespace,
				}, createdStage)).To(Succeed())
			}).Should(Succeed())

			By("verifying the Stage has the correct spec")
			Expect(createdStage.Spec.Vector).To(Equal(stageVector))

			By("verifying the Stage carries the managed-by label pointing to the StageSync")
			expectedLabelValue := getLabelValue(types.NamespacedName{
				Name:      stageSyncName,
				Namespace: stageSyncNamespace,
			})
			Expect(createdStage.GetLabels()).To(HaveKeyWithValue("managed-by", expectedLabelValue))

			By("verifying the StageSync status condition is set to Applied=True on the remote cluster")
			updatedStageSync := &global.StageSync{}
			Eventually(func(g Gomega) {
				g.Expect(remoteK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageSyncName,
					Namespace: stageSyncNamespace,
				}, updatedStageSync)).To(Succeed())
				condition := findCondition(updatedStageSync.Status.Conditions, global.StageSyncAppliedCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(condition.Reason).To(Equal(global.StageCreationSuccessfulReason))
			}).Should(Succeed())
		})
	})

	Context("When the target namespace does not exist on the local cluster", func() {
		const nonExistentNamespace = "non-existent-namespace"
		var stageSync *global.StageSync

		BeforeEach(func() {
			By("creating the StageSync on the remote cluster")
			stageSync = buildStageSync(stageSyncName, stageSyncNamespace, stageName, nonExistentNamespace, "common.konfidence.cloud/v1alpha1")
			Expect(remoteK8sClient.Create(ctx, stageSync)).To(Succeed())
		})

		AfterEach(func() {
			By("cleaning up the StageSync from the remote cluster")
			Expect(remoteK8sClient.Delete(ctx, stageSync)).To(Succeed())
		})

		It("should set the StageSync status to Applied=False with reason NamespaceNotFound", func() {
			Eventually(func(g Gomega) {
				updated := &global.StageSync{}
				g.Expect(remoteK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageSyncName,
					Namespace: stageSyncNamespace,
				}, updated)).To(Succeed())
				condition := findCondition(updated.Status.Conditions, global.StageSyncAppliedCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionFalse))
				g.Expect(condition.Reason).To(Equal(global.NamespaceNotFoundReason))
			}).Should(Succeed())
		})
	})

	Context("When the Stage API version in the template is not served on the local cluster", func() {
		var stageSync *global.StageSync

		BeforeEach(func() {
			By("creating the StageSync on the remote cluster")
			stageSync = buildStageSync(stageSyncName, stageSyncNamespace, stageName, stageNamespace, "common.konfidence.cloud/v999alpha1")
			Expect(remoteK8sClient.Create(ctx, stageSync)).To(Succeed())
		})

		AfterEach(func() {
			By("cleaning up the StageSync from the remote cluster")
			Expect(remoteK8sClient.Delete(ctx, stageSync)).To(Succeed())
		})

		It("should set the StageSync status to Applied=False with reason APIVersionNotSupported", func() {
			Eventually(func(g Gomega) {
				updated := &global.StageSync{}
				g.Expect(remoteK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageSyncName,
					Namespace: stageSyncNamespace,
				}, updated)).To(Succeed())
				condition := findCondition(updated.Status.Conditions, global.StageSyncAppliedCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionFalse))
				g.Expect(condition.Reason).To(Equal(global.APIVersionNotSupportedReason))
			}).Should(Succeed())
		})
	})
})

// buildStageRaw builds a marshalled Stage object with the given name, namespace and apiVersion.
func buildStageRaw(name, namespace, apiVersion string) []byte {
	stage := &common.Stage{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       common.StageKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: common.StageSpec{
			Vector: testStageVector,
		},
	}
	raw, err := json.Marshal(stage)
	Expect(err).NotTo(HaveOccurred())
	return raw
}

// buildStageSync builds a StageSync object with a Stage template for the given parameters.
func buildStageSync(stageSyncName, stageSyncNamespace, stageName, stageNamespace, stageAPIVersion string) *global.StageSync {
	return &global.StageSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stageSyncName,
			Namespace: stageSyncNamespace,
		},
		Spec: global.StageSyncSpec{
			StageTemplate: runtime.RawExtension{Raw: buildStageRaw(stageName, stageNamespace, stageAPIVersion)},
		},
	}
}

// findCondition returns the condition with the given type, or nil if not found.
func findCondition(conditions []metav1.Condition, condType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == condType {
			return &conditions[i]
		}
	}
	return nil
}
