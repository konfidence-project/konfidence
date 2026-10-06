package controller

import (
	"fmt"
	"sort"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// stalledArtifactDeployment is an ArtifactDeployment reporting Stalled=True.
type stalledArtifactDeployment struct {
	name    string
	reason  string
	message string
}

// setStalled marks the vector deployment blocked. Ready is forced False here rather than
// left to the caller, so no return path can leave the two contradicting each other.
func setStalled(vectorDeployment *konfidence.VectorDeployment, reason, message string) {
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
		Type:               konfidence.StalledCondition,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: vectorDeployment.Generation,
	})
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
		Type:               konfidence.VectorReadyCondition,
		Status:             metav1.ConditionFalse,
		Reason:             konfidence.VectorReadyReasonStalled,
		Message:            "Vector Stalled",
		ObservedGeneration: vectorDeployment.Generation,
	})
}

// clearStalled records that this reconcile found nothing blocking. Called on every pass so
// absence of the condition means only that the object was never reconciled.
func clearStalled(vectorDeployment *konfidence.VectorDeployment) {
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
		Type:               konfidence.StalledCondition,
		Status:             metav1.ConditionFalse,
		Reason:             konfidence.StalledReasonNotStalled,
		Message:            "No blocking condition detected",
		ObservedGeneration: vectorDeployment.Generation,
	})
}

// collectStalledArtifactDeployment returns the ArtifactDeployment's stall details if it reports Stalled=True.
func collectStalledArtifactDeployment(artifactDeployment *konfidence.ArtifactDeployment) (stalledArtifactDeployment, bool) {
	condition := meta.FindStatusCondition(artifactDeployment.Status.Conditions, konfidence.StalledCondition)
	if condition == nil || condition.Status != metav1.ConditionTrue {
		return stalledArtifactDeployment{}, false
	}

	return stalledArtifactDeployment{
		name:    artifactDeployment.Name,
		reason:  condition.Reason,
		message: condition.Message,
	}, true
}

// pickStalledArtifactDeployment chooses which ArtifactDeployment the parent names. Lowest name wins: arbitrary, but
// stable, so the message does not flap as ArtifactDeployments reconcile in informer order.
func pickStalledArtifactDeployment(artifactDeployments []stalledArtifactDeployment) (stalledArtifactDeployment, bool) {
	if len(artifactDeployments) == 0 {
		return stalledArtifactDeployment{}, false
	}

	sort.Slice(artifactDeployments, func(i, j int) bool { return artifactDeployments[i].name < artifactDeployments[j].name })

	return artifactDeployments[0], true
}

func stalledArtifactDeploymentMessage(artifactDeployment stalledArtifactDeployment, total int) string {
	message := fmt.Sprintf("ArtifactDeployment %s is stalled (%s): %s", artifactDeployment.name, artifactDeployment.reason, artifactDeployment.message)
	if total > 1 {
		message = fmt.Sprintf("%s; %d artifact deployments are stalled", message, total)
	}

	return message
}
