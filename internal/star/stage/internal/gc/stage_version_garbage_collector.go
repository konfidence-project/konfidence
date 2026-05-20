package gc

import (
	"context"
	"fmt"
	"time"

	landscape "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const StageVersionGarbageCollectorName = "stage-version-garbage-collector"

type StageVersionGarbageCollector struct {
	client.Client
	Interval time.Duration
	Recorder events.EventRecorder
}

func (gc *StageVersionGarbageCollector) Start(ctx context.Context) error {
	log := logf.FromContext(ctx)
	ticker := time.NewTicker(gc.Interval)
	defer func() {
		log.Info("Stopping stageVersion garbage collector")
		ticker.Stop()
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			log.Info("Executing stageVersion garbage collector...")
			// TODO for now we assume all stageVersions and stageVersionUsages are in the same namespace
			// get all stageVersions
			stageVersions := &landscape.StageVersionList{}
			if err := gc.List(ctx, stageVersions); err != nil {
				log.Error(err, "unable to list stageVersions")
			}

			// get all stageVersionUsages
			stageVersionUsages := &landscape.StageVersionUsageList{}
			if err := gc.List(ctx, stageVersionUsages); err != nil {
				log.Error(err, "unable to list stageVersionUsages")
			}

			// check if stageVersions are still referenced by any stageVersionUsages
			referencedStageVersions := map[string]bool{}
			for _, stageVersionUsage := range stageVersionUsages.Items {
				for _, stageVersionName := range stageVersionUsage.Status.ResolvedStageVersions {
					referencedStageVersions[stageVersionName] = true
				}
			}

			// delete unreferenced stageVersions
			for _, stageVersion := range stageVersions.Items {
				if !referencedStageVersions[stageVersion.Name] {
					log.Info("Deleting stage version", "name", stageVersion.Name)

					// read in the referenced stage to write a deletion event to it
					stage := &landscape.Stage{}
					if err := gc.Get(
						ctx,
						types.NamespacedName{Namespace: stageVersion.Namespace, Name: stageVersion.Spec.StageRef.Name},
						stage,
					); err != nil {
						log.Error(err, "unable to read referenced Stage", "stageVersion", stageVersion.Name, "stage", stageVersion.Spec.StageRef.Name)
						continue
					}
					if err := gc.Delete(ctx, &stageVersion); err != nil {
						// we continue trying to delete other stageVersions and only log the error here
						log.Error(err, "unable to delete stage version", "name", stageVersion.Name)
					}
					gc.Recorder.Eventf(stage, nil, corev1.EventTypeNormal, "StageVersionDeleted", "StageVersionDeleted", fmt.Sprintf("StageVersion %s has been deleted by the garbage collector", stageVersion.Name))
				}
			}

			log.Info("StageVersion garbage collector finished")
		}
	}
}
