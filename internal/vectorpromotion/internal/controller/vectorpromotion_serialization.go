package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
)

const (
	// siblingBlockedRequeueInterval paces re-evaluation while another
	// promotion of the same config has started.
	siblingBlockedRequeueInterval = 10 * time.Second
)

// reconcileExecution serializes execution per config: at most one promotion is
// in progress, only the newest cleared one executes, and stale cleared
// promotions are superseded.
func (r *VectorPromotionReconciler) reconcileExecution(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	siblings, err := listSiblingPromotions(ctx, r.Client, vectorPromotion)
	if err != nil {
		return ctrl.Result{}, err
	}

	for i := range siblings {
		sibling := &siblings[i]
		if sibling.UID == vectorPromotion.UID || !promotion.IsRunning(sibling) {
			continue
		}
		log.V(1).Info("execution blocked by started sibling", "sibling", sibling.Name)
		return ctrl.Result{RequeueAfter: siblingBlockedRequeueInterval}, nil
	}

	newest := promotion.NewestCleared(siblings)
	if newest == nil || newest.UID != vectorPromotion.UID {
		return ctrl.Result{}, r.supersede(ctx, vectorPromotion, newest)
	}

	if err := r.supersedeStaleSiblings(ctx, vectorPromotion, siblings); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, r.execute(ctx, vectorPromotion)
}

// supersedeStaleSiblings marks older non-terminal siblings as superseded
// before the newest approved promotion executes. Siblings created after the
// executing promotion (e.g. still waiting for approval) keep their chance to
// run later.
func (r *VectorPromotionReconciler) supersedeStaleSiblings(
	ctx context.Context, newest *konfidence.VectorPromotion, siblings []konfidence.VectorPromotion,
) error {
	var errs []error
	for i := range siblings {
		sibling := &siblings[i]
		if sibling.UID == newest.UID || promotion.IsFinished(sibling) || promotion.Newer(sibling, newest) {
			continue
		}
		errs = append(errs, r.supersede(ctx, sibling, newest))
	}
	return errors.Join(errs...)
}

// supersede terminates a promotion because a newer approved promotion for the
// same config exists.
func (r *VectorPromotionReconciler) supersede(ctx context.Context, vectorPromotion, newest *konfidence.VectorPromotion) error {
	log := logf.FromContext(ctx)

	message := fmt.Sprintf("promotion of vector %q was superseded by a newer approved promotion",
		vectorPromotion.Spec.Vector)
	supersededBy := ""
	if newest != nil {
		message = fmt.Sprintf("promotion of vector %q was superseded by promotion %q of vector %q",
			vectorPromotion.Spec.Vector, newest.Name, newest.Spec.Vector)
		supersededBy = newest.Name
	}
	if err := r.updateStatus(ctx, vectorPromotion,
		finish(metav1.ConditionFalse, konfidence.ReasonPromotionSuperseded, message)); err != nil {
		return err
	}
	log.Info("superseded promotion", "promotion", vectorPromotion.Name, "supersededBy", supersededBy)
	r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionSuperseded", EventActionExecution, message)
	return nil
}
