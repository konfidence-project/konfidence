package controller

import (
	"fmt"
	"sort"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const stalledReasonNotStalled = "NotStalled"

// stalledChild is an ArtifactDeployment reporting Stalled=True.
type stalledChild struct {
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
		Reason:             reason,
		Message:            message,
		ObservedGeneration: vectorDeployment.Generation,
	})
}

// clearStalled records that this reconcile found nothing blocking. Called on every pass so
// absence of the condition means only that the object was never reconciled.
func clearStalled(vectorDeployment *konfidence.VectorDeployment) {
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
		Type:               konfidence.StalledCondition,
		Status:             metav1.ConditionFalse,
		Reason:             stalledReasonNotStalled,
		Message:            "No blocking condition detected",
		ObservedGeneration: vectorDeployment.Generation,
	})
}

// collectStalledChild returns the child's stall details if it reports Stalled=True.
func collectStalledChild(artifactDeployment *konfidence.ArtifactDeployment) (stalledChild, bool) {
	condition := meta.FindStatusCondition(artifactDeployment.Status.Conditions, konfidence.StalledCondition)
	if condition == nil || condition.Status != metav1.ConditionTrue {
		return stalledChild{}, false
	}

	return stalledChild{
		name:    artifactDeployment.Name,
		reason:  condition.Reason,
		message: condition.Message,
	}, true
}

// pickStalledChild chooses which child the parent names. Lowest name wins: arbitrary, but
// stable, so the message does not flap as children reconcile in informer order.
func pickStalledChild(children []stalledChild) (stalledChild, bool) {
	if len(children) == 0 {
		return stalledChild{}, false
	}

	sort.Slice(children, func(i, j int) bool { return children[i].name < children[j].name })

	return children[0], true
}

func stalledChildMessage(child stalledChild, total int) string {
	message := fmt.Sprintf("ArtifactDeployment %s is stalled (%s): %s", child.name, child.reason, child.message)
	if total > 1 {
		message = fmt.Sprintf("%s; %d artifact deployments are stalled", message, total)
	}

	return message
}
