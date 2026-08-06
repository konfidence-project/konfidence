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

	// inProgressStaleTimeout is how long a sibling may claim InProgress before
	// it stops blocking others. Execution is a single resolve+patch, so a
	// Running condition this old is a crashed or wedged attempt, and honoring
	// it forever would deadlock the config. Note the stale promotion itself is
	// only driven terminal once a newer approved sibling executes and
	// supersedes it; see doc.go, "Crash recovery".
	inProgressStaleTimeout = 5 * time.Minute
)

// reconcileExecution serializes execution per config: at most one promotion is
// in progress, only the newest approved one executes, and stale approved
// promotions are superseded.
func (r *VectorPromotionReconciler) reconcileExecution(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	siblings, err := listSiblingPromotions(ctx, r.Client, vectorPromotion)
	if err != nil {
		return ctrl.Result{}, err
	}

	for i := range siblings {
		sibling := &siblings[i]
		if sibling.UID == vectorPromotion.UID || !promotion.IsInProgress(sibling) {
			continue
		}
		if promotion.InProgressLongerThan(sibling, inProgressStaleTimeout) {
			log.Info("ignoring stale in-progress sibling", "sibling", sibling.Name, "staleAfter", inProgressStaleTimeout)
			r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeWarning, "StaleSiblingIgnored",
				EventActionExecution,
				fmt.Sprintf("sibling promotion %q has been in progress for over %s; ignoring it", sibling.Name, inProgressStaleTimeout))
			continue
		}
		log.V(1).Info("execution blocked by in-progress sibling", "sibling", sibling.Name)
		return ctrl.Result{RequeueAfter: siblingBlockedRequeueInterval}, nil
	}

	newest := promotion.NewestApproved(siblings)
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
func (r *VectorPromotionReconciler) supersedeStaleSiblings(ctx context.Context, newest *konfidence.VectorPromotion, siblings []konfidence.VectorPromotion) error {
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

	message := "promotion was superseded by a newer approved promotion"
	supersededBy := ""
	if newest != nil {
		message = fmt.Sprintf("promotion was superseded by newer approved promotion %q", newest.Name)
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
