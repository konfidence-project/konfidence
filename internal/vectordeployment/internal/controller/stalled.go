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

// clearStalled records that this reconcile found nothing blocking. Called on every pass that gets through the
// ArtifactDeployments, so a healthy vector carries Stalled=False rather than no condition at all.
func clearStalled(vectorDeployment *konfidence.VectorDeployment) {
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

	picked, ok := pickStalledArtifactDeployment(stalled)
	if !ok {
		clearStalled(vectorDeployment)
		return
	}

	setStalled(vectorDeployment, konfidence.VectorDeploymentStalledReasonArtifactDeploymentStalled,
		stalledArtifactDeploymentMessage(picked, len(stalled)))
}

// isArtifactDeploymentReady treats a stalled ArtifactDeployment as not ready, so the vector cannot reach Ready=True
// past one even if a deployer reports both.
func isArtifactDeploymentReady(artifactDeployment *konfidence.ArtifactDeployment) bool {
	return meta.IsStatusConditionTrue(artifactDeployment.Status.Conditions, konfidence.ArtifactDeploymentReadyCondition) &&
		!meta.IsStatusConditionTrue(artifactDeployment.Status.Conditions, konfidence.StalledCondition)
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
