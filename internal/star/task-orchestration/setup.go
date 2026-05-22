package taskorchestration

import (
	"github.com/go-logr/logr"
	"github.com/konfidence-project/konfidence/internal/star/task-orchestration/internal/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const OperatorFlagName = "TaskOrchestration"

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
