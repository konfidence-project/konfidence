/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// StageSyncReconciler watches a StageSync object on the remote client (GCP) and creates/updates/deletes a corresponding Stage object on the local cluster (LCP).
type StageSyncReconciler struct {
	// LocalClient is the client accessing the LCP
	LocalClient client.Client
	// RemoteClient is the client accessing the GCP
	RemoteClient client.Client
	RemoteCache  cache.Cache
	Scheme       *runtime.Scheme
}

// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=stagesyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=stagesyncs/status,verbs=get;update;patch

func (r *StageSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconcile stageSync started...")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StageSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mapStageSyncToRequests := func(ctx context.Context, obj *global.StageSync) []reconcile.Request {
		return r.getNamespacedNameReconcileRequest(ctx, obj)
	}

	b := ctrl.NewControllerManagedBy(mgr).
		WatchesRawSource(
			source.Kind(
				r.RemoteCache,
				&global.StageSync{},
				handler.TypedEnqueueRequestsFromMapFunc[*global.StageSync, reconcile.Request](mapStageSyncToRequests),
			),
		)
	return b.
		Named("sync").
		Complete(r)
}

func (r *StageSyncReconciler) getNamespacedNameReconcileRequest(_ context.Context, object client.Object) []reconcile.Request {
	return []reconcile.Request{{NamespacedName: types.NamespacedName{
		Name:      object.GetName(),
		Namespace: object.GetNamespace(),
	}}}
}
