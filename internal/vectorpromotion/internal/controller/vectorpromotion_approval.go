package controller

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// reconcileApproval stamps the open approval gate onto a promotion that
// requires one. Promotions without an approval gate never reach this: they
// carry no Approved condition at all — a gate that does not exist leaves no
// record — and clear straight into execution.
func (r *VectorPromotionReconciler) reconcileApproval(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) error {
	log := logf.FromContext(ctx)
	original := vectorPromotion.DeepCopy()

	message := fmt.Sprintf("promotion of vector %q requires approval before execution", vectorPromotion.Spec.Vector)
	setApprovedCondition(vectorPromotion, metav1.ConditionFalse, konfidence.ReasonPromotionWaitingForApproval, message)
	if err := r.patchStatusWithEvent(ctx, vectorPromotion, original); err != nil {
		return err
	}
	log.Info("promotion waiting for approval", "vector", vectorPromotion.Spec.Vector)
	r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionWaitingForApproval",
		EventActionApproval, message)
	return nil
}

// setApprovedCondition writes the Approved condition on the in-memory object;
// the derived status.state is applied centrally when the status is patched.
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
}
