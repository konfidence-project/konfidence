// Package deploymenttarget contains optional helpers for implementing the Konfidence
// DeploymentTarget contract. It provides opt-in builder helpers, but leaves
// controller bootstrap and target parameter interpretation to the deployer.
package deploymenttarget

import (
	"context"
	"fmt"

	konfidencev1alpha1 "github.com/konfidence-project/konfidence/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// FindReady lists ready targets for a given ArtifactType in the namespace. There should be at most one
// target of a type per namespace. This method returns an error if multiple ready targets match.
func FindReady(
	ctx context.Context,
	reader client.Reader,
	namespace string,
	artifactType konfidencev1alpha1.ArtifactType,
) (*konfidencev1alpha1.DeploymentTarget, error) {
	targets := &konfidencev1alpha1.DeploymentTargetList{}
	if err := reader.List(ctx, targets, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("list DeploymentTargets in namespace %q: %w", namespace, err)
	}

	var match *konfidencev1alpha1.DeploymentTarget
	for i := range targets.Items {
		if targets.Items[i].ArtifactType() != artifactType || !IsReady(&targets.Items[i]) {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("multiple DeploymentTargets in namespace %q for artifact type %q", namespace, artifactType)
		}
		match = &targets.Items[i]
	}
	if match == nil {
		return nil, apierrors.NewNotFound(schema.GroupResource{Group: konfidencev1alpha1.GroupVersion.Group, Resource: "deploymenttargets"}, string(artifactType))
	}
	return match, nil
}

// IsReady reports whether target's Ready condition is true.
func IsReady(target *konfidencev1alpha1.DeploymentTarget) bool {
	if target == nil {
		return false
	}
	condition := meta.FindStatusCondition(target.Status.Conditions, konfidencev1alpha1.DeploymentTargetReadyCondition)
	return condition != nil && condition.Status == metav1.ConditionTrue && condition.ObservedGeneration == target.Generation
}

// EnqueueArtifactDeploymentsForTarget returns a handler that enqueues the
// ArtifactDeployments of artifactType in the namespace of a changed matching
// DeploymentTarget.
func EnqueueArtifactDeploymentsForTarget(artifactType konfidencev1alpha1.ArtifactType, reader client.Reader) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		requests, err := requestsForTarget(ctx, reader, obj, artifactType)
		if err != nil {
			log.FromContext(ctx).Error(err, "map DeploymentTarget change to ArtifactDeployments", "namespace", obj.GetNamespace(), "artifactType", artifactType)
			return nil
		}
		return requests
	})
}

func requestsForTarget(
	ctx context.Context,
	reader client.Reader,
	obj client.Object,
	artifactType konfidencev1alpha1.ArtifactType,
) ([]reconcile.Request, error) {
	resource, ok := obj.(konfidencev1alpha1.ArtifactTypeProvider)
	if !ok || resource.ArtifactType() != artifactType {
		return nil, nil
	}

	deployments := &konfidencev1alpha1.ArtifactDeploymentList{}
	if err := reader.List(ctx, deployments, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil, err
	}

	requests := make([]reconcile.Request, 0)
	for i := range deployments.Items {
		deployment := &deployments.Items[i]
		if deployment.ArtifactType() == artifactType {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(deployment)})
		}
	}
	return requests, nil
}
