//nolint:staticcheck // ST1001: allow dot-import for test specs using Ginkgo/Gomega
package controller

import (
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Stalled condition", func() {
	const artifactName = "artifact-a"

	newVectorDeployment := func(generation int64) *konfidence.VectorDeployment {
		vectorDeployment := &konfidence.VectorDeployment{}
		vectorDeployment.Name = "test-vector"
		vectorDeployment.Generation = generation

		return vectorDeployment
	}

	stalledCondition := func(vectorDeployment *konfidence.VectorDeployment) *metav1.Condition {
		return meta.FindStatusCondition(vectorDeployment.Status.Conditions, konfidence.StalledCondition)
	}

	Context("writing the condition", func() {
		// A healthy reconcile must leave Stalled present and False, not absent.
		It("should write Stalled=False when nothing blocks", func() {
			vectorDeployment := newVectorDeployment(3)

			clearStalled(vectorDeployment)

			condition := stalledCondition(vectorDeployment)
			Expect(condition).ToNot(BeNil())
			Expect(condition.Status).To(Equal(metav1.ConditionFalse))
			Expect(condition.ObservedGeneration).To(Equal(int64(3)))
		})

		It("should force Ready=False whenever Stalled is True", func() {
			vectorDeployment := newVectorDeployment(1)
			meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
				Type:               konfidence.VectorReadyCondition,
				Status:             metav1.ConditionTrue,
				Reason:             konfidence.VectorReadyCondition,
				ObservedGeneration: 1,
			})

			setStalled(vectorDeployment, konfidence.VectorDeploymentStalledReasonArtifactDeploymentNamingCollision, "collision")

			Expect(meta.IsStatusConditionTrue(vectorDeployment.Status.Conditions, konfidence.StalledCondition)).To(BeTrue())
			Expect(meta.IsStatusConditionTrue(vectorDeployment.Status.Conditions, konfidence.VectorReadyCondition)).To(BeFalse())
			readyCondition := meta.FindStatusCondition(vectorDeployment.Status.Conditions, konfidence.VectorReadyCondition)
			Expect(readyCondition.Reason).To(Equal(konfidence.VectorReadyReasonStalled))
			Expect(readyCondition.Message).To(Equal("Vector Stalled"))
		})

		// A reason change while still True must update in place, not drop and re-add the entry.
		It("should keep a single entry across a reason transition", func() {
			vectorDeployment := newVectorDeployment(1)

			setStalled(vectorDeployment, konfidence.VectorDeploymentStalledReasonArtifactDeploymentNamingCollision, "collision")
			setStalled(vectorDeployment, konfidence.VectorDeploymentStalledReasonArtifactDeploymentStalled, "artifact deployment stalled")

			var count int
			for _, condition := range vectorDeployment.Status.Conditions {
				if condition.Type == konfidence.StalledCondition {
					count++
				}
			}
			Expect(count).To(Equal(1))
			Expect(stalledCondition(vectorDeployment).Reason).To(Equal(konfidence.VectorDeploymentStalledReasonArtifactDeploymentStalled))
		})

		It("should flip back to False once the cause resolves", func() {
			vectorDeployment := newVectorDeployment(1)

			setStalled(vectorDeployment, konfidence.VectorDeploymentStalledReasonArtifactDeploymentStalled, "artifact deployment stalled")
			clearStalled(vectorDeployment)

			Expect(stalledCondition(vectorDeployment).Status).To(Equal(metav1.ConditionFalse))
		})

		It("should drop the Ready=False that the stall left behind once it resolves", func() {
			vectorDeployment := newVectorDeployment(1)

			setStalled(vectorDeployment, konfidence.VectorDeploymentStalledReasonArtifactDeploymentStalled, "artifact deployment stalled")
			clearStalled(vectorDeployment)

			Expect(meta.FindStatusCondition(vectorDeployment.Status.Conditions, konfidence.VectorReadyCondition)).To(BeNil())
		})

		It("should leave a Ready condition it did not write alone", func() {
			vectorDeployment := newVectorDeployment(1)
			meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
				Type:   konfidence.VectorReadyCondition,
				Status: metav1.ConditionTrue,
				Reason: konfidence.VectorReadyCondition,
			})

			clearStalled(vectorDeployment)

			Expect(meta.IsStatusConditionTrue(vectorDeployment.Status.Conditions, konfidence.VectorReadyCondition)).To(BeTrue())
		})

		// Spec fixed, generation bumped, controller not run yet: the condition must stay
		// detectably stale rather than read as a fresh verdict on the new spec.
		It("should carry the generation the stall was computed from", func() {
			vectorDeployment := newVectorDeployment(7)
			setStalled(vectorDeployment, konfidence.VectorDeploymentStalledReasonArtifactDeploymentStalled, "artifact deployment stalled")

			vectorDeployment.Generation = 8

			condition := stalledCondition(vectorDeployment)
			Expect(condition.Status).To(Equal(metav1.ConditionTrue))
			Expect(condition.ObservedGeneration).To(Equal(int64(7)))
			Expect(condition.ObservedGeneration).To(BeNumerically("<", vectorDeployment.Generation))
		})
	})

	Context("aggregating stalled ArtifactDeployments", func() {
		DescribeTable("should recognise a stalled ArtifactDeployment",
			func(conditions []metav1.Condition, expectStalled bool) {
				artifactDeployment := &konfidence.ArtifactDeployment{}
				artifactDeployment.Name = artifactName
				artifactDeployment.Status.Conditions = conditions

				stalledArtifactDeployment, ok := collectStalledArtifactDeployment(artifactDeployment)

				Expect(ok).To(Equal(expectStalled))
				if expectStalled {
					Expect(stalledArtifactDeployment.name).To(Equal(artifactName))
				}
			},
			Entry("with no conditions", nil, false),
			Entry("when not stalled", []metav1.Condition{{
				Type:   konfidence.StalledCondition,
				Status: metav1.ConditionFalse,
				Reason: konfidence.StalledReasonNotStalled,
			}}, false),
			Entry("when stalled", []metav1.Condition{{
				Type:    konfidence.StalledCondition,
				Status:  metav1.ConditionTrue,
				Reason:  konfidence.ArtifactDeploymentStalledReasonManifestMissing,
				Message: "no konfidence manifest",
			}}, true),
		)

		// The named ArtifactDeployment must not depend on observation order, or the message flaps.
		It("should pick the same ArtifactDeployment regardless of input order", func() {
			forward := []stalledArtifactDeployment{
				{name: artifactName, reason: konfidence.ArtifactDeploymentStalledReasonManifestMissing},
				{name: "artifact-b", reason: konfidence.ArtifactDeploymentStalledReasonDeploymentResultNotUnique},
				{name: "artifact-c", reason: konfidence.ArtifactDeploymentStalledReasonManifestMissing},
			}
			reversed := []stalledArtifactDeployment{forward[2], forward[1], forward[0]}

			first, ok := pickStalledArtifactDeployment(forward)
			Expect(ok).To(BeTrue())
			second, ok := pickStalledArtifactDeployment(reversed)
			Expect(ok).To(BeTrue())

			Expect(first.name).To(Equal(second.name))
			Expect(first.name).To(Equal(artifactName))
		})

		It("should keep lastTransitionTime while the same stall persists across reconciles", func() {
			vectorDeployment := newVectorDeployment(1)
			stalledArtifactDeployment := &konfidence.ArtifactDeployment{}
			stalledArtifactDeployment.Name = artifactName
			stalledArtifactDeployment.Status.Conditions = []metav1.Condition{{
				Type:   konfidence.StalledCondition,
				Status: metav1.ConditionTrue,
				Reason: konfidence.ArtifactDeploymentStalledReasonManifestMissing,
			}}

			reportStalledArtifactDeployments(vectorDeployment, []*konfidence.ArtifactDeployment{stalledArtifactDeployment})
			since := metav1.NewTime(time.Now().Add(-time.Hour))
			stalledCondition(vectorDeployment).LastTransitionTime = since

			reportStalledArtifactDeployments(vectorDeployment, []*konfidence.ArtifactDeployment{stalledArtifactDeployment})

			Expect(stalledCondition(vectorDeployment).LastTransitionTime).To(Equal(since))
		})

		It("should clear Stalled when no ArtifactDeployment is stalled", func() {
			vectorDeployment := newVectorDeployment(1)
			setStalled(vectorDeployment, konfidence.VectorDeploymentStalledReasonArtifactDeploymentStalled, "artifact deployment stalled")

			reportStalledArtifactDeployments(vectorDeployment, []*konfidence.ArtifactDeployment{{}})

			Expect(stalledCondition(vectorDeployment).Status).To(Equal(metav1.ConditionFalse))
		})

		DescribeTable("should not count a stalled ArtifactDeployment as ready",
			func(ready, stalled metav1.ConditionStatus, expectReady bool) {
				artifactDeployment := &konfidence.ArtifactDeployment{}
				artifactDeployment.Status.Conditions = []metav1.Condition{
					{Type: konfidence.ArtifactDeploymentReadyCondition, Status: ready, Reason: "Test"},
					{Type: konfidence.StalledCondition, Status: stalled, Reason: "Test"},
				}

				Expect(isArtifactDeploymentReady(artifactDeployment)).To(Equal(expectReady))
			},
			Entry("ready and not stalled", metav1.ConditionTrue, metav1.ConditionFalse, true),
			Entry("ready and stalled", metav1.ConditionTrue, metav1.ConditionTrue, false),
			Entry("not ready", metav1.ConditionFalse, metav1.ConditionFalse, false),
		)

		It("should pick nothing from an empty set", func() {
			_, ok := pickStalledArtifactDeployment(nil)
			Expect(ok).To(BeFalse())
		})

		It("should name the ArtifactDeployment and its reason in the message", func() {
			stalledArtifactDeployment := stalledArtifactDeployment{
				name:    artifactName,
				reason:  konfidence.ArtifactDeploymentStalledReasonManifestMissing,
				message: "no konfidence manifest in artifact",
			}

			Expect(stalledArtifactDeploymentMessage(stalledArtifactDeployment, 1)).To(SatisfyAll(
				ContainSubstring(artifactName),
				ContainSubstring(konfidence.ArtifactDeploymentStalledReasonManifestMissing),
			))
			Expect(stalledArtifactDeploymentMessage(stalledArtifactDeployment, 3)).To(ContainSubstring("3 artifact deployments are stalled"))
		})
	})
})
