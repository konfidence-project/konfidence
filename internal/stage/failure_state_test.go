package stage

import (
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExplicitStageVersionFailure(t *testing.T) {
	conditions := []metav1.Condition{
		{Type: konfidence.StageVersionReady, Status: metav1.ConditionTrue},
		{Type: konfidence.StageVersionFailedCondition, Status: metav1.ConditionTrue},
	}
	if got := StateFromConditions(conditions); got != StageVersionStateFailed {
		t.Fatalf("failure must take precedence, got %s", got)
	}
	conditions[1].Status = metav1.ConditionFalse
	if got := StateFromConditions(conditions); got != StageVersionStateReady {
		t.Fatalf("inactive failure changed Ready, got %s", got)
	}
	conditions = []metav1.Condition{{Type: konfidence.StageVersionReady, Status: metav1.ConditionFalse}}
	if got := StateFromConditions(conditions); got != StageVersionStatePending {
		t.Fatalf("ordinary not-ready became failure, got %s", got)
	}
}
