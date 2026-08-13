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

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
)

// execute resolves the promotion's own target snapshot, writes the pinned
// vector to that Stage, and marks the promotion succeeded. The config is
// never read: the promotion is self-contained, and the config reconciler
// aggregates promotion outcomes on its side.
//
// The function runs twice per promotion in the happy path (once to stamp
// Running, once more after the resulting status event) and re-runs wholesale
// after a crash: the Running patch is skipped once InProgress and
// promoteStage is the idempotency check for the stage write itself.
func (r *VectorPromotionReconciler) execute(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) error {
	original := vectorPromotion.DeepCopy()

	stage, err := resolveTargetStage(ctx, r.Client, vectorPromotion.Namespace, vectorPromotion.Spec.Target)
	if err != nil {
		return r.reportUnresolvedTarget(ctx, vectorPromotion, original, err)
	}

	if !promotion.IsInProgress(vectorPromotion) {
		setPromotionCondition(vectorPromotion, metav1.ConditionFalse, konfidence.ReasonPromotionRunning,
			fmt.Sprintf("promoting vector %q to stage %q", vectorPromotion.Spec.Vector, stage.Name))
		if err := r.patchStatusWithEvent(ctx, vectorPromotion, original); err != nil {
			return err
		}
		original = vectorPromotion.DeepCopy()
	}

	if err := r.promoteStage(ctx, stage, vectorPromotion); err != nil {
		return err
	}

	msg := fmt.Sprintf("promoted vector %q to stage %q in namespace %q",
		vectorPromotion.Spec.Vector, stage.Name, stage.Namespace)
	setPromotionCondition(vectorPromotion, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, msg)
	vectorPromotion.Status.PromotedStageRef = &corev1.TypedObjectReference{
		APIGroup:  ptr.To(konfidence.GroupVersion.Group),
		Kind:      konfidence.StageKind,
		Name:      stage.Name,
		Namespace: ptr.To(stage.Namespace),
	}
	if err := r.patchStatusWithEvent(ctx, vectorPromotion, original); err != nil {
		return err
	}
	r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionSuccessful", EventActionExecution, msg)
	return nil
}

// reportUnresolvedTarget surfaces a definitive resolution failure on the
// promotion: it goes Blocked (non-terminal, retried). The config's Ready
// condition is owned by the config reconciler, which watches the same
// resources. Transient API errors are returned as-is for plain backoff.
func (r *VectorPromotionReconciler) reportUnresolvedTarget(ctx context.Context, vectorPromotion, original *konfidence.VectorPromotion, err error) error {
	var resErr *resolutionError
	if !errors.As(err, &resErr) {
		return err
	}
	message := fmt.Sprintf("cannot promote vector %q: %s", vectorPromotion.Spec.Vector, resErr.message)
	r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeWarning, "PromotionTargetUnresolved",
		EventActionExecution, message)
	setPromotionCondition(vectorPromotion, metav1.ConditionFalse, konfidence.ReasonPromotionTargetUnresolved, message)
	if patchErr := r.patchStatusWithEvent(ctx, vectorPromotion, original); patchErr != nil {
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
