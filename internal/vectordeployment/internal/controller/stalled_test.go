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
		It("should write Stalled=False", func() {
			vectorDeployment := newVectorDeployment(3)

			clearStalledCondition(vectorDeployment)

			condition := stalledCondition(vectorDeployment)
			Expect(condition).ToNot(BeNil())
			Expect(condition.Status).To(Equal(metav1.ConditionFalse))
			Expect(condition.ObservedGeneration).To(Equal(int64(3)))
		})

		It("should keep a single entry across a reason transition", func() {
			vectorDeployment := newVectorDeployment(1)

			setStalledCondition(vectorDeployment, konfidence.VectorDeploymentStalledReasonArtifactDeploymentNamingCollision, "collision")
			setStalledCondition(vectorDeployment, konfidence.VectorDeploymentStalledReasonArtifactDeploymentStalled, "artifact deployment stalled")

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

			setStalledCondition(vectorDeployment, konfidence.VectorDeploymentStalledReasonArtifactDeploymentStalled, "artifact deployment stalled")
			clearStalledCondition(vectorDeployment)

			Expect(stalledCondition(vectorDeployment).Status).To(Equal(metav1.ConditionFalse))
		})
	})

	Context("aggregating stalled ArtifactDeployments", func() {
		// The named ArtifactDeployment must not depend on observation order, or the message flaps.
		It("should pick the same ArtifactDeployment regardless of input order", func() {
			forward := []*konfidence.ArtifactDeployment{
				{ObjectMeta: metav1.ObjectMeta{Name: artifactName}},
				{ObjectMeta: metav1.ObjectMeta{Name: "artifact-b"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "artifact-c"}},
			}
			reversed := []*konfidence.ArtifactDeployment{forward[2], forward[1], forward[0]}

			first := determineStalledArtifactDeployment(forward)
			Expect(first).ToNot(BeNil())
			second := determineStalledArtifactDeployment(reversed)
			Expect(second).ToNot(BeNil())

			Expect(first.Name).To(Equal(second.Name))
			Expect(first.Name).To(Equal(artifactName))
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

			reconcileStalledStatusWithDeployments(vectorDeployment, []*konfidence.ArtifactDeployment{stalledArtifactDeployment})
			since := metav1.NewTime(time.Now().Add(-time.Hour))
			stalledCondition(vectorDeployment).LastTransitionTime = since

			reconcileStalledStatusWithDeployments(vectorDeployment, []*konfidence.ArtifactDeployment{stalledArtifactDeployment})

			Expect(stalledCondition(vectorDeployment).LastTransitionTime).To(Equal(since))
		})

		It("should clear Stalled when no ArtifactDeployment is stalled", func() {
			vectorDeployment := newVectorDeployment(1)
			setStalledCondition(vectorDeployment, konfidence.VectorDeploymentStalledReasonArtifactDeploymentStalled, "artifact deployment stalled")

			reconcileStalledStatusWithDeployments(vectorDeployment, []*konfidence.ArtifactDeployment{{}})

			Expect(stalledCondition(vectorDeployment).Status).To(Equal(metav1.ConditionFalse))
		})

		It("should pick nothing from an empty set", func() {
			Expect(determineStalledArtifactDeployment(nil)).To(BeNil())
		})

		It("should name the ArtifactDeployment and its reason in the message", func() {
			artifactDeployment := &konfidence.ArtifactDeployment{}
			artifactDeployment.Name = artifactName
			artifactDeployment.Status.Conditions = []metav1.Condition{{
				Type:    konfidence.StalledCondition,
				Status:  metav1.ConditionTrue,
				Reason:  konfidence.ArtifactDeploymentStalledReasonManifestMissing,
				Message: "no konfidence manifest in artifact",
			}}

			Expect(stalledArtifactDeploymentMessage(artifactDeployment, 1)).To(SatisfyAll(
				ContainSubstring(artifactName),
				ContainSubstring(konfidence.ArtifactDeploymentStalledReasonManifestMissing),
			))
			Expect(stalledArtifactDeploymentMessage(artifactDeployment, 3)).To(ContainSubstring("3 artifact deployments are stalled"))
		})
	})
})
