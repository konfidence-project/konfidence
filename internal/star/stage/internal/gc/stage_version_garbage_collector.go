package gc

import (
	"context"
	"time"

	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type StageVersionGarbageCollector struct {
	client.Client
	Interval time.Duration
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
					if err := gc.Delete(ctx, &stageVersion); err != nil {
						// we continue trying to delete other stageVersions and only log the error here
						log.Error(err, "unable to delete stage version", "name", stageVersion.Name)
					}
				}
			}

			log.Info("StageVersion garbage collector finished")
		}
	}
}
