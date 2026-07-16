package project

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/konfidence-project/konfidence/internal/project/internal/controller"
)

const OperatorFlagName = "Project"

// Options configures the project controllers.
type Options struct{}

// SetupControllers registers all project controllers with the given manager.
func SetupControllers(mgr ctrl.Manager, _ Options) error {
	return controller.NewProjectReconciler(mgr).SetupWithManager(mgr)
}
