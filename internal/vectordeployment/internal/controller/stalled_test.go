//nolint:staticcheck // ST1001: allow dot-import for test specs using Ginkgo/Gomega
package controller

import (
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

			setStalled(vectorDeployment, konfidence.StalledReasonArtifactDeploymentNamingCollision, "collision")

			Expect(meta.IsStatusConditionTrue(vectorDeployment.Status.Conditions, konfidence.StalledCondition)).To(BeTrue())
			Expect(meta.IsStatusConditionTrue(vectorDeployment.Status.Conditions, konfidence.VectorReadyCondition)).To(BeFalse())
		})

		// A reason change while still True must update in place, not drop and re-add the entry.
		It("should keep a single entry across a reason transition", func() {
			vectorDeployment := newVectorDeployment(1)

			setStalled(vectorDeployment, konfidence.StalledReasonArtifactDeploymentNamingCollision, "collision")
			setStalled(vectorDeployment, konfidence.StalledReasonChildArtifactDeploymentStalled, "child blocked")

			var count int
			for _, condition := range vectorDeployment.Status.Conditions {
				if condition.Type == konfidence.StalledCondition {
					count++
				}
			}
			Expect(count).To(Equal(1))
			Expect(stalledCondition(vectorDeployment).Reason).To(Equal(konfidence.StalledReasonChildArtifactDeploymentStalled))
		})

		It("should flip back to False once the cause resolves", func() {
			vectorDeployment := newVectorDeployment(1)

			setStalled(vectorDeployment, konfidence.StalledReasonChildArtifactDeploymentStalled, "child blocked")
			clearStalled(vectorDeployment)

			Expect(stalledCondition(vectorDeployment).Status).To(Equal(metav1.ConditionFalse))
		})

		// Spec fixed, generation bumped, controller not run yet: the condition must stay
		// detectably stale rather than read as a fresh verdict on the new spec.
		It("should carry the generation the stall was computed from", func() {
			vectorDeployment := newVectorDeployment(7)
			setStalled(vectorDeployment, konfidence.StalledReasonChildArtifactDeploymentStalled, "child blocked")

			vectorDeployment.Generation = 8

			condition := stalledCondition(vectorDeployment)
			Expect(condition.Status).To(Equal(metav1.ConditionTrue))
			Expect(condition.ObservedGeneration).To(Equal(int64(7)))
			Expect(condition.ObservedGeneration).To(BeNumerically("<", vectorDeployment.Generation))
		})
	})

	Context("aggregating stalled children", func() {
		DescribeTable("should recognise a stalled child",
			func(conditions []metav1.Condition, expectStalled bool) {
				artifactDeployment := &konfidence.ArtifactDeployment{}
				artifactDeployment.Name = artifactName
				artifactDeployment.Status.Conditions = conditions

				child, ok := collectStalledChild(artifactDeployment)

				Expect(ok).To(Equal(expectStalled))
				if expectStalled {
					Expect(child.name).To(Equal(artifactName))
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
				Reason:  konfidence.StalledReasonManifestMissing,
				Message: "no konfidence manifest",
			}}, true),
		)

		// The named child must not depend on observation order, or the message flaps.
		It("should pick the same child regardless of input order", func() {
			forward := []stalledChild{
				{name: artifactName, reason: konfidence.StalledReasonManifestMissing},
				{name: "artifact-b", reason: konfidence.StalledReasonDeploymentResultNotUnique},
				{name: "artifact-c", reason: konfidence.StalledReasonManifestMissing},
			}
			reversed := []stalledChild{forward[2], forward[1], forward[0]}

			first, ok := pickStalledChild(forward)
			Expect(ok).To(BeTrue())
			second, ok := pickStalledChild(reversed)
			Expect(ok).To(BeTrue())

			Expect(first.name).To(Equal(second.name))
			Expect(first.name).To(Equal(artifactName))
		})

		It("should pick nothing from an empty set", func() {
			_, ok := pickStalledChild(nil)
			Expect(ok).To(BeFalse())
		})

		It("should name the child and its reason in the message", func() {
			child := stalledChild{
				name:    artifactName,
				reason:  konfidence.StalledReasonManifestMissing,
				message: "no konfidence manifest in artifact",
			}

			Expect(stalledChildMessage(child, 1)).To(SatisfyAll(
				ContainSubstring(artifactName),
				ContainSubstring(konfidence.StalledReasonManifestMissing),
			))
			Expect(stalledChildMessage(child, 3)).To(ContainSubstring("3 artifact deployments are stalled"))
		})
	})
})
