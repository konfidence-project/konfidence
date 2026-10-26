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

// transition records a phase change on the promotion and emits a warning
// event when the status patch fails. See transition.
func (r *VectorPromotionReconciler) transition(
	ctx context.Context,
	vectorPromotion *konfidence.VectorPromotion,
	reason, message string,
	mutate ...func(*konfidence.VectorPromotion),
) error {
	err := transition(ctx, r.Client, vectorPromotion, reason, message, mutate...)
	if err != nil {
		r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeWarning, "StatusPatchFailed", EventActionStatusPatch, err.Error())
	}
	return err
}

// transition is the single way the promotion phase is written. It applies
// the optional mutations (other status fields that belong to the same
// transition), sets the Succeeded condition with the status that belongs to
// the reason, refreshes `status.state`, and persists everything in one
// optimistic-locked status patch; it is a no-op when nothing changed. The
// in-memory promotion is updated in place, so callers can keep deciding on
// it after the call.
//
// The patch is optimistic-locked because promotion status is also written by
// gate writers: a stale write must conflict and retrigger reconciliation
// instead of overwriting their gate entries.
//
// `meta.SetStatusCondition` only stamps LastTransitionTime when the condition
// status flips, but every reason change is a transition: the live phases
// (GatesPending → PromotionReady → PromotionRunning) share status Unknown.
// The timestamp is forced on a reason change so it always shows when the
// current phase began.
func transition(
	ctx context.Context,
	c client.Client,
	vectorPromotion *konfidence.VectorPromotion,
	reason, message string,
	mutate ...func(*konfidence.VectorPromotion),
) error {
	status, ok := promotion.ConditionStatusFor(reason)
	if !ok {
		return fmt.Errorf("unknown promotion phase %q for VectorPromotion %q", reason, vectorPromotion.Name)
	}

	original := vectorPromotion.DeepCopy()
	for _, m := range mutate {
		m(vectorPromotion)
	}

	previous := meta.FindStatusCondition(original.Status.Conditions, konfidence.ConditionTypeSucceeded)
	meta.SetStatusCondition(&vectorPromotion.Status.Conditions, metav1.Condition{
		Type:               konfidence.ConditionTypeSucceeded,
		Status:             status,
		ObservedGeneration: vectorPromotion.Generation,
		Reason:             reason,
		Message:            message,
	})
	if previous != nil && previous.Reason != reason {
		meta.FindStatusCondition(vectorPromotion.Status.Conditions, konfidence.ConditionTypeSucceeded).
			LastTransitionTime = metav1.Now()
	}
	vectorPromotion.Status.State = promotion.DeriveState(vectorPromotion)

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
