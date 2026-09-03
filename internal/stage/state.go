package stage

import (
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StageVersionState is the derived rollout state of a StageVersion.
type StageVersionState string

const (
	// StageVersionStatePending means the stage version was created but its vector deployment has not started yet.
	StageVersionStatePending StageVersionState = "PendingDeployment"
	// StageVersionStateDeploying means the vector deployment is in progress.
	StageVersionStateDeploying StageVersionState = "DeployingVector"
	// StageVersionStateMigrating means the migration tasks are running.
	StageVersionStateMigrating StageVersionState = "MigratingVector"
	// StageVersionStateActivating means the activation tasks are running. This state is transient:
	// the activation and ready conditions are currently set in the same reconcile pass.
	StageVersionStateActivating StageVersionState = "ActivatingVector"
	// StageVersionStateReady means the stage version is fully rolled out.
	StageVersionStateReady StageVersionState = "Ready"
	// StageVersionStateFailed is reserved for failure detection and is not emitted yet,
	// because no failure conditions are set on StageVersions today.
	StageVersionStateFailed StageVersionState = "Failed"
)

// StateFromConditions derives the rollout state of a StageVersion from its status conditions.
// The conditions are set in a linear chain by the stage version controller, so the most advanced
// condition that is true wins.
func StateFromConditions(conditions []metav1.Condition) StageVersionState {
	switch {
	case meta.IsStatusConditionTrue(conditions, konfidence.StageVersionReady):
		return StageVersionStateReady
	case meta.IsStatusConditionTrue(conditions, konfidence.VectorActivationCreatedCondition):
		return StageVersionStateActivating
	case meta.IsStatusConditionTrue(conditions, konfidence.VectorMigratedCondition),
		meta.IsStatusConditionTrue(conditions, konfidence.VectorMigrationCreatedCondition):
		return StageVersionStateMigrating
	case meta.IsStatusConditionTrue(conditions, konfidence.VectorDeploymentCreatedCondition):
		return StageVersionStateDeploying
	default:
		return StageVersionStatePending
	}
}
