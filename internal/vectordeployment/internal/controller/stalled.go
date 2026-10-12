package controller

import (
	"errors"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// stallOnPermanentError sets Stalled when err is a PermanentError and reports whether it did. Other errors may be
// transient and are left to the retry.
func stallOnPermanentError(vectorDeployment *konfidence.VectorDeployment, err error) bool {
	var permanentErr *PermanentError
	if !errors.As(err, &permanentErr) {
		return false
	}
	setStalledConditionTrue(vectorDeployment, permanentErr.Reason, permanentErr.Error())
	return true
}

func setStalledConditionTrue(vectorDeployment *konfidence.VectorDeployment, reason, message string) {
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
		Type:               konfidence.StalledCondition,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: vectorDeployment.Generation,
	})
}

func setStalledConditionFalse(vectorDeployment *konfidence.VectorDeployment) {
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
		Type:               konfidence.StalledCondition,
		Status:             metav1.ConditionFalse,
		Reason:             konfidence.StalledReasonNotStalled,
		Message:            "No blocking condition detected",
		ObservedGeneration: vectorDeployment.Generation,
	})
}

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

// determineStalledArtifactDeployment returns the stalled ArtifactDeployment with the lowest name, so the choice is stable.
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
	message := fmt.Sprintf("ArtifactDeployment %s is stalled", artifactDeployment.Name)
	if condition := meta.FindStatusCondition(artifactDeployment.Status.Conditions, konfidence.StalledCondition); condition != nil {
		message = fmt.Sprintf("%s (%s): %s", message, condition.Reason, condition.Message)
	}
	if total > 1 {
		message = fmt.Sprintf("%s; %d artifact deployments are stalled", message, total)
	}

	return message
}
