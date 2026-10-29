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
