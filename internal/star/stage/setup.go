package stage

import (
	"time"

	"github.com/go-logr/logr"
	"github.com/konfidence-project/konfidence/internal/star/stage/internal/gc"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const OperatorFlagName = "Stage"

func SetupControllers(mgr manager.Manager, logger logr.Logger) (err error) {
	if err := (&StageReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorder(StageControllerName),
	}).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to create controller", "controller", "Stage")
		return err
	}

	if err := (&StageVersionReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorder(StageVersionControllerName),
	}).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to create controller", "controller", "StageVersion")
		return err
	}

	if err := (&StageVersionUsageReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to create controller", "controller", "StageVersionUsage")
		return err
	}
	return nil
}

func NewGarbageCollector(mgr manager.Manager) *gc.StageVersionGarbageCollector {
	garbageCollector := &gc.StageVersionGarbageCollector{
		Client:   mgr.GetClient(),
		Interval: 15 * time.Second,
		Recorder: mgr.GetEventRecorder(gc.StageVersionGarbageCollectorName),
	}
	return garbageCollector
}
