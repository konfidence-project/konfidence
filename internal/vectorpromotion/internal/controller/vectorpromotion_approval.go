package controller

import (
	"context"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
)

// reconcileApproval moves an unapproved promotion into WaitingForApproval or
// approves it directly when it does not require approval. The status update
// retriggers reconciliation, which then proceeds to execution.
func (r *VectorPromotionReconciler) reconcileApproval(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) error {
	log := logf.FromContext(ctx)
	original := vectorPromotion.DeepCopy()

	if vectorPromotion.Spec.RequireApproval {
		setApprovedCondition(vectorPromotion, metav1.ConditionFalse, konfidence.ReasonPromotionWaitingForApproval,
			"promotion requires approval before execution")
		if err := r.patchStatusWithEvent(ctx, vectorPromotion, original); err != nil {
			return err
		}
		log.Info("promotion waiting for approval")
		r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionWaitingForApproval",
			EventActionApproval, "promotion is waiting for approval")
		return nil
	}

	setApprovedCondition(vectorPromotion, metav1.ConditionTrue, konfidence.ReasonPromotionAutoApproved,
		"promotion is approved automatically because it does not require approval")
	if err := r.patchStatusWithEvent(ctx, vectorPromotion, original); err != nil {
		return err
	}
	log.Info("promotion auto-approved")
	r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionAutoApproved",
		EventActionApproval, "promotion approved automatically")
	return nil
}

// setApprovedCondition writes the Approved condition and refreshes the
// derived status.state.
func setApprovedCondition(
	vectorPromotion *konfidence.VectorPromotion,
	status metav1.ConditionStatus,
	reason, message string,
) {
	meta.SetStatusCondition(&vectorPromotion.Status.Conditions, metav1.Condition{
		Type:               konfidence.ConditionTypeApproved,
		Status:             status,
		ObservedGeneration: vectorPromotion.Generation,
		Reason:             reason,
		Message:            message,
	})
	vectorPromotion.Status.State = promotion.DeriveState(vectorPromotion)
}
