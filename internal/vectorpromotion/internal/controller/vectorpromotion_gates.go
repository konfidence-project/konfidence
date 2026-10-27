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

// reconcileGates handles a promotion whose gates are not satisfied yet. It
// seeds a Pending status entry for every declared gate that has none, records
// the active promotion (Succeeded=Unknown) on first sight, and reports the
// gates on the GatesSatisfied condition: False (GatesPending or GateError)
// while a gate is open, True once every gate is satisfied (immediately for
// promotions without gates). GatesSatisfied latches: it is never written again
// after True, and pkg/promotiongate refuses gate changes from then on.
func (r *VectorPromotionReconciler) reconcileGates(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) error {
	log := logf.FromContext(ctx)
	previous := meta.FindStatusCondition(vectorPromotion.Status.Conditions, konfidence.ConditionTypeGatesSatisfied).DeepCopy()

	status, reason, message := gatesSummary(vectorPromotion)
	seed := func(p *konfidence.VectorPromotion) { promotiongate.Seed(p) }
	if err := r.updateStatus(ctx, vectorPromotion,
		seed,
		initSucceeded(fmt.Sprintf("promotion of vector %q has not started", vectorPromotion.Spec.Vector)),
		setCondition(konfidence.ConditionTypeGatesSatisfied, status, reason, message),
	); err != nil {
		return err
	}

	if previous != nil && previous.Status == status && previous.Message == message {
		return nil
	}
	log.Info("promotion gates evaluated", "vector", vectorPromotion.Spec.Vector, "satisfied", status, "message", message)
	switch {
	case status == metav1.ConditionTrue:
		r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionGatesSatisfied", EventActionGates, message)
	case len(promotiongate.Errored(vectorPromotion)) > 0:
		r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeWarning, "PromotionGateError", EventActionGates, message)
	default:
		r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionWaitingForGates", EventActionGates, message)
	}
	return nil
}

// gatesSummary derives the GatesSatisfied condition from the gates. Errored
// gates take precedence over pending ones, so a gate that can never pass is
// surfaced instead of looking like an ordinary wait.
func gatesSummary(vectorPromotion *konfidence.VectorPromotion) (status metav1.ConditionStatus, reason, message string) {
	if promotiongate.AllSatisfied(vectorPromotion) {
		if len(vectorPromotion.Spec.Gates) == 0 {
			return metav1.ConditionTrue, konfidence.ReasonAllGatesSatisfied,
				fmt.Sprintf("promotion of vector %q declares no gates", vectorPromotion.Spec.Vector)
		}
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
