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

	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const testStageVector = "ocm.example.com/test-vector:1.0.0"

const (
	stageSyncName      = "test-stagesync"
	stageSyncNamespace = "default"
	stageName          = "test-stage"
)

var _ = Describe("StageSync Controller", Ordered, func() {
	const (
		stageNamespace = "default"
		stageVector    = testStageVector
	)

	Context("When reconciling a stageSync on the remote cluster", func() {
		var stageSync *global.StageSync

		BeforeEach(func() {
			By("creating the StageSync on the remote cluster")
			stageSync = buildStageSync(stageNamespace, "landscape.konfidence.cloud/v1alpha1")
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
				}, &landscape.Stage{})
				g.Expect(client.IgnoreNotFound(err)).To(Succeed())
				g.Expect(err).To(MatchError(ContainSubstring("not found")))
			}).Should(Succeed())

			By("verifying the StageSync is fully removed from the remote cluster")
			Eventually(func(g Gomega) {
				err := remoteK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageSyncName,
					Namespace: stageSyncNamespace,
				}, &global.StageSync{})
				g.Expect(client.IgnoreNotFound(err)).To(Succeed())
				g.Expect(err).To(MatchError(ContainSubstring("not found")))
			}).Should(Succeed())
		})

		It("should successfully create a stage on the local cluster", func() {
			By("verifying the Stage is created on the local cluster")
			createdStage := &landscape.Stage{}
			Eventually(func(g Gomega) {
				g.Expect(localK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageName,
					Namespace: stageNamespace,
				}, createdStage)).To(Succeed())
			}).Should(Succeed())

			By("verifying the Stage has the correct spec")
			Expect(createdStage.Spec.Vector).To(Equal(stageVector))

			By("verifying the Stage carries the app.kubernetes.io/managed-by label")
			Expect(createdStage.GetLabels()).To(HaveKeyWithValue("app.kubernetes.io/managed-by", StageSyncControllerName))

			By("verifying the Stage carries the galaxy-stage-sync label pointing to the StageSync")
			expectedParentLabel := sanitizeLabelValue(types.NamespacedName{
				Name:      stageSyncName,
				Namespace: stageSyncNamespace,
			}.String())
			Expect(createdStage.GetLabels()).To(HaveKeyWithValue(galaxyStageSyncLabelKey, expectedParentLabel))

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
				g.Expect(condition.Reason).To(Equal(global.StageReconcileSuccessfulReason))
			}).Should(Succeed())

			By("verifying the StageSync status reflects the full stage status")
			Expect(updatedStageSync.Status.StageStatus.Raw).NotTo(BeEmpty())
			var reflectedStatus landscape.StageStatus
			Expect(json.Unmarshal(updatedStageSync.Status.StageStatus.Raw, &reflectedStatus)).To(Succeed())
			Expect(reflectedStatus).To(Equal(createdStage.Status))
		})
	})

	Context("When the target namespace does not exist on the local cluster", func() {
		const nonExistentNamespace = "non-existent-namespace"
		var stageSync *global.StageSync

		BeforeEach(func() {
			By("creating the StageSync on the remote cluster")
			stageSync = buildStageSync(nonExistentNamespace, "landscape.konfidence.cloud/v1alpha1")
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
			stageSync = buildStageSync(stageNamespace, "landscape.konfidence.cloud/v999alpha1")
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

	Context("When a StageSync is deleted", func() {
		var stageSync *global.StageSync

		BeforeEach(func() {
			By("creating the StageSync on the remote cluster")
			stageSync = buildStageSync(stageNamespace, "landscape.konfidence.cloud/v1alpha1")
			Expect(remoteK8sClient.Create(ctx, stageSync)).To(Succeed())

			By("waiting for the Stage to be created on the local cluster")
			Eventually(func(g Gomega) {
				g.Expect(localK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageName,
					Namespace: stageNamespace,
				}, &landscape.Stage{})).To(Succeed())
			}).Should(Succeed())
		})

		AfterEach(func() {
			By("ensuring the StageSync is cleaned up")
			stageSync := &global.StageSync{}
			err := remoteK8sClient.Get(ctx, types.NamespacedName{Name: stageSyncName, Namespace: stageSyncNamespace}, stageSync)
			if err == nil {
				// remove any leftover finalizers so the object can be deleted
				stageSync.Finalizers = nil
				Expect(remoteK8sClient.Update(ctx, stageSync)).To(Succeed())
				Expect(client.IgnoreNotFound(remoteK8sClient.Delete(ctx, stageSync))).To(Succeed())
			}

			By("ensuring the Stage is cleaned up")
			stage := &landscape.Stage{}
			err = localK8sClient.Get(ctx, types.NamespacedName{Name: stageName, Namespace: stageNamespace}, stage)
			if err == nil {
				stage.Finalizers = nil
				Expect(localK8sClient.Update(ctx, stage)).To(Succeed())
				Expect(client.IgnoreNotFound(localK8sClient.Delete(ctx, stage))).To(Succeed())
			}

			By("waiting for both objects to be fully gone")
			Eventually(func(g Gomega) {
				err := remoteK8sClient.Get(ctx, types.NamespacedName{
					Name: stageSyncName, Namespace: stageSyncNamespace,
				}, &global.StageSync{})
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}).Should(Succeed())
			Eventually(func(g Gomega) {
				err := localK8sClient.Get(ctx, types.NamespacedName{
					Name: stageName, Namespace: stageNamespace,
				}, &landscape.Stage{})
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}).Should(Succeed())
		})

		It("should delete the local Stage and remove the finalizer from the StageSync", func() {
			By("deleting the StageSync on the remote cluster")
			Expect(remoteK8sClient.Delete(ctx, stageSync)).To(Succeed())

			By("verifying the Stage is deleted from the local cluster")
			Eventually(func(g Gomega) {
				err := localK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageName,
					Namespace: stageNamespace,
				}, &landscape.Stage{})
				g.Expect(client.IgnoreNotFound(err)).To(Succeed())
				g.Expect(err).To(MatchError(ContainSubstring("not found")))
			}).Should(Succeed())

			By("verifying the StageSync is fully removed (finalizer released)")
			Eventually(func(g Gomega) {
				err := remoteK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageSyncName,
					Namespace: stageSyncNamespace,
				}, &global.StageSync{})
				g.Expect(client.IgnoreNotFound(err)).To(Succeed())
				g.Expect(err).To(MatchError(ContainSubstring("not found")))
			}).Should(Succeed())
		})

		It("should set StageDeletionBlocked condition when the local Stage is stuck on a finalizer", func() {
			By("adding a blocking finalizer to the local Stage")
			stage := &landscape.Stage{}
			Expect(localK8sClient.Get(ctx, types.NamespacedName{
				Name:      stageName,
				Namespace: stageNamespace,
			}, stage)).To(Succeed())
			stage.Finalizers = append(stage.Finalizers, "test.konfidence.cloud/block-deletion")
			Expect(localK8sClient.Update(ctx, stage)).To(Succeed())

			By("deleting the StageSync on the remote cluster")
			Expect(remoteK8sClient.Delete(ctx, stageSync)).To(Succeed())

			By("verifying the StageSync status has StageDeletionBlocked condition set")
			Eventually(func(g Gomega) {
				updated := &global.StageSync{}
				g.Expect(remoteK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageSyncName,
					Namespace: stageSyncNamespace,
				}, updated)).To(Succeed())
				condition := findCondition(updated.Status.Conditions, global.StageDeletedCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Reason).To(Equal(global.StageDeletionBlockedReason))
			}).Should(Succeed())

			By("verifying the StageSync finalizer is still present (not yet released)")
			stuck := &global.StageSync{}
			Expect(remoteK8sClient.Get(ctx, types.NamespacedName{
				Name:      stageSyncName,
				Namespace: stageSyncNamespace,
			}, stuck)).To(Succeed())
			Expect(stuck.Finalizers).To(ContainElement(syncControllerFinalizer))
		})
	})
})

// buildStageRaw builds a marshalled Stage object with the given name, namespace and apiVersion.
func buildStageRaw(name, namespace, apiVersion string) []byte {
	stage := &landscape.Stage{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       landscape.StageKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: landscape.StageSpec{
			Vector: testStageVector,
		},
	}
	raw, err := json.Marshal(stage)
	Expect(err).NotTo(HaveOccurred())
	return raw
}

// buildStageSync builds a StageSync object with a Stage template for the given parameters.
// The StageSync name, namespace and Stage name are fixed to the test constants.
func buildStageSync(stageNamespace, stageAPIVersion string) *global.StageSync {
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
