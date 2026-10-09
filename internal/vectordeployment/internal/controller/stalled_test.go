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

	newArtifactDeployment := func(name string, stalled metav1.ConditionStatus) *konfidence.ArtifactDeployment {
		artifactDeployment := &konfidence.ArtifactDeployment{}
		artifactDeployment.Name = name
		artifactDeployment.Status.Conditions = []metav1.Condition{{
			Type:    konfidence.StalledCondition,
			Status:  stalled,
			Reason:  konfidence.ArtifactDeploymentStalledReasonManifestMissing,
			Message: "no konfidence manifest in artifact",
		}}

		return artifactDeployment
	}

	Context("aggregating stalled ArtifactDeployments", func() {
		// The named ArtifactDeployment must not depend on observation order, or the message flaps.
		It("should pick the lowest stalled name regardless of input order, skipping ones that are not stalled", func() {
			forward := []*konfidence.ArtifactDeployment{
				newArtifactDeployment("artifact-0", metav1.ConditionFalse),
				newArtifactDeployment(artifactName, metav1.ConditionTrue),
				newArtifactDeployment("artifact-b", metav1.ConditionTrue),
			}
			reversed := []*konfidence.ArtifactDeployment{forward[2], forward[1], forward[0]}

			Expect(determineStalledArtifactDeployment(forward).Name).To(Equal(artifactName))
			Expect(determineStalledArtifactDeployment(reversed).Name).To(Equal(artifactName))
		})

		It("should name one stalled ArtifactDeployment and count only the stalled ones", func() {
			vectorDeployment := newVectorDeployment(4)

			reconcileStalledStatusWithDeployments(vectorDeployment, []*konfidence.ArtifactDeployment{
				newArtifactDeployment("artifact-b", metav1.ConditionTrue),
				newArtifactDeployment(artifactName, metav1.ConditionTrue),
				newArtifactDeployment("artifact-0", metav1.ConditionFalse),
			})

			condition := stalledCondition(vectorDeployment)
			Expect(condition.Status).To(Equal(metav1.ConditionTrue))
			Expect(condition.Reason).To(Equal(konfidence.VectorDeploymentStalledReasonArtifactDeploymentStalled))
			Expect(condition.Message).To(Equal("ArtifactDeployment artifact-a is stalled (ManifestMissing): " +
				"no konfidence manifest in artifact; 2 artifact deployments are stalled"))
			Expect(condition.ObservedGeneration).To(Equal(int64(4)))
		})

		// Stalled and Ready are independent: a vector can be Ready and Stalled at once.
		It("should leave Ready alone when it reports a stall", func() {
			vectorDeployment := newVectorDeployment(1)
			meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
				Type:   konfidence.VectorReadyCondition,
				Status: metav1.ConditionTrue,
				Reason: konfidence.VectorReadyCondition,
			})

			reconcileStalledStatusWithDeployments(vectorDeployment, []*konfidence.ArtifactDeployment{
				newArtifactDeployment(artifactName, metav1.ConditionTrue),
			})

			Expect(stalledCondition(vectorDeployment).Status).To(Equal(metav1.ConditionTrue))
			Expect(meta.IsStatusConditionTrue(vectorDeployment.Status.Conditions, konfidence.VectorReadyCondition)).To(BeTrue())
		})

		It("should keep lastTransitionTime while the same stall persists across reconciles", func() {
			vectorDeployment := newVectorDeployment(1)
			stalledArtifactDeployment := newArtifactDeployment(artifactName, metav1.ConditionTrue)

			reconcileStalledStatusWithDeployments(vectorDeployment, []*konfidence.ArtifactDeployment{stalledArtifactDeployment})
			since := metav1.NewTime(time.Now().Add(-time.Hour))
			stalledCondition(vectorDeployment).LastTransitionTime = since

			reconcileStalledStatusWithDeployments(vectorDeployment, []*konfidence.ArtifactDeployment{stalledArtifactDeployment})

			Expect(stalledCondition(vectorDeployment).LastTransitionTime).To(Equal(since))
		})

		It("should clear an earlier stall once no ArtifactDeployment is stalled", func() {
			vectorDeployment := newVectorDeployment(1)
			setStalledConditionTrue(vectorDeployment, konfidence.VectorDeploymentStalledReasonArtifactDeploymentStalled, "artifact deployment stalled")

			reconcileStalledStatusWithDeployments(vectorDeployment, []*konfidence.ArtifactDeployment{
				{},
				newArtifactDeployment(artifactName, metav1.ConditionFalse),
			})

			Expect(stalledCondition(vectorDeployment).Status).To(Equal(metav1.ConditionFalse))
			Expect(stalledCondition(vectorDeployment).Reason).To(Equal(konfidence.StalledReasonNotStalled))
		})

		It("should pick nothing from an empty set", func() {
			Expect(determineStalledArtifactDeployment(nil)).To(BeNil())
		})
	})
})
