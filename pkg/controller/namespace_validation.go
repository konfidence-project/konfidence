package utils

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// NamespaceNotReadyReason is the condition reason set when a resource's namespace does not
	// yet carry the required konfidence.cloud/type label.
	NamespaceNotReadyReason = "NamespaceNotReady"

	// NamespaceNotReadyRequeueInterval is how long a reconciler waits before re-checking a
	// namespace that is not yet labeled.
	NamespaceNotReadyRequeueInterval = 10 * time.Second
)

// MustBeInProjectNamespace guards a reconcile: it requires obj to live in a project namespace
// (konfidence.cloud/type=project). It returns a non-nil result the caller must return
// immediately when the namespace is not ready; nil means reconcile continues.
func MustBeInProjectNamespace(
	ctx context.Context,
	c client.Client,
	obj client.Object,
	conditions *[]metav1.Condition,
	conditionType string,
) (*ctrl.Result, error) {
	return guardNamespace(ctx, c, obj, conditions, conditionType, "project")
}

// MustBeInLandscapeNamespace guards a reconcile: it requires obj to live in a landscape namespace
// (konfidence.cloud/type=landscape). It returns a non-nil result the caller must return
// immediately when the namespace is not ready; nil means reconcile continues.
func MustBeInLandscapeNamespace(
	ctx context.Context,
	c client.Client,
	obj client.Object,
	conditions *[]metav1.Condition,
	conditionType string,
) (*ctrl.Result, error) {
	return guardNamespace(ctx, c, obj, conditions, conditionType, "landscape")
}

// guardNamespace requires obj's namespace to carry konfidence.cloud/type=expectedType. When it
// does not, guardNamespace sets conditionType to NamespaceNotReady, patches the status, and
// returns a requeue result; an error reading the namespace backs off normally. It is a no-op
// (nil result) when obj is being deleted or the namespace is valid.
func guardNamespace(
	ctx context.Context,
	c client.Client,
	obj client.Object,
	conditions *[]metav1.Condition,
	conditionType, expectedType string,
) (*ctrl.Result, error) {
	if !obj.GetDeletionTimestamp().IsZero() {
		return nil, nil
	}

	ns := &corev1.Namespace{}
	switch err := c.Get(ctx, types.NamespacedName{Name: obj.GetNamespace()}, ns); {
	case err != nil && !apierrors.IsNotFound(err):
		return &ctrl.Result{}, err
	case err == nil && ns.Labels[ProjectTypeLabel] == expectedType:
		return nil, nil
	}

	message := fmt.Sprintf(
		"resource must be created in a %s namespace: namespace %q is missing required label %s=%s",
		expectedType, obj.GetNamespace(), ProjectTypeLabel, expectedType)

	original := obj.DeepCopyObject().(client.Object)
	result := &ctrl.Result{RequeueAfter: NamespaceNotReadyRequeueInterval}
	if meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               conditionType,
		Status:             metav1.ConditionFalse,
		Reason:             NamespaceNotReadyReason,
		Message:            message,
		ObservedGeneration: obj.GetGeneration(),
	}) {
		if err := c.Status().Patch(ctx, obj, client.MergeFrom(original)); err != nil {
			return result, err
		}
	}
	return result, nil
}
