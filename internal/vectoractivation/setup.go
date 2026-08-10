package vectoractivation

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/konfidence-project/konfidence/internal/vectoractivation/internal/controller"
	"github.com/konfidence-project/konfidence/pkg/operator"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const OperatorFlagName = "VectorActivation"

// Domain wires the vector activation controllers into the operator's --controllers flag.
func Domain() operator.Domain {
	return operator.Domain{
		Name:        OperatorFlagName,
		Controllers: "VectorActivation",
		Setup: func(_ context.Context, deps operator.Deps) error {
			return SetupControllers(deps.Mgr, deps.Logger)
		},
	}
}

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
