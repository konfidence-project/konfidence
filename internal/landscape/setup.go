package landscape

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/konfidence-project/konfidence/internal/landscape/internal/controller"
)

const OperatorFlagName = "Landscape"

// Options configures the landscape controllers.
type Options struct{}

// SetupControllers registers all landscape controllers with the given manager.
func SetupControllers(mgr ctrl.Manager, _ Options) error {
	return controller.NewLandscapeReconciler(mgr).SetupWithManager(mgr)
}
