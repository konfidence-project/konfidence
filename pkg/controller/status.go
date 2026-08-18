package utils

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PatchStatusIfChanged patches obj's status when currentStatus differs from originalStatus and
// preserves reconcileErr if both reconciliation and status patching fail.
func PatchStatusIfChanged(
	ctx context.Context,
	clusterClient client.Client,
	obj client.Object,
	original client.Object,
	currentStatus any,
	originalStatus any,
	patchErrorMessage string,
	reconcileErr error,
	reconcileErrorMessage string,
) error {
	if reflect.DeepEqual(currentStatus, originalStatus) {
		return reconcileErr
	}

	patchErr := clusterClient.Status().Patch(ctx, obj, client.MergeFrom(original))
	if patchErr == nil {
		return reconcileErr
	}

	// If the object was deleted between reconciliation and status patching, ignore the patch error.
	if apierrors.IsNotFound(patchErr) {
		return reconcileErr
	}

	// If the patch failed due to conflict (resourceVersion mismatch), another reconciliation
	// will be triggered, so we can safely return the reconcile error without the patch error.
	if apierrors.IsConflict(patchErr) {
		return reconcileErr
	}

	if reconcileErr == nil {
		return fmt.Errorf("%s: %w", patchErrorMessage, patchErr)
	}

	reconcileErr = fmt.Errorf("%s: %w", reconcileErrorMessage, reconcileErr)
	patchErr = fmt.Errorf("%s: %w", patchErrorMessage, patchErr)
	return errors.Join(reconcileErr, patchErr)
}
