package controller

import (
	"context"
	"fmt"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
// in progress. A newer ready or terminal promotion makes this promotion obsolete,
// unless the newer promotion was rejected. Supersession is checked before both
// outstanding retention and execution, regardless of this promotion's gate state.
func (r *VectorPromotionReconciler) reconcileExecution(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	siblings, err := listSiblingPromotions(ctx, r.Client, vectorPromotion)
	if err != nil {
		return ctrl.Result{}, err
	}

	if newest := findNewerPromotionThatSupersedesCurrent(vectorPromotion, siblings); newest != nil {
		return ctrl.Result{}, r.supersede(ctx, vectorPromotion, newest)
	}
	if !promotion.IsReadyForExecution(vectorPromotion) {
		return ctrl.Result{}, r.enforceOutstandingRetention(ctx, vectorPromotion, siblings)
	}

	for i := range siblings {
		sibling := &siblings[i]
		if !promotion.IsRunning(sibling) {
			continue
		}
		log.V(1).Info("execution blocked by started sibling", "sibling", sibling.Name)
		return ctrl.Result{RequeueAfter: siblingBlockedRequeueInterval}, nil
	}

	return ctrl.Result{}, r.execute(ctx, vectorPromotion)
}

// findNewerPromotionThatSupersedesCurrent returns the highest-sequence sibling
// that makes current obsolete, or nil when current can remain active. It only
// selects a sibling; it does not change any resource's status.
// Waiting and ready promotions follow the same rule. Selecting the newest
// qualifying sibling only makes the supersession message deterministic.
func findNewerPromotionThatSupersedesCurrent(current *konfidence.VectorPromotion, siblings []konfidence.VectorPromotion) *konfidence.VectorPromotion {
	var newest *konfidence.VectorPromotion
	for i := range siblings {
		sibling := &siblings[i]
		if !canSupersede(sibling, current) {
			continue
		}
		if newest == nil || promotion.Newer(sibling, newest) {
			newest = sibling
		}
	}
	return newest
}

func canSupersede(newer, current *konfidence.VectorPromotion) bool {
	return promotion.Newer(newer, current) && supersedesOlderPromotions(newer)
}

// supersedesOlderPromotions defines which observed states prevent older
// promotions from executing. Gate errors and pending gates alone do not qualify.
func supersedesOlderPromotions(p *konfidence.VectorPromotion) bool {
	switch {
	case promotion.HasOverrideSet(p, konfidence.PromotionOverrideTypeReject):
		// Rejection never displaces older promotions, even before the rejection
		// is reconciled or when gates had already been satisfied.
		return false
	case promotion.IsTerminal(p):
		// Both Succeeded=True and Succeeded=False make older promotions obsolete.
		// This includes failures, supersession, and outstanding-limit eviction.
		return true
	case promotion.IsReadyForExecution(p):
		// Includes satisfied gates and explicit bypass, even while execution
		// is blocked or retrying. Older promotions must not act as a fallback.
		return true
	default:
		return false
	}
}

// Each waiting promotion counts newer waiting siblings and terminates only
// itself if it falls outside the configured window. No sibling is written.
func (r *VectorPromotionReconciler) enforceOutstandingRetention(ctx context.Context, current *konfidence.VectorPromotion,
	siblings []konfidence.VectorPromotion) error {
	keep := defaultKeepOutstandingPromotions
	config, err := getPromotionConfig(ctx, r.Client, current)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to fetch promotion configuration for outstanding retention: %w", err)
	}
	if err == nil && config.Spec.KeepOutstandingPromotions != nil {
		keep = int(*config.Spec.KeepOutstandingPromotions)
	}
	if keep == 0 {
		return nil
	}
	newer := 0
	for i := range siblings {
		if promotion.IsWaiting(&siblings[i]) && promotion.Newer(&siblings[i], current) {
			newer++
		}
	}
	if newer < keep {
		return nil
	}
	message := fmt.Sprintf("promotion of vector %q was superseded because the limit of %d outstanding promotions was exceeded", current.Spec.Vector, keep)
	return r.updateStatus(ctx, current,
		finish(metav1.ConditionFalse, konfidence.ReasonPromotionOutstandingLimitReached, message))
}

// supersede terminates a promotion because a newer ready or terminal promotion
// for the same config exists.
func (r *VectorPromotionReconciler) supersede(ctx context.Context, vectorPromotion, newest *konfidence.VectorPromotion) error {
	message := fmt.Sprintf("promotion of vector %q was superseded by promotion %q of vector %q",
		vectorPromotion.Spec.Vector, newest.Name, newest.Spec.Vector)
	return r.markSuperseded(ctx, vectorPromotion, message)
}

func (r *VectorPromotionReconciler) markSuperseded(ctx context.Context, vectorPromotion *konfidence.VectorPromotion, message string) error {
	if err := r.updateStatus(ctx, vectorPromotion,
		finish(metav1.ConditionFalse, konfidence.ReasonPromotionSuperseded, message)); err != nil {
		return err
	}
	logf.FromContext(ctx).Info("superseded promotion", "promotion", vectorPromotion.Name, "message", message)
	r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionSuperseded", EventActionExecution, message)
	return nil
}
