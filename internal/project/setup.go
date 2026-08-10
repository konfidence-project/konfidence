package project

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/konfidence-project/konfidence/internal/project/internal/controller"
	"github.com/konfidence-project/konfidence/pkg/operator"
)

const OperatorFlagName = "Project"

// Domain wires the project controllers into the operator's --controllers flag.
func Domain() operator.Domain {
	return operator.Domain{
		Name:        OperatorFlagName,
		Controllers: "Project",
		Setup: func(_ context.Context, deps operator.Deps) error {
			return SetupControllers(deps.Mgr, Options{})
		},
	}
}

// Options configures the project controllers.
type Options struct{}

// SetupControllers registers all project controllers with the given manager.
func SetupControllers(mgr ctrl.Manager, _ Options) error {
	return controller.NewProjectReconciler(mgr).SetupWithManager(mgr)
}
