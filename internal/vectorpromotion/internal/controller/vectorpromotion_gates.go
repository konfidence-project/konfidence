package controller

import (
	"context"
	"fmt"
	"strings"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/konfidence-project/konfidence/pkg/promotiongate"
)

// reconcileGates handles the gate phase of a Waiting promotion. It seeds a
// Pending status entry for every declared gate that has none yet and records
// the outcome as the phase on the Succeeded condition (status Unknown):
// GateError or GatesPending while a gate is open, PromotionReady once every
// gate is satisfied (immediately for promotions without gates). Writing
// PromotionReady freezes the gates: pkg/promotiongate refuses changes from
// then on.
//
// The controller never interprets gate types and never satisfies a gate
// itself; gate writers do that through pkg/promotiongate. Seeding only adds
// missing entries and the status patch is optimistic-locked, so a gate written
// concurrently is never overwritten: a stale patch conflicts and is retried on
// fresh state. The same lock makes "all gates satisfied → Ready" atomic with
// respect to a gate being reopened.
func (r *VectorPromotionReconciler) reconcileGates(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) error {
	log := logf.FromContext(ctx)
	previous := meta.FindStatusCondition(vectorPromotion.Status.Conditions, konfidence.ConditionTypeSucceeded).DeepCopy()

	// gatesSummary treats a missing entry as Pending, so it can run before seeding.
	reason, message := gatesSummary(vectorPromotion)
	seed := func(p *konfidence.VectorPromotion) { promotiongate.Seed(p) }
	if err := r.transition(ctx, vectorPromotion, reason, message, seed); err != nil {
		return err
	}

	if previous != nil && previous.Reason == reason && previous.Message == message {
		return nil
	}
	log.Info("promotion gates evaluated", "vector", vectorPromotion.Spec.Vector, "reason", reason, "message", message)
	switch reason {
	case konfidence.ReasonGateError:
		r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeWarning, "PromotionGateError", EventActionGates, message)
	case konfidence.ReasonGatesPending:
		r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionWaitingForGates", EventActionGates, message)
	default:
		r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionReady", EventActionGates, message)
	}
	return nil
}

// gatesSummary maps the gates onto the gate-phase reason. Errored gates take
// precedence over pending ones, so a gate that can never pass is surfaced
// instead of looking like an ordinary wait.
func gatesSummary(vectorPromotion *konfidence.VectorPromotion) (reason, message string) {
	if promotiongate.AllSatisfied(vectorPromotion) {
		if len(vectorPromotion.Spec.Gates) == 0 {
			return konfidence.ReasonPromotionReady,
				fmt.Sprintf("promotion of vector %q declares no gates", vectorPromotion.Spec.Vector)
		}
		return konfidence.ReasonPromotionReady,
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
		return konfidence.ReasonGateError,
			fmt.Sprintf("promotion of vector %q is blocked by gates in error: %s",
				vectorPromotion.Spec.Vector, strings.Join(details, ", "))
	}
	return konfidence.ReasonGatesPending,
		fmt.Sprintf("promotion of vector %q is waiting for gates: %s",
			vectorPromotion.Spec.Vector, strings.Join(promotiongate.Open(vectorPromotion), ", "))
}
