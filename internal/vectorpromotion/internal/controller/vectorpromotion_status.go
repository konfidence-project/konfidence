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
)

// statusMutation edits the in-memory promotion status as part of one update.
type statusMutation func(*konfidence.VectorPromotion)

// updateStatus applies mutations and emits a warning event when the status
// patch fails. See updateStatus.
func (r *VectorPromotionReconciler) updateStatus(
	ctx context.Context, vectorPromotion *konfidence.VectorPromotion, mutations ...statusMutation,
) error {
	err := updateStatus(ctx, r.Client, vectorPromotion, mutations...)
	if err != nil {
		r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeWarning, "StatusPatchFailed", EventActionStatusPatch, err.Error())
	}
	return err
}

// updateStatus applies the mutations to the in-memory promotion and persists
// them in one optimistic-locked status patch; it is a no-op when nothing
// changed. The promotion is updated in place, so callers keep deciding on it.
// The patch is optimistic-locked because promotion status is also written by
// gate writers: a stale write must conflict and retrigger reconciliation
// instead of overwriting their gate entries.
func updateStatus(
	ctx context.Context, c client.Client, vectorPromotion *konfidence.VectorPromotion, mutations ...statusMutation,
) error {
	original := vectorPromotion.DeepCopy()
	for _, mutate := range mutations {
		mutate(vectorPromotion)
	}
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

// setCondition returns a mutation that sets a condition with the standard
// semantics: lastTransitionTime only moves when the status flips.
func setCondition(conditionType string, status metav1.ConditionStatus, reason, message string) statusMutation {
	return func(p *konfidence.VectorPromotion) {
		meta.SetStatusCondition(&p.Status.Conditions, metav1.Condition{
			Type:               conditionType,
			Status:             status,
			ObservedGeneration: p.Generation,
			Reason:             reason,
			Message:            message,
		})
	}
}

// initSucceededCondition returns a mutation that records the promotion as active
// (Succeeded=Unknown/PromotionPending) unless a Succeeded condition exists.
func initSucceededCondition(message string) statusMutation {
	return func(p *konfidence.VectorPromotion) {
		if meta.FindStatusCondition(p.Status.Conditions, konfidence.ConditionTypeSucceeded) == nil {
			setCondition(konfidence.ConditionTypeSucceeded, metav1.ConditionUnknown,
				konfidence.ReasonPromotionPending, message)(p)
		}
	}
}

// finish returns a mutation that ends the promotion: Succeeded True or False
// with the given reason, and completionTime.
func finish(status metav1.ConditionStatus, reason, message string) statusMutation {
	return func(p *konfidence.VectorPromotion) {
		setCondition(konfidence.ConditionTypeSucceeded, status, reason, message)(p)
		now := metav1.Now()
		p.Status.CompletionTime = &now
	}
}
