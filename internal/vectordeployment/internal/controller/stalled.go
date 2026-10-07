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
		Message:            message,
		ObservedGeneration: vectorDeployment.Generation,
	})
}

// clearStalledCondition records that this reconcile found nothing blocking.
func clearStalledCondition(vectorDeployment *konfidence.VectorDeployment) {
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
		Type:               konfidence.StalledCondition,
		Status:             metav1.ConditionFalse,
		Reason:             konfidence.StalledReasonNotStalled,
		Message:            "No blocking condition detected",
		ObservedGeneration: vectorDeployment.Generation,
	})

	// setStalled is the only writer of Ready=False, so without this its Ready=False would outlive the stall.
	ready := meta.FindStatusCondition(vectorDeployment.Status.Conditions, konfidence.VectorReadyCondition)
	if ready != nil && ready.Reason == konfidence.VectorReadyReasonStalled {
		meta.RemoveStatusCondition(&vectorDeployment.Status.Conditions, konfidence.VectorReadyCondition)
	}
}

// reportStalledArtifactDeployments sets or clears Stalled from this reconcile's ArtifactDeployments. Deciding once,
// rather than clearing up front and setting later, keeps lastTransitionTime from resetting on every reconcile.
func reportStalledArtifactDeployments(vectorDeployment *konfidence.VectorDeployment, artifactDeployments []*konfidence.ArtifactDeployment) {
	var stalled []stalledArtifactDeployment
	for _, artifactDeployment := range artifactDeployments {
		if stalledArtifactDeployment, ok := collectStalledArtifactDeployment(artifactDeployment); ok {
			stalled = append(stalled, stalledArtifactDeployment)
		}
	}

	picked := pickStalledArtifactDeployment(stalled)
	if picked == nil {
		clearStalledCondition(vectorDeployment)
		return
	}

	setStalled(vectorDeployment, konfidence.VectorDeploymentStalledReasonArtifactDeploymentStalled,
		stalledArtifactDeploymentMessage(*picked, len(stalled)))
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

// pickStalledArtifactDeployment picks the first ArtifactDeployment from the sorted list so selection is stable.
// This is important for the message to remain consistent across reconciliations.
// It also returns true or false to indicate whether a deployment was picked.
func pickStalledArtifactDeployment(artifactDeployments []stalledArtifactDeployment) *stalledArtifactDeployment {
	if len(artifactDeployments) == 0 {
		return nil
	}

	sort.Slice(artifactDeployments, func(i, j int) bool { return artifactDeployments[i].name < artifactDeployments[j].name })

	return &artifactDeployments[0]
}

func stalledArtifactDeploymentMessage(artifactDeployment stalledArtifactDeployment, total int) string {
	message := fmt.Sprintf("ArtifactDeployment %s is stalled (%s): %s", artifactDeployment.name, artifactDeployment.reason, artifactDeployment.message)
	if total > 1 {
		message = fmt.Sprintf("%s; %d artifact deployments are stalled", message, total)
	}

	return message
}
