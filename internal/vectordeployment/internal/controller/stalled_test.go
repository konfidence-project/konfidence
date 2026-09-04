package controller

import (
	"strings"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testArtifactName = "artifact-a"

func vectorDeploymentWithGeneration(generation int64) *konfidence.VectorDeployment {
	vectorDeployment := &konfidence.VectorDeployment{}
	vectorDeployment.Name = "test-vector"
	vectorDeployment.Generation = generation

	return vectorDeployment
}

// A healthy reconcile has to leave Stalled present and False. If the controller only wrote
// the condition on the failure path, "nothing is blocking" would be indistinguishable from
// "never evaluated" for any reader following the kstatus convention.
func TestClearStalledWritesFalseCondition(t *testing.T) {
	vectorDeployment := vectorDeploymentWithGeneration(3)

	clearStalled(vectorDeployment)

	condition := meta.FindStatusCondition(vectorDeployment.Status.Conditions, konfidence.StalledCondition)
	if condition == nil {
		t.Fatal("expected Stalled condition to be written on a healthy reconcile")
	}
	if condition.Status != metav1.ConditionFalse {
		t.Fatalf("Stalled = %q, want %q", condition.Status, metav1.ConditionFalse)
	}
	if condition.ObservedGeneration != 3 {
		t.Fatalf("ObservedGeneration = %d, want 3", condition.ObservedGeneration)
	}
}

// Stalled=True and Ready=True is a contradiction, so setStalled owns both writes.
func TestSetStalledForcesReadyFalse(t *testing.T) {
	vectorDeployment := vectorDeploymentWithGeneration(1)
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
		Type:               konfidence.VectorReadyCondition,
		Status:             metav1.ConditionTrue,
		Reason:             konfidence.VectorReadyCondition,
		ObservedGeneration: 1,
	})

	setStalled(vectorDeployment, konfidence.StalledReasonArtifactDeploymentNamingCollision, "collision")

	if !meta.IsStatusConditionTrue(vectorDeployment.Status.Conditions, konfidence.StalledCondition) {
		t.Fatal("expected Stalled=True")
	}
	if meta.IsStatusConditionTrue(vectorDeployment.Status.Conditions, konfidence.VectorReadyCondition) {
		t.Fatal("Ready must be False whenever Stalled is True")
	}
}

// A reason changing while the status stays True must update in place rather than drop and
// re-add the condition, so consumers keep watching one entry keyed by type.
func TestSetStalledReasonTransitionKeepsSingleEntry(t *testing.T) {
	vectorDeployment := vectorDeploymentWithGeneration(1)

	setStalled(vectorDeployment, konfidence.StalledReasonArtifactDeploymentNamingCollision, "collision")
	setStalled(vectorDeployment, konfidence.StalledReasonChildArtifactDeploymentStalled, "child blocked")

	var count int
	for _, condition := range vectorDeployment.Status.Conditions {
		if condition.Type == konfidence.StalledCondition {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("found %d Stalled conditions, want exactly 1", count)
	}

	condition := meta.FindStatusCondition(vectorDeployment.Status.Conditions, konfidence.StalledCondition)
	if condition.Reason != konfidence.StalledReasonChildArtifactDeploymentStalled {
		t.Fatalf("Reason = %q, want %q", condition.Reason, konfidence.StalledReasonChildArtifactDeploymentStalled)
	}
}

// Once the blocking cause is gone the condition has to flip back, otherwise a resolved
// stall stays reported forever.
func TestClearStalledAfterSetFlipsBackToFalse(t *testing.T) {
	vectorDeployment := vectorDeploymentWithGeneration(1)

	setStalled(vectorDeployment, konfidence.StalledReasonChildArtifactDeploymentStalled, "child blocked")
	clearStalled(vectorDeployment)

	condition := meta.FindStatusCondition(vectorDeployment.Status.Conditions, konfidence.StalledCondition)
	if condition.Status != metav1.ConditionFalse {
		t.Fatalf("Stalled = %q, want %q after the cause cleared", condition.Status, metav1.ConditionFalse)
	}
}

// The stale-generation case: the spec is fixed, generation bumps, and the controller has
// not reconciled yet. Stalled must still read True against the generation it was computed
// from, so a client can tell the condition describes the previous spec rather than treating
// it as a fresh verdict on the new one.
func TestStalledCarriesTheObservedGenerationItWasComputedFrom(t *testing.T) {
	vectorDeployment := vectorDeploymentWithGeneration(7)
	setStalled(vectorDeployment, konfidence.StalledReasonChildArtifactDeploymentStalled, "child blocked")

	// User edits the spec; the controller has not run again.
	vectorDeployment.Generation = 8

	condition := meta.FindStatusCondition(vectorDeployment.Status.Conditions, konfidence.StalledCondition)
	if condition.Status != metav1.ConditionTrue {
		t.Fatalf("Stalled = %q, want %q", condition.Status, metav1.ConditionTrue)
	}
	if condition.ObservedGeneration != 7 {
		t.Fatalf("ObservedGeneration = %d, want 7 (the generation the stall was computed from)", condition.ObservedGeneration)
	}
	if condition.ObservedGeneration >= vectorDeployment.Generation {
		t.Fatal("condition should be detectably stale relative to metadata.generation")
	}
}

// Which child gets named must not depend on the order children were observed, or the
// parent's message flaps as unrelated children reconcile.
func TestPickStalledChildIsDeterministic(t *testing.T) {
	forward := []stalledChild{
		{name: testArtifactName, reason: konfidence.StalledReasonManifestMissing},
		{name: "artifact-b", reason: konfidence.StalledReasonDeploymentResultNotUnique},
		{name: "artifact-c", reason: konfidence.StalledReasonManifestMissing},
	}
	reversed := []stalledChild{forward[2], forward[1], forward[0]}

	first, ok := pickStalledChild(forward)
	if !ok {
		t.Fatal("expected a child to be picked")
	}
	second, ok := pickStalledChild(reversed)
	if !ok {
		t.Fatal("expected a child to be picked")
	}

	if first.name != second.name {
		t.Fatalf("pick depends on input order: %q vs %q", first.name, second.name)
	}
	if first.name != testArtifactName {
		t.Fatalf("picked %q, want the lowest name", first.name)
	}
}

func TestPickStalledChildEmpty(t *testing.T) {
	if _, ok := pickStalledChild(nil); ok {
		t.Fatal("expected no child to be picked from an empty set")
	}
}

// The parent's message has to name the child and its own reason, so an operator reading the
// vector does not have to go hunting for which artifact is blocked and why.
func TestStalledChildMessageNamesChildAndReason(t *testing.T) {
	child := stalledChild{
		name:    testArtifactName,
		reason:  konfidence.StalledReasonManifestMissing,
		message: "no konfidence manifest in artifact",
	}

	single := stalledChildMessage(child, 1)
	if !strings.Contains(single, testArtifactName) || !strings.Contains(single, konfidence.StalledReasonManifestMissing) {
		t.Fatalf("message %q should name the child and its reason", single)
	}

	multiple := stalledChildMessage(child, 3)
	if !strings.Contains(multiple, "3 artifact deployments are stalled") {
		t.Fatalf("message %q should report how many children are stalled", multiple)
	}
}

func TestCollectStalledChild(t *testing.T) {
	tests := []struct {
		name       string
		conditions []metav1.Condition
		wantOK     bool
	}{
		{name: "no conditions", wantOK: false},
		{
			name: "not stalled",
			conditions: []metav1.Condition{{
				Type:   konfidence.StalledCondition,
				Status: metav1.ConditionFalse,
				Reason: stalledReasonNotStalled,
			}},
			wantOK: false,
		},
		{
			name: "stalled",
			conditions: []metav1.Condition{{
				Type:    konfidence.StalledCondition,
				Status:  metav1.ConditionTrue,
				Reason:  konfidence.StalledReasonManifestMissing,
				Message: "no konfidence manifest",
			}},
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifactDeployment := &konfidence.ArtifactDeployment{}
			artifactDeployment.Name = testArtifactName
			artifactDeployment.Status.Conditions = tt.conditions

			child, ok := collectStalledChild(artifactDeployment)
			if ok != tt.wantOK {
				t.Fatalf("collectStalledChild ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && child.name != testArtifactName {
				t.Fatalf("child.name = %q, want %q", child.name, testArtifactName)
			}
		})
	}
}
