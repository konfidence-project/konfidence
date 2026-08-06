package controller

import (
	"context"
	"fmt"
	"reflect"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
)

// patchStatusWithEvent patches the promotion status if it changed and emits a
// warning event when the patch fails.
func (r *VectorPromotionReconciler) patchStatusWithEvent(ctx context.Context, vectorPromotion, original *konfidence.VectorPromotion) error {
	err := patchPromotionStatus(ctx, r.Client, vectorPromotion, original)
	if err != nil {
		r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeWarning, "StatusPatchFailed", EventActionStatusPatch, err.Error())
	}
	return err
}

// setPromotionCondition writes the Succeeded condition and refreshes the
// derived status.state. `meta.SetStatusCondition` only stamps
// LastTransitionTime when the condition status flips, not on a reason change;
// a reason change (e.g. Running to Superseded, both False) must still register
// as a transition because the TTL clock and the config mirror key on it, so
// the timestamp is forced here.
func setPromotionCondition(
	vectorPromotion *konfidence.VectorPromotion,
	status metav1.ConditionStatus,
	reason, message string,
) {
	previous := meta.FindStatusCondition(vectorPromotion.Status.Conditions, konfidence.ConditionTypeSucceeded)
	reasonOnlyChange := previous != nil && previous.Status == status && previous.Reason != reason
	meta.SetStatusCondition(&vectorPromotion.Status.Conditions, metav1.Condition{
		Type:               konfidence.ConditionTypeSucceeded,
		Status:             status,
		ObservedGeneration: vectorPromotion.Generation,
		Reason:             reason,
		Message:            message,
	})
	if reasonOnlyChange {
		current := meta.FindStatusCondition(vectorPromotion.Status.Conditions, konfidence.ConditionTypeSucceeded)
		current.LastTransitionTime = metav1.Now()
	}
	vectorPromotion.Status.State = promotion.DeriveState(vectorPromotion)
}

// patchPromotionStatus patches the promotion status if it changed relative to
// original. The patch is optimistic-locked: promotion status is also written
// by the external approver, and a stale write must conflict and retrigger
// reconciliation instead of silently overwriting the conditions array.
func patchPromotionStatus(
	ctx context.Context,
	c client.Client,
	vectorPromotion, original *konfidence.VectorPromotion,
) error {
	if reflect.DeepEqual(vectorPromotion.Status, original.Status) {
		return nil
	}
	patch := client.MergeFromWithOptions(original, client.MergeFromWithOptimisticLock{})
	if err := c.Status().Patch(ctx, vectorPromotion, patch); err != nil {
		return fmt.Errorf("failed to patch status of VectorPromotion %q in namespace %q: %w",
			vectorPromotion.Name, vectorPromotion.Namespace, err)
	}
	return nil
}
