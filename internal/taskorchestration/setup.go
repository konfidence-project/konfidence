package taskorchestration

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/konfidence-project/konfidence/internal/taskorchestration/internal/controller"
	"github.com/konfidence-project/konfidence/pkg/operator"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const OperatorFlagName = "TaskOrchestration"

// Domain wires the task orchestration controllers into the operator's --controllers flag.
func Domain() operator.Domain {
	return operator.Domain{
		Name:        OperatorFlagName,
		Controllers: "TaskOrchestration",
		Setup: func(_ context.Context, deps operator.Deps) error {
			return SetupControllers(deps.Mgr, deps.Logger)
		},
	}
}

func SetupControllers(mgr manager.Manager, logger logr.Logger) error {
	if err := (&controller.TaskOrchestrationReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorder(controller.TaskOrchestrationControllerName),
	}).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to create controller", "controller", "TaskOrchestration")
		return err
	}
	return nil
}
