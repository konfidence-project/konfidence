package stage_test

import (
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/stage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func conditions(types ...string) []metav1.Condition {
	result := make([]metav1.Condition, 0, len(types))
	for _, t := range types {
		result = append(result, metav1.Condition{Type: t, Status: metav1.ConditionTrue, Reason: "Test"})
	}
	return result
}

func TestStateFromConditions(t *testing.T) {
	tests := []struct {
		name       string
		conditions []metav1.Condition
		expected   stage.StageVersionState
	}{
		{
			name:       "no conditions",
			conditions: nil,
			expected:   stage.StageVersionStatePending,
		},
		{
			name:       "vector deployment created",
			conditions: conditions(konfidence.VectorDeploymentCreatedCondition),
			expected:   stage.StageVersionStateDeploying,
		},
		{
			name: "vector migration created",
			conditions: conditions(
				konfidence.VectorDeploymentCreatedCondition,
				konfidence.VectorMigrationCreatedCondition,
			),
			expected: stage.StageVersionStateMigrating,
		},
		{
			name: "vector migrated",
			conditions: conditions(
				konfidence.VectorDeploymentCreatedCondition,
				konfidence.VectorMigrationCreatedCondition,
				konfidence.VectorMigratedCondition,
			),
			expected: stage.StageVersionStateMigrating,
		},
		{
			name: "vector activation created",
			conditions: conditions(
				konfidence.VectorDeploymentCreatedCondition,
				konfidence.VectorMigrationCreatedCondition,
				konfidence.VectorMigratedCondition,
				konfidence.VectorActivationCreatedCondition,
			),
			expected: stage.StageVersionStateActivating,
		},
		{
			name: "full chain",
			conditions: conditions(
				konfidence.VectorDeploymentCreatedCondition,
				konfidence.VectorMigrationCreatedCondition,
				konfidence.VectorMigratedCondition,
				konfidence.VectorActivationCreatedCondition,
				konfidence.StageVersionReady,
			),
			expected: stage.StageVersionStateReady,
		},
		{
			name: "false conditions are ignored",
			conditions: []metav1.Condition{
				{Type: konfidence.VectorDeploymentCreatedCondition, Status: metav1.ConditionTrue, Reason: "Test"},
				{Type: konfidence.VectorMigrationCreatedCondition, Status: metav1.ConditionFalse, Reason: "Test"},
				{Type: konfidence.StageVersionReady, Status: metav1.ConditionFalse, Reason: "Test"},
			},
			expected: stage.StageVersionStateDeploying,
		},
		{
			name: "unknown conditions are ignored",
			conditions: []metav1.Condition{
				{Type: konfidence.StageVersionReady, Status: metav1.ConditionUnknown, Reason: "Test"},
			},
			expected: stage.StageVersionStatePending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if state := stage.StateFromConditions(tt.conditions); state != tt.expected {
				t.Errorf("expected state %q, got %q", tt.expected, state)
			}
		})
	}
}
