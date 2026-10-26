package controller

import (
	"context"
	"fmt"
	"strings"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/konfidence-project/konfidence/pkg/promotiongate"
)

// reconcileGates runs while a promotion is Waiting or Ready, i.e. before
// execution starts: it seeds a Pending status entry for every declared gate
// that has none yet and mirrors the gates into the informational
// GatesSatisfied condition — False while any gate is open, True once all are
// satisfied. It must also run for Ready promotions because a gate writer
// satisfying the last gate flips the derived state straight to Ready; the
// next reconcile is the first to observe it.
//
// The controller never interprets gate types and never satisfies a gate
// itself; gate writers do that through pkg/promotiongate. Seeding only adds
// missing entries and the status patch is optimistic-locked, so a writer that
// set its gate before the first reconcile is never overwritten: a stale patch
// conflicts and is retried on fresh state. Promotions without gates carry no
// GatesSatisfied condition at all.
func (r *VectorPromotionReconciler) reconcileGates(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) error {
	if len(vectorPromotion.Spec.Gates) == 0 {
		return nil
	}
	log := logf.FromContext(ctx)
	original := vectorPromotion.DeepCopy()
	previous := meta.FindStatusCondition(original.Status.Conditions, konfidence.ConditionTypeGatesSatisfied)

	promotiongate.Seed(vectorPromotion)
	status, reason, message := gatesSummary(vectorPromotion)
	setGatesSatisfiedCondition(vectorPromotion, status, reason, message)
	if err := r.patchStatusWithEvent(ctx, vectorPromotion, original); err != nil {
		return err
	}

	if previous != nil && previous.Reason == reason && previous.Message == message {
		return nil
	}
	log.Info("promotion gates changed", "vector", vectorPromotion.Spec.Vector, "reason", reason, "message", message)
	switch reason {
	case konfidence.ReasonGateError:
		r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeWarning, "PromotionGateError", EventActionGates, message)
	case konfidence.ReasonGatesPending:
		r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionWaitingForGates", EventActionGates, message)
	default:
		r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionGatesSatisfied", EventActionGates, message)
	}
	return nil
}

// gatesSummary derives the GatesSatisfied condition from the gates. Errored
// gates take precedence over pending ones, so a gate that can never pass is
// surfaced instead of looking like an ordinary wait.
func gatesSummary(vectorPromotion *konfidence.VectorPromotion) (status metav1.ConditionStatus, reason, message string) {
	if promotiongate.AllSatisfied(vectorPromotion) {
		return metav1.ConditionTrue, konfidence.ReasonAllGatesSatisfied,
			fmt.Sprintf("all %d gates of the promotion of vector %q are satisfied",
				len(vectorPromotion.Spec.Gates), vectorPromotion.Spec.Vector)
	}
	if errored := promotiongate.Errored(vectorPromotion); len(errored) > 0 {
		details := make([]string, 0, len(errored))
		for _, gate := range errored {
			detail := gate.Type
			if why := strings.TrimSpace(gate.Reason + " " + gate.Message); why != "" {
				detail = fmt.Sprintf("%s (%s)", gate.Type, why)
			}
			details = append(details, detail)
		}
		return metav1.ConditionFalse, konfidence.ReasonGateError,
			fmt.Sprintf("promotion of vector %q is blocked by gates in error: %s",
				vectorPromotion.Spec.Vector, strings.Join(details, ", "))
	}
	return metav1.ConditionFalse, konfidence.ReasonGatesPending,
		fmt.Sprintf("promotion of vector %q is waiting for gates: %s",
			vectorPromotion.Spec.Vector, strings.Join(promotiongate.Open(vectorPromotion), ", "))
}

// setGatesSatisfiedCondition writes the GatesSatisfied condition on the
// in-memory object; the derived status.state is applied centrally when the
// status is patched.
func setGatesSatisfiedCondition(
	vectorPromotion *konfidence.VectorPromotion,
	status metav1.ConditionStatus,
	reason, message string,
) {
	meta.SetStatusCondition(&vectorPromotion.Status.Conditions, metav1.Condition{
		Type:               konfidence.ConditionTypeGatesSatisfied,
		Status:             status,
		ObservedGeneration: vectorPromotion.Generation,
		Reason:             reason,
		Message:            message,
	})
}
