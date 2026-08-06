package deploymenttarget

import (
	"context"

	"github.com/konfidence-project/konfidence/internal/deploymenttarget/internal/controller"
	"github.com/konfidence-project/konfidence/pkg/operator"
)

const OperatorFlagName = "DeploymentTarget"

// Domain wires DeploymentTarget lifecycle handling into the operator.
func Domain() operator.Domain {
	return operator.Domain{
		Name:        OperatorFlagName,
		Controllers: "DeploymentTarget",
		Setup: func(_ context.Context, deps operator.Deps) error {
			return controller.NewReconciler(deps.Mgr).SetupWithManager(deps.Mgr)
		},
	}
}
