package controller

import (
	"fmt"
	"sort"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// stalledReasonNotStalled is the reason carried by Stalled=False. A reason is required on
// every condition, and naming the healthy case explicitly keeps "evaluated, nothing
// blocking" distinguishable in the status from a condition that was never written.
const stalledReasonNotStalled = "NotStalled"

// stalledChild is an ArtifactDeployment that reported Stalled=True, captured so the parent
// can name one of them.
type stalledChild struct {
	name    string
	reason  string
	message string
}

// setStalled records that the vector deployment cannot progress without manual action.
//
// Ready is forced to False in the same call rather than left to the caller's control flow.
// Stalled=True alongside Ready=True is a contradiction a reader could act on wrongly, and
// writing both here means a later edit that adds an early return cannot break the
// invariant by forgetting one of them.
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

// clearStalled records that this reconcile observed no blocking cause.
//
// Called at the top of every reconcile so that Stalled is always present on an object the
// controller has seen. kstatus-style tooling reads an absent Stalled as "not stalled", so
// leaving it off the healthy path would make "nothing is blocking" indistinguishable from
// "never evaluated"; writing it always reserves absence for the latter. meta.SetStatusCondition
// leaves an unchanged condition untouched, so repeating this on every reconcile does not
// churn lastTransitionTime or produce a status patch.
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

// pickStalledChild chooses which stalled child the parent reports when several are blocked
// at once.
//
// Only one can be named in the parent's message, and the choice has to be stable: reporting
// whichever child the informer delivered last makes the parent's message flap as unrelated
// children reconcile. Lowest name wins, arbitrary but deterministic.
func pickStalledChild(children []stalledChild) (stalledChild, bool) {
	if len(children) == 0 {
		return stalledChild{}, false
	}

	sort.Slice(children, func(i, j int) bool { return children[i].name < children[j].name })

	return children[0], true
}

// stalledChildMessage describes a blocked child on the parent, naming the child and its own
// reason so a reader does not have to go find the ArtifactDeployment to learn what to fix.
func stalledChildMessage(child stalledChild, total int) string {
	message := fmt.Sprintf("ArtifactDeployment %s is stalled (%s): %s", child.name, child.reason, child.message)
	if total > 1 {
		message = fmt.Sprintf("%s; %d artifact deployments are stalled", message, total)
	}

	return message
}
