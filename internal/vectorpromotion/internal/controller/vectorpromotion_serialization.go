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
	// promotion of the same config is executing.
	siblingBlockedRequeueInterval = 10 * time.Second

	// executionDeadline is how long a promotion may stay InProgress. Execution
	// is a single resolve+patch, so a Running condition this old is a crashed
	// or wedged attempt. Whoever observes the overrun (the promotion itself on
	// a retry, or a blocked sibling) retires it to Failed/PromotionTimedOut,
	// keeping the one-InProgress-per-config invariant intact instead of
	// ignoring it.
	executionDeadline = 5 * time.Minute
)

// reconcileExecution serializes execution per config: at most one promotion is
// in progress, only the newest approved one executes, and stale approved
// promotions are superseded.
func (r *VectorPromotionReconciler) reconcileExecution(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if promotion.IsInProgress(vectorPromotion) && promotion.InProgressLongerThan(vectorPromotion, executionDeadline) {
		return ctrl.Result{}, r.timeOut(ctx, vectorPromotion)
	}

	siblings, err := listSiblingPromotions(ctx, r.Client, vectorPromotion)
	if err != nil {
		return ctrl.Result{}, err
	}

	for i := range siblings {
		sibling := &siblings[i]
		if sibling.UID == vectorPromotion.UID || !promotion.IsInProgress(sibling) {
			continue
		}
		if promotion.InProgressLongerThan(sibling, executionDeadline) {
			if err := r.timeOut(ctx, sibling); err != nil {
				return ctrl.Result{}, err
			}
			continue
		}
		log.V(1).Info("execution blocked by in-progress sibling", "sibling", sibling.Name)
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

// timeOut retires a promotion stuck InProgress past the execution deadline.
// The optimistic-locked patch makes racing observers conflict harmlessly.
func (r *VectorPromotionReconciler) timeOut(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) error {
	log := logf.FromContext(ctx)

	message := fmt.Sprintf("promotion of vector %q stayed in progress for over %s and was retired",
		vectorPromotion.Spec.Vector, executionDeadline)
	original := vectorPromotion.DeepCopy()
	setPromotionCondition(vectorPromotion, metav1.ConditionFalse, konfidence.ReasonPromotionTimedOut, message)
	if err := r.patchStatusWithEvent(ctx, vectorPromotion, original); err != nil {
		return err
	}
	log.Info("timed out promotion", "promotion", vectorPromotion.Name, "deadline", executionDeadline)
	r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeWarning, "PromotionTimedOut", EventActionExecution, message)
	return nil
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
		if sibling.UID == newest.UID || promotion.IsTerminal(sibling) || promotion.Newer(sibling, newest) {
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
	original := vectorPromotion.DeepCopy()
	setPromotionCondition(vectorPromotion, metav1.ConditionFalse, konfidence.ReasonPromotionSuperseded, message)
	if err := r.patchStatusWithEvent(ctx, vectorPromotion, original); err != nil {
		return err
	}
	log.Info("superseded promotion", "promotion", vectorPromotion.Name, "supersededBy", supersededBy)
	r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionSuperseded", EventActionExecution, message)
	return nil
}
