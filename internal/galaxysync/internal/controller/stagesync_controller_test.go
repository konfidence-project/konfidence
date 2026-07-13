package controller

import (
	"encoding/json"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
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
		var stageSync *konfidence.StageSync

		BeforeEach(func() {
			By("creating the StageSync on the remote cluster")
			stageSync = buildStageSync(stageNamespace, "konfidence.cloud/v1alpha1")
			Expect(remoteK8sClient.Create(ctx, stageSync)).To(Succeed())
		})

		//nolint:dupl
		AfterEach(func() {
			By("cleaning up the StageSync from the remote cluster")
			Expect(remoteK8sClient.Delete(ctx, stageSync)).To(Succeed())

			By("verifying the Stage is deleted from the local cluster after the StageSync is removed")
			Eventually(func(g Gomega) {
				err := localK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageName,
					Namespace: stageNamespace,
				}, &konfidence.Stage{})
				g.Expect(client.IgnoreNotFound(err)).To(Succeed())
				g.Expect(err).To(MatchError(ContainSubstring("not found")))
			}).Should(Succeed())

			By("verifying the StageSync is fully removed from the remote cluster")
			Eventually(func(g Gomega) {
				err := remoteK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageSyncName,
					Namespace: stageSyncNamespace,
				}, &konfidence.StageSync{})
				g.Expect(client.IgnoreNotFound(err)).To(Succeed())
				g.Expect(err).To(MatchError(ContainSubstring("not found")))
			}).Should(Succeed())
		})

		It("should successfully create a stage on the local cluster", func() {
			By("verifying the Stage is created on the local cluster")
			createdStage := &konfidence.Stage{}
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
			updatedStageSync := &konfidence.StageSync{}
			Eventually(func(g Gomega) {
				g.Expect(remoteK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageSyncName,
					Namespace: stageSyncNamespace,
				}, updatedStageSync)).To(Succeed())
				condition := findCondition(updatedStageSync.Status.Conditions, konfidence.StageSyncAppliedCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(condition.Reason).To(Equal(konfidence.StageReconcileSuccessfulReason))
			}).Should(Succeed())

			By("verifying the StageSync status reflects the full stage status")
			Expect(updatedStageSync.Status.StageStatus.Raw).NotTo(BeEmpty())
			var reflectedStatus konfidence.StageStatus
			Expect(json.Unmarshal(updatedStageSync.Status.StageStatus.Raw, &reflectedStatus)).To(Succeed())
			Expect(reflectedStatus).To(Equal(createdStage.Status))

			By("verifying the remote StageSync carries the synced-by-star label")
			expectedLabelKey := stageSyncedByLabelPrefix + "test-star"
			Expect(updatedStageSync.GetLabels()).To(HaveKeyWithValue(expectedLabelKey, "true"))
		})
	})

	Context("When the StageSync template is updated", func() {
		var stageSync *konfidence.StageSync

		BeforeEach(func() {
			By("creating the StageSync on the remote cluster")
			stageSync = buildStageSync(stageNamespace, "konfidence.cloud/v1alpha1")
			Expect(remoteK8sClient.Create(ctx, stageSync)).To(Succeed())

			By("waiting for the Stage to be created on the local cluster")
			Eventually(func(g Gomega) {
				g.Expect(localK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageName,
					Namespace: stageNamespace,
				}, &konfidence.Stage{})).To(Succeed())
			}).Should(Succeed())
		})

		AfterEach(func() {
			By("cleaning up the StageSync from the remote cluster")
			Expect(remoteK8sClient.Delete(ctx, stageSync)).To(Succeed())

			By("waiting for the Stage to be deleted from the local cluster")
			Eventually(func(g Gomega) {
				err := localK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageName,
					Namespace: stageNamespace,
				}, &konfidence.Stage{})
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}).Should(Succeed())

			By("waiting for the StageSync to be fully removed")
			Eventually(func(g Gomega) {
				err := remoteK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageSyncName,
					Namespace: stageSyncNamespace,
				}, &konfidence.StageSync{})
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}).Should(Succeed())
		})

		It("should update the local Stage spec when the template vector changes", func() {
			const updatedVector = "ocm.example.com/test-vector:2.0.0"

			By("updating the StageSync template with a new vector")
			Eventually(func(g Gomega) {
				fresh := &konfidence.StageSync{}
				g.Expect(remoteK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageSyncName,
					Namespace: stageSyncNamespace,
				}, fresh)).To(Succeed())

				fresh.Spec.StageTemplate = runtime.RawExtension{
					Raw: buildStageRaw(stageName, stageNamespace, "konfidence.cloud/v1alpha1", updatedVector),
				}
				g.Expect(remoteK8sClient.Update(ctx, fresh)).To(Succeed())
			}).Should(Succeed())

			By("verifying the local Stage spec is updated")
			Eventually(func(g Gomega) {
				stage := &konfidence.Stage{}
				g.Expect(localK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageName,
					Namespace: stageNamespace,
				}, stage)).To(Succeed())
				g.Expect(stage.Spec.Vector).To(Equal(updatedVector))
			}).Should(Succeed())

			By("verifying the StageSync status condition remains Applied=True")
			Eventually(func(g Gomega) {
				updated := &konfidence.StageSync{}
				g.Expect(remoteK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageSyncName,
					Namespace: stageSyncNamespace,
				}, updated)).To(Succeed())
				condition := findCondition(updated.Status.Conditions, konfidence.StageSyncAppliedCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(condition.Reason).To(Equal(konfidence.StageReconcileSuccessfulReason))
			}).Should(Succeed())
		})
	})

	Context("When a pre-existing unmanaged Stage conflicts", func() {
		var stageSync *konfidence.StageSync
		var unmanagedStage *konfidence.Stage

		BeforeEach(func() {
			By("creating an unmanaged Stage on the local cluster")
			unmanagedStage = &konfidence.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      stageName,
					Namespace: stageNamespace,
				},
				Spec: konfidence.StageSpec{
					Vector: "ocm.example.com/other:1.0.0",
				},
			}
			Expect(localK8sClient.Create(ctx, unmanagedStage)).To(Succeed())

			By("creating the StageSync on the remote cluster")
			stageSync = buildStageSync(stageNamespace, "konfidence.cloud/v1alpha1")
			Expect(remoteK8sClient.Create(ctx, stageSync)).To(Succeed())
		})

		AfterEach(func() {
			By("cleaning up the StageSync from the remote cluster")
			Expect(client.IgnoreNotFound(remoteK8sClient.Delete(ctx, stageSync))).To(Succeed())
			Eventually(func(g Gomega) {
				err := remoteK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageSyncName,
					Namespace: stageSyncNamespace,
				}, &konfidence.StageSync{})
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}).Should(Succeed())

			By("cleaning up the unmanaged Stage from the local cluster")
			Expect(client.IgnoreNotFound(localK8sClient.Delete(ctx, unmanagedStage))).To(Succeed())
			Eventually(func(g Gomega) {
				err := localK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageName,
					Namespace: stageNamespace,
				}, &konfidence.Stage{})
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}).Should(Succeed())
		})

		It("should set Applied=False with reason ConflictWithUnmanagedStage", func() {
			Eventually(func(g Gomega) {
				updated := &konfidence.StageSync{}
				g.Expect(remoteK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageSyncName,
					Namespace: stageSyncNamespace,
				}, updated)).To(Succeed())
				condition := findCondition(updated.Status.Conditions, konfidence.StageSyncAppliedCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionFalse))
				g.Expect(condition.Reason).To(Equal(konfidence.ConflictWithUnmanagedStageReason))
			}).Should(Succeed())

			By("verifying the unmanaged Stage is NOT modified")
			stage := &konfidence.Stage{}
			Expect(localK8sClient.Get(ctx, types.NamespacedName{
				Name:      stageName,
				Namespace: stageNamespace,
			}, stage)).To(Succeed())
			Expect(stage.Spec.Vector).To(Equal("ocm.example.com/other:1.0.0"))
			Expect(stage.GetLabels()).NotTo(HaveKey(managedByLabelKey))
		})
	})

	Context("When the target namespace does not exist on the local cluster", func() {
		const nonExistentNamespace = "non-existent-namespace"
		var stageSync *konfidence.StageSync

		BeforeEach(func() {
			By("creating the StageSync on the remote cluster")
			stageSync = buildStageSync(nonExistentNamespace, "konfidence.cloud/v1alpha1")
			Expect(remoteK8sClient.Create(ctx, stageSync)).To(Succeed())
		})

		AfterEach(func() {
			By("cleaning up the StageSync from the remote cluster")
			Expect(remoteK8sClient.Delete(ctx, stageSync)).To(Succeed())
		})

		It("should set the StageSync status to Applied=False with reason NamespaceNotFound", func() {
			Eventually(func(g Gomega) {
				updated := &konfidence.StageSync{}
				g.Expect(remoteK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageSyncName,
					Namespace: stageSyncNamespace,
				}, updated)).To(Succeed())
				condition := findCondition(updated.Status.Conditions, konfidence.StageSyncAppliedCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionFalse))
				g.Expect(condition.Reason).To(Equal(konfidence.NamespaceNotFoundReason))
			}).Should(Succeed())
		})
	})

	Context("When the Stage API version in the template is not served on the local cluster", func() {
		var stageSync *konfidence.StageSync

		BeforeEach(func() {
			By("creating the StageSync on the remote cluster")
			stageSync = buildStageSync(stageNamespace, "konfidence.cloud/v999alpha1")
			Expect(remoteK8sClient.Create(ctx, stageSync)).To(Succeed())
		})

		AfterEach(func() {
			By("cleaning up the StageSync from the remote cluster")
			Expect(remoteK8sClient.Delete(ctx, stageSync)).To(Succeed())
		})

		It("should set the StageSync status to Applied=False with reason APIVersionNotSupported", func() {
			Eventually(func(g Gomega) {
				updated := &konfidence.StageSync{}
				g.Expect(remoteK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageSyncName,
					Namespace: stageSyncNamespace,
				}, updated)).To(Succeed())
				condition := findCondition(updated.Status.Conditions, konfidence.StageSyncAppliedCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionFalse))
				g.Expect(condition.Reason).To(Equal(konfidence.APIVersionNotSupportedReason))
			}).Should(Succeed())
		})
	})

	Context("When a StageSync is deleted", func() {
		var stageSync *konfidence.StageSync

		BeforeEach(func() {
			By("creating the StageSync on the remote cluster")
			stageSync = buildStageSync(stageNamespace, "konfidence.cloud/v1alpha1")
			Expect(remoteK8sClient.Create(ctx, stageSync)).To(Succeed())

			By("waiting for the Stage to be created on the local cluster")
			Eventually(func(g Gomega) {
				g.Expect(localK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageName,
					Namespace: stageNamespace,
				}, &konfidence.Stage{})).To(Succeed())
			}).Should(Succeed())
		})

		AfterEach(func() {
			By("ensuring the StageSync is cleaned up")
			stageSync := &konfidence.StageSync{}
			err := remoteK8sClient.Get(ctx, types.NamespacedName{Name: stageSyncName, Namespace: stageSyncNamespace}, stageSync)
			if err == nil {
				// remove any leftover finalizers so the object can be deleted
				stageSync.Finalizers = nil
				Expect(remoteK8sClient.Update(ctx, stageSync)).To(Succeed())
				Expect(client.IgnoreNotFound(remoteK8sClient.Delete(ctx, stageSync))).To(Succeed())
			}

			By("ensuring the Stage is cleaned up")
			stage := &konfidence.Stage{}
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
				}, &konfidence.StageSync{})
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}).Should(Succeed())
			Eventually(func(g Gomega) {
				err := localK8sClient.Get(ctx, types.NamespacedName{
					Name: stageName, Namespace: stageNamespace,
				}, &konfidence.Stage{})
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}).Should(Succeed())
		})

		//nolint:dupl
		It("should delete the local Stage and remove the finalizer from the StageSync", func() {
			By("deleting the StageSync on the remote cluster")
			Expect(remoteK8sClient.Delete(ctx, stageSync)).To(Succeed())

			By("verifying the Stage is deleted from the local cluster")
			Eventually(func(g Gomega) {
				err := localK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageName,
					Namespace: stageNamespace,
				}, &konfidence.Stage{})
				g.Expect(client.IgnoreNotFound(err)).To(Succeed())
				g.Expect(err).To(MatchError(ContainSubstring("not found")))
			}).Should(Succeed())

			By("verifying the StageSync is fully removed (finalizer released)")
			Eventually(func(g Gomega) {
				err := remoteK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageSyncName,
					Namespace: stageSyncNamespace,
				}, &konfidence.StageSync{})
				g.Expect(client.IgnoreNotFound(err)).To(Succeed())
				g.Expect(err).To(MatchError(ContainSubstring("not found")))
			}).Should(Succeed())
		})

		It("should set StageDeletionBlocked condition when the local Stage is stuck on a finalizer", func() {
			By("adding a blocking finalizer to the local Stage")
			stage := &konfidence.Stage{}
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
				updated := &konfidence.StageSync{}
				g.Expect(remoteK8sClient.Get(ctx, types.NamespacedName{
					Name:      stageSyncName,
					Namespace: stageSyncNamespace,
				}, updated)).To(Succeed())
				condition := findCondition(updated.Status.Conditions, konfidence.StageDeletedCondition)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Reason).To(Equal(konfidence.StageDeletionBlockedReason))
			}).Should(Succeed())

			By("verifying the StageSync finalizer is still present (not yet released)")
			stuck := &konfidence.StageSync{}
			Expect(remoteK8sClient.Get(ctx, types.NamespacedName{
				Name:      stageSyncName,
				Namespace: stageSyncNamespace,
			}, stuck)).To(Succeed())
			Expect(stuck.Finalizers).To(ContainElement(syncControllerFinalizer))
		})
	})
})

// buildStageRaw builds a marshalled Stage object with the given name, namespace, apiVersion and vector.
func buildStageRaw(name, namespace, apiVersion, vector string) []byte {
	stage := &konfidence.Stage{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       konfidence.StageKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: konfidence.StageSpec{
			Vector: vector,
		},
	}
	raw, err := json.Marshal(stage)
	Expect(err).NotTo(HaveOccurred())
	return raw
}

// buildStageSync builds a StageSync object with a Stage template for the given parameters.
// The StageSync name, namespace and Stage name are fixed to the test constants.
func buildStageSync(stageNamespace, stageAPIVersion string) *konfidence.StageSync {
	return &konfidence.StageSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stageSyncName,
			Namespace: stageSyncNamespace,
		},
		Spec: konfidence.StageSyncSpec{
			StageTemplate: runtime.RawExtension{Raw: buildStageRaw(stageName, stageNamespace, stageAPIVersion, testStageVector)},
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
