package vectordeployment

import (
	"github.com/go-logr/logr"
	"github.com/konfidence-project/konfidence/internal/star/vector-deployment/internal/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Options configures the vector deployment controllers.
type Options struct {
	// OcmAdapter provides OCM operations (vector descriptor and artifact manifest fetching).
	OcmAdapter controller.VectorOcmPort
}

// SetupControllers registers all vector deployment controllers with the given manager.
func SetupControllers(mgr manager.Manager, logger logr.Logger, opts Options) error {
	if err := (&controller.VectorDeploymentReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Recorder:   mgr.GetEventRecorder(controller.VectorDeploymentControllerName),
		OcmAdapter: opts.OcmAdapter,
	}).SetupWithManager(mgr, "vectordeployment"); err != nil {
		logger.Error(err, "unable to create controller", "controller", "VectorDeployment")
		return err
	}
	return nil
}
