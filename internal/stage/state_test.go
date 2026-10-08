package stage_test

import (
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/stage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func conditions(types ...string) []metav1.Condition {
	result := make([]metav1.Condition, 0, len(types))
	for _, t := range types {
		result = append(result, metav1.Condition{Type: t, Status: metav1.ConditionTrue, Reason: "Test"})
	}
	return result
}

var _ = Describe("StateFromConditions", func() {
	DescribeTable("derives the stage version state",
		func(input []metav1.Condition, expected stage.StageVersionState) {
			Expect(stage.StateFromConditions(input)).To(Equal(expected))
		},
		Entry("with no conditions", nil, stage.StageVersionStatePending),
		Entry("when vector deployment is created",
			conditions(konfidence.VectorDeploymentCreatedCondition),
			stage.StageVersionStateDeploying,
		),
		Entry("when vector migration is created",
			conditions(
				konfidence.VectorDeploymentCreatedCondition,
				konfidence.VectorMigrationCreatedCondition,
			),
			stage.StageVersionStateMigrating,
		),
		Entry("when the vector is migrated",
			conditions(
				konfidence.VectorDeploymentCreatedCondition,
				konfidence.VectorMigrationCreatedCondition,
				konfidence.VectorMigratedCondition,
			),
			stage.StageVersionStateMigrating,
		),
		Entry("when vector activation is created",
			conditions(
				konfidence.VectorDeploymentCreatedCondition,
				konfidence.VectorMigrationCreatedCondition,
				konfidence.VectorMigratedCondition,
				konfidence.VectorActivationCreatedCondition,
			),
			stage.StageVersionStateActivating,
		),
		Entry("with the full chain",
			conditions(
				konfidence.VectorDeploymentCreatedCondition,
				konfidence.VectorMigrationCreatedCondition,
				konfidence.VectorMigratedCondition,
				konfidence.VectorActivationCreatedCondition,
				konfidence.StageVersionReady,
			),
			stage.StageVersionStateReady,
		),
		Entry("ignores false conditions",
			[]metav1.Condition{
				{Type: konfidence.VectorDeploymentCreatedCondition, Status: metav1.ConditionTrue, Reason: "Test"},
				{Type: konfidence.VectorMigrationCreatedCondition, Status: metav1.ConditionFalse, Reason: "Test"},
				{Type: konfidence.StageVersionReady, Status: metav1.ConditionFalse, Reason: "Test"},
			},
			stage.StageVersionStateDeploying,
		),
		Entry("ignores unknown conditions",
			[]metav1.Condition{
				{Type: konfidence.StageVersionReady, Status: metav1.ConditionUnknown, Reason: "Test"},
			},
			stage.StageVersionStatePending,
		),
	)
})
