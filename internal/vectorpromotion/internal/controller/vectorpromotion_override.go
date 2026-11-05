package controller

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *VectorPromotionReconciler) reject(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) error {
	override := vectorPromotion.Status.Override
	message := fmt.Sprintf("promotion of vector %q was rejected by %s: %s",
		vectorPromotion.Spec.Vector, override.SetBy, override.Reason)
	if err := r.updateStatus(ctx, vectorPromotion,
		finish(metav1.ConditionFalse, konfidence.ReasonPromotionRejected, message)); err != nil {
		return err
	}
	logf.FromContext(ctx).Info("rejected promotion", "vector", vectorPromotion.Spec.Vector, "rejectedBy", override.SetBy)
	r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionRejected", EventActionOverride, message)
	return nil
}
