package vectoractivation

import (
	"github.com/go-logr/logr"
	"github.com/konfidence-project/konfidence/internal/star/vector-activation/internal/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func SetupControllers(mgr manager.Manager, logger logr.Logger) error {
	if err := (&controller.VectorActivationReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorder(controller.ActivationControllerName),
	}).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to create controller", "controller", "VectorActivation")
		return err
	}
	return nil
}
