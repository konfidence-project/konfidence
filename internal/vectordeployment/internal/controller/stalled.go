package controller

import (
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// setStalledConditionTrue marks the vector deployment stalled.
func setStalledConditionTrue(vectorDeployment *konfidence.VectorDeployment, reason, message string) {
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
		Type:               konfidence.StalledCondition,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: vectorDeployment.Generation,
	})
}

// setStalledConditionFalse records that this reconcile found nothing blocking.
func setStalledConditionFalse(vectorDeployment *konfidence.VectorDeployment) {
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
		Type:               konfidence.StalledCondition,
		Status:             metav1.ConditionFalse,
		Reason:             konfidence.StalledReasonNotStalled,
		Message:            "No blocking condition detected",
		ObservedGeneration: vectorDeployment.Generation,
	})
}

// reconcileStalledStatusWithDeployments reports a Stalled condition on the vector deployment if any ArtifactDeployments are stalled.
// In case no ArtifactDeployments are stalled, it clears the Stalled condition if it exists.
func reconcileStalledStatusWithDeployments(
	vectorDeployment *konfidence.VectorDeployment,
	artifactDeployments []*konfidence.ArtifactDeployment) {
	stalledDeployment := determineStalledArtifactDeployment(artifactDeployments)
	if stalledDeployment == nil {
		setStalledConditionFalse(vectorDeployment)
		return
	}

	stalledCount := 0
	for _, artifactDeployment := range artifactDeployments {
		if meta.IsStatusConditionTrue(artifactDeployment.Status.Conditions, konfidence.StalledCondition) {
			stalledCount++
		}
	}

	setStalledConditionTrue(vectorDeployment, konfidence.VectorDeploymentStalledReasonArtifactDeploymentStalled,
		stalledArtifactDeploymentMessage(stalledDeployment, stalledCount))
}

// determineStalledArtifactDeployment picks a stalled ArtifactDeployment in a deterministic way.
// If there are multiple stalled ArtifactDeployments, the same is chosen every time unless the list of
// artifactDeployments itself changed.
func determineStalledArtifactDeployment(artifactDeployments []*konfidence.ArtifactDeployment) *konfidence.ArtifactDeployment {
	var picked *konfidence.ArtifactDeployment
	for _, artifactDeployment := range artifactDeployments {
		if !meta.IsStatusConditionTrue(artifactDeployment.Status.Conditions, konfidence.StalledCondition) {
			continue
		}
		if picked == nil || artifactDeployment.Name < picked.Name {
			picked = artifactDeployment
		}
	}

	return picked
}

func stalledArtifactDeploymentMessage(artifactDeployment *konfidence.ArtifactDeployment, total int) string {
	condition := meta.FindStatusCondition(artifactDeployment.Status.Conditions, konfidence.StalledCondition)
	message := fmt.Sprintf("ArtifactDeployment %s is stalled (%s): %s", artifactDeployment.Name, condition.Reason, condition.Message)
	if total > 1 {
		message = fmt.Sprintf("%s; %d artifact deployments are stalled", message, total)
	}

	return message
}
