package landscape

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/konfidence-project/konfidence/internal/landscape/internal/controller"
	"github.com/konfidence-project/konfidence/pkg/operator"
)

const OperatorFlagName = "Landscape"

// Domain wires the landscape controllers into the operator's --controllers flag.
func Domain() operator.Domain {
	return operator.Domain{
		Name:        OperatorFlagName,
		Controllers: "Landscape",
		Setup: func(_ context.Context, deps operator.Deps) error {
			return SetupControllers(deps.Mgr, Options{})
		},
	}
}

// Options configures the landscape controllers.
type Options struct{}

// SetupControllers registers all landscape controllers with the given manager.
func SetupControllers(mgr ctrl.Manager, _ Options) error {
	return controller.NewLandscapeReconciler(mgr).SetupWithManager(mgr)
}
