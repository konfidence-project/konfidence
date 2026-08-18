package stage

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/konfidence-project/konfidence/internal/stage/internal/controller"
	"github.com/konfidence-project/konfidence/internal/stage/internal/gc"
	"github.com/konfidence-project/konfidence/pkg/operator"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const OperatorFlagName = "Stage"

// Domain wires the stage controllers into the operator's --controllers flag.
func Domain() operator.Domain {
	return operator.Domain{
		Name:        OperatorFlagName,
		Controllers: "Stage, StageVersion, StageVersionUsage, stageVersion garbage collector",
		Setup: func(ctx context.Context, deps operator.Deps) error {
			if err := SetupControllers(deps.Mgr, deps.Logger); err != nil {
				return err
			}
			garbageCollector := NewGarbageCollector(deps.Mgr)
			deps.Logger.Info("starting stageVersion garbage collector")
			go func() {
				if err := garbageCollector.Start(ctx); err != nil {
					deps.Logger.Error(err, "stageVersion garbage collector failed")
					deps.Shutdown()
				}
			}()
			return nil
		},
	}
}

func SetupControllers(mgr manager.Manager, logger logr.Logger) (err error) {
	if err := (&controller.StageReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorder(controller.StageControllerName),
	}).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to create controller", "controller", "Stage")
		return err
	}

	if err := (&controller.StageVersionReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorder(controller.StageVersionControllerName),
	}).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to create controller", "controller", "StageVersion")
		return err
	}

	if err := (&controller.StageVersionUsageReconciler{
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
