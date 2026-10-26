package deploymenttarget

import (
	"context"

	konfidencev1alpha1 "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var referencedTargetsMarker handler.TypedEventHandler[client.Object, reconcile.Request] = struct {
	handler.TypedEventHandler[client.Object, reconcile.Request]
}{}

// EnqueueReferencedTargets returns a handler that enqueues DeploymentTargets whose
// parameters reference the watched resource and whose artifact type is supported.
//
// Use it with a controller watch, for example:
//
//	deploymenttarget.NewControllerManagedBy(mgr).
//	    Watches(&MyResource{}, deploymenttarget.EnqueueReferencedTargets())
func EnqueueReferencedTargets() handler.TypedEventHandler[client.Object, reconcile.Request] {
	return referencedTargetsMarker
}

func enqueueTargetsReferencing(gvk schema.GroupVersionKind, supported konfidencev1alpha1.ArtifactTypeSet, reader client.Reader) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		requests, err := requestsForReference(ctx, reader, obj, gvk, supported)
		if err != nil {
			log.FromContext(ctx).Error(err, "map referenced resource change to DeploymentTargets", "namespace", obj.GetNamespace(), "kind", gvk.Kind)
			return nil
		}
		return requests
	})
}

func requestsForReference(
	ctx context.Context,
	reader client.Reader,
	obj client.Object,
	gvk schema.GroupVersionKind,
	supported konfidencev1alpha1.ArtifactTypeSet,
) ([]reconcile.Request, error) {
	if obj == nil {
		return nil, nil
	}

	targets := &konfidencev1alpha1.DeploymentTargetList{}
	if err := reader.List(ctx, targets, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil, err
	}

	requests := make([]reconcile.Request, 0)
	for i := range targets.Items {
		target := &targets.Items[i]
		params := target.Spec.Parameters
		if !supported.Contains(target.ArtifactType()) || params == nil {
			continue
		}
		if params.APIGroup == gvk.Group && params.Kind == gvk.Kind && params.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(target)})
		}
	}
	return requests, nil
}
