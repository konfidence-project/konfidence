package controller

import "time"

func SetManifestPollIntervalForTest(reconciler *VectorDeploymentReconciler, interval time.Duration) {
	reconciler.manifestPollInterval = interval
}

func InflightJobCountForTest(reconciler *VectorDeploymentReconciler) int {
	if reconciler.jobs == nil {
		return 0
	}

	reconciler.jobs.mu.Lock()
	defer reconciler.jobs.mu.Unlock()

	return len(reconciler.jobs.inflight)
}
