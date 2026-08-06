package controller

import (
	"context"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// mapVectorTemplateToConfigs enqueues every config sourcing from the changed template.
func (r *VectorPromotionConfigReconciler) mapVectorTemplateToConfigs(ctx context.Context, obj client.Object) []reconcile.Request {
	return r.configRequests(ctx, obj.GetNamespace(), client.MatchingFields{configSourceTemplateField: obj.GetName()})
}

// mapStageToConfigs enqueues configs referencing the changed stage as source
// or target. The stage lives in a landscape namespace; its Landscape CR
// (indexed by status.namespace) points back to the project namespace where
// the configs live.
func (r *VectorPromotionConfigReconciler) mapStageToConfigs(ctx context.Context, obj client.Object) []reconcile.Request {
	log := logf.FromContext(ctx)

	landscapes := &konfidence.LandscapeList{}
	if err := r.List(ctx, landscapes, client.MatchingFields{landscapeNamespaceField: obj.GetNamespace()}); err != nil {
		log.Error(err, "failed to resolve landscape for stage namespace", "namespace", obj.GetNamespace())
		return nil
	}

	var requests []reconcile.Request
	for i := range landscapes.Items {
		landscape := &landscapes.Items[i]
		key := landscape.Name + "/" + obj.GetName()
		requests = append(requests,
			r.configRequests(ctx, landscape.Namespace, client.MatchingFields{configSourceStageField: key})...)
		requests = append(requests,
			r.configRequests(ctx, landscape.Namespace, client.MatchingFields{configTargetStageField: key})...)
	}
	return dedupeRequests(requests)
}

// mapLandscapeToConfigs enqueues configs referencing the changed landscape.
// Unindexed list plus filter: configs and the landscape share the project
// namespace, which holds few configs.
func (r *VectorPromotionConfigReconciler) mapLandscapeToConfigs(ctx context.Context, obj client.Object) []reconcile.Request {
	log := logf.FromContext(ctx)

	configs := &konfidence.VectorPromotionConfigList{}
	if err := r.List(ctx, configs, client.InNamespace(obj.GetNamespace())); err != nil {
		log.Error(err, "failed to list configs for landscape", "landscape", obj.GetName())
		return nil
	}
	var requests []reconcile.Request
	for i := range configs.Items {
		config := &configs.Items[i]
		if config.Spec.Source.Landscape == obj.GetName() || config.Spec.Target.Landscape == obj.GetName() {
			requests = append(requests, requestFor(config))
		}
	}
	return requests
}

func (r *VectorPromotionConfigReconciler) configRequests(ctx context.Context, namespace string, matching client.MatchingFields) []reconcile.Request {
	log := logf.FromContext(ctx)

	configs := &konfidence.VectorPromotionConfigList{}
	if err := r.List(ctx, configs, client.InNamespace(namespace), matching); err != nil {
		log.Error(err, "failed to list configs for watch mapping", "namespace", namespace)
		return nil
	}
	requests := make([]reconcile.Request, 0, len(configs.Items))
	for i := range configs.Items {
		requests = append(requests, requestFor(&configs.Items[i]))
	}
	return requests
}

func requestFor(config *konfidence.VectorPromotionConfig) reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{Namespace: config.Namespace, Name: config.Name}}
}

func dedupeRequests(requests []reconcile.Request) []reconcile.Request {
	seen := make(map[reconcile.Request]struct{}, len(requests))
	deduped := requests[:0]
	for _, request := range requests {
		if _, ok := seen[request]; ok {
			continue
		}
		seen[request] = struct{}{}
		deduped = append(deduped, request)
	}
	return deduped
}

func latestVectorChanged() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldTemplate, okOld := e.ObjectOld.(*konfidence.VectorTemplate)
			newTemplate, okNew := e.ObjectNew.(*konfidence.VectorTemplate)
			return okOld && okNew && oldTemplate.Status.LatestVector != newTemplate.Status.LatestVector
		},
		DeleteFunc:  func(e event.DeleteEvent) bool { return true },
		GenericFunc: func(e event.GenericEvent) bool { return false },
	}
}

func stageVectorChanged() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldStage, okOld := e.ObjectOld.(*konfidence.Stage)
			newStage, okNew := e.ObjectNew.(*konfidence.Stage)
			return okOld && okNew && oldStage.Spec.Vector != newStage.Spec.Vector
		},
		DeleteFunc:  func(e event.DeleteEvent) bool { return true },
		GenericFunc: func(e event.GenericEvent) bool { return false },
	}
}

func landscapeNamespaceChanged() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldLandscape, okOld := e.ObjectOld.(*konfidence.Landscape)
			newLandscape, okNew := e.ObjectNew.(*konfidence.Landscape)
			return okOld && okNew && oldLandscape.Status.Namespace != newLandscape.Status.Namespace
		},
		DeleteFunc:  func(e event.DeleteEvent) bool { return true },
		GenericFunc: func(e event.GenericEvent) bool { return false },
	}
}
