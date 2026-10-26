package controller

import (
	"context"
	"errors"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	utils "github.com/konfidence-project/konfidence/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// execute resolves the promotion's own target snapshot, writes the pinned
// vector to that Stage, and marks the promotion succeeded. The config is
// never read: the promotion is self-contained, and the config reconciler
// aggregates promotion outcomes on its side.
//
// The function runs twice per promotion in the happy path (once to stamp
// startTime, once more after the resulting status event) and re-runs
// wholesale after a crash: startTime is only set once and promoteStage is the
// idempotency check for the stage write itself.
func (r *VectorPromotionReconciler) execute(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) error {
	stage, err := resolveTargetStage(ctx, r.Client, vectorPromotion.Namespace, vectorPromotion.Spec.Target)
	if err != nil {
		return r.reportUnresolvedTarget(ctx, vectorPromotion, err)
	}

	resolved := setCondition(konfidence.ConditionTypeTargetResolved, metav1.ConditionTrue, konfidence.ReasonTargetResolved,
		fmt.Sprintf("target stage %q resolved in namespace %q", stage.Name, stage.Namespace))
	if vectorPromotion.Status.StartTime == nil {
		start := func(p *konfidence.VectorPromotion) {
			now := metav1.Now()
			p.Status.StartTime = &now
		}
		running := setCondition(konfidence.ConditionTypeSucceeded, metav1.ConditionUnknown, konfidence.ReasonPromotionRunning,
			fmt.Sprintf("promoting vector %q to stage %q", vectorPromotion.Spec.Vector, stage.Name))
		if err := r.updateStatus(ctx, vectorPromotion, resolved, start, running); err != nil {
			return err
		}
	}

	if err := r.promoteStage(ctx, stage, vectorPromotion); err != nil {
		return err
	}

	msg := fmt.Sprintf("promoted vector %q to stage %q in namespace %q",
		vectorPromotion.Spec.Vector, stage.Name, stage.Namespace)
	recordPromotedStage := func(p *konfidence.VectorPromotion) {
		p.Status.PromotedStageRef = &corev1.TypedObjectReference{
			APIGroup:  ptr.To(konfidence.GroupVersion.Group),
			Kind:      konfidence.StageKind,
			Name:      stage.Name,
			Namespace: ptr.To(stage.Namespace),
		}
	}
	if err := r.updateStatus(ctx, vectorPromotion, resolved, recordPromotedStage,
		finish(metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, msg)); err != nil {
		return err
	}
	r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionSuccessful", EventActionExecution, msg)
	return nil
}

// reportUnresolvedTarget surfaces a definitive resolution failure on the
// TargetResolved condition; the promotion stays active and is retried. The
// config's Ready condition is owned by the config reconciler, which watches
// the same resources. Transient API errors are returned as-is for plain
// backoff.
func (r *VectorPromotionReconciler) reportUnresolvedTarget(ctx context.Context, vectorPromotion *konfidence.VectorPromotion, err error) error {
	var resErr *resolutionError
	if !errors.As(err, &resErr) {
		return err
	}
	message := fmt.Sprintf("cannot promote vector %q: %s", vectorPromotion.Spec.Vector, resErr.message)
	r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeWarning, "PromotionTargetUnresolved",
		EventActionExecution, message)
	unresolved := setCondition(konfidence.ConditionTypeTargetResolved, metav1.ConditionFalse, resErr.reason, message)
	if patchErr := r.updateStatus(ctx, vectorPromotion, unresolved); patchErr != nil {
		return errors.Join(err, patchErr)
	}
	return err
}

// promoteStage writes the promoted vector to the stage spec if it differs and
// records the writing promotion for provenance. The equality check makes
// crash-resume re-runs no-ops.
func (r *VectorPromotionReconciler) promoteStage(ctx context.Context, stage *konfidence.Stage, vectorPromotion *konfidence.VectorPromotion) error {
	log := logf.FromContext(ctx)

	promotedBy := fmt.Sprintf("%s/%s", vectorPromotion.Namespace, vectorPromotion.Name)
	if stage.Spec.Vector == vectorPromotion.Spec.Vector && stage.Annotations[utils.PromotedByAnnotation] == promotedBy {
		return nil
	}
	original := stage.DeepCopy()
	previousVector := stage.Spec.Vector
	stage.Spec.Vector = vectorPromotion.Spec.Vector
	if stage.Annotations == nil {
		stage.Annotations = map[string]string{}
	}
	stage.Annotations[utils.PromotedByAnnotation] = promotedBy
	if err := r.Patch(ctx, stage, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("failed to patch vector onto stage %q in namespace %q: %w", stage.Name, stage.Namespace, err)
	}
	log.Info("promoted vector to stage",
		"stage", stage.Namespace+"/"+stage.Name,
		"vector", vectorPromotion.Spec.Vector,
		"previousVector", previousVector)
	return nil
}
