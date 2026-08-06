package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	utils "github.com/konfidence-project/konfidence/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
)

// resolutionError is a definitive target-resolution failure that is surfaced
// on the VectorPromotionConfig, as opposed to a transient API error.
type resolutionError struct {
	reason  string
	message string
}

func (e *resolutionError) Error() string { return e.message }

// execute resolves the target, then writes the pinned vector to the target
// Stage and marks the promotion succeeded. A missing config is terminal; an
// unresolvable target is surfaced on the config and retried with backoff
// while the promotion stays approved.
//
// The function runs twice per promotion in the happy path (once to stamp
// Running, once more after the resulting status event) and re-runs wholesale
// after a crash: the Running patch is skipped once InProgress and
// promoteStage is the idempotency check for the stage write itself.
func (r *VectorPromotionReconciler) execute(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) error {
	log := logf.FromContext(ctx)
	original := vectorPromotion.DeepCopy()

	config, err := getPromotionConfig(ctx, r.Client, vectorPromotion)
	if apierrors.IsNotFound(err) {
		msg := fmt.Sprintf("promotion configuration %q not found", vectorPromotion.Spec.VectorPromotionConfigRef)
		log.Info("promotion configuration not found, failing terminally", "config", vectorPromotion.Spec.VectorPromotionConfigRef)
		setPromotionCondition(vectorPromotion, metav1.ConditionFalse, konfidence.ReasonPromotionConfigurationNotFound, msg)
		return r.patchStatusWithEvent(ctx, vectorPromotion, original)
	}
	if err != nil {
		return fmt.Errorf("failed to fetch promotion configuration: %w", err)
	}

	stage, err := resolveTargetStage(ctx, r.Client, config)
	if err != nil {
		return r.reportUnresolvedTarget(ctx, vectorPromotion, original, config, err)
	}

	if !promotion.IsInProgress(vectorPromotion) {
		setPromotionCondition(vectorPromotion, metav1.ConditionFalse, konfidence.ReasonPromotionRunning, "promotion is executing")
		if err := r.patchStatusWithEvent(ctx, vectorPromotion, original); err != nil {
			return err
		}

		if err := r.propagateToConfig(ctx, config, vectorPromotion); err != nil {
			return err
		}

		original = vectorPromotion.DeepCopy()
	}

	if err := r.promoteStage(ctx, stage, vectorPromotion); err != nil {
		return err
	}

	msg := fmt.Sprintf("promoted vector to stage %q in namespace %q", stage.Name, stage.Namespace)
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
	return r.propagateToConfig(ctx, config, vectorPromotion)
}

// reportUnresolvedTarget surfaces a definitive resolution failure on the
// promotion: it goes Blocked (non-terminal, retried) and mirrors onto the
// config's last-promotion view. The config's Ready condition itself is owned
// by the config reconciler, which watches the same resources. Transient API
// errors are returned as-is for plain backoff.
func (r *VectorPromotionReconciler) reportUnresolvedTarget(ctx context.Context, vectorPromotion, original *konfidence.VectorPromotion, config *konfidence.VectorPromotionConfig, err error) error {
	var resErr *resolutionError
	if !errors.As(err, &resErr) {
		return err
	}
	r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeWarning, "PromotionTargetUnresolved",
		EventActionExecution, resErr.message)
	setPromotionCondition(vectorPromotion, metav1.ConditionFalse, konfidence.ReasonPromotionTargetUnresolved, resErr.message)
	if patchErr := r.patchStatusWithEvent(ctx, vectorPromotion, original); patchErr != nil {
		return errors.Join(err, patchErr)
	}
	if patchErr := r.propagateToConfig(ctx, config, vectorPromotion); patchErr != nil {
		return errors.Join(err, patchErr)
	}
	return err
}

// resolveLandscapeNamespace resolves a Landscape name (in the config's
// namespace) to the namespace it manages.
func resolveLandscapeNamespace(ctx context.Context, c client.Client, namespace, name string) (string, error) {
	landscape := &konfidence.Landscape{}
	key := types.NamespacedName{Namespace: namespace, Name: name}
	err := c.Get(ctx, key, landscape)
	if apierrors.IsNotFound(err) {
		return "", &resolutionError{
			reason:  konfidence.VectorPromotionConfigLandscapeNotFoundReason,
			message: fmt.Sprintf("landscape %q does not exist in namespace %q", key.Name, key.Namespace),
		}
	}
	if err != nil {
		return "", fmt.Errorf("failed to fetch landscape %q: %w", key.Name, err)
	}
	if landscape.Status.Namespace == "" {
		return "", &resolutionError{
			reason:  konfidence.VectorPromotionConfigLandscapeNotReadyReason,
			message: fmt.Sprintf("landscape %q has no managed namespace yet", landscape.Name),
		}
	}
	return landscape.Status.Namespace, nil
}

// resolveTargetStage resolves the config's target Stage through the Landscape
// in the config's namespace. The Stage is never created here: a missing
// landscape or stage is reported as a resolutionError for the user to act on.
func resolveTargetStage(ctx context.Context, c client.Client, config *konfidence.VectorPromotionConfig) (*konfidence.Stage, error) {
	namespace, err := resolveLandscapeNamespace(ctx, c, config.Namespace, config.Spec.Target.Landscape)
	if err != nil {
		return nil, err
	}

	stage := &konfidence.Stage{}
	key := types.NamespacedName{Namespace: namespace, Name: config.Spec.Target.Name}
	err = c.Get(ctx, key, stage)
	if apierrors.IsNotFound(err) {
		return nil, &resolutionError{
			reason: konfidence.VectorPromotionConfigStageNotFoundReason,
			message: fmt.Sprintf("stage %q does not exist in landscape namespace %q; create it before promoting",
				key.Name, key.Namespace),
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stage %q in landscape namespace %q: %w", key.Name, key.Namespace, err)
	}
	return stage, nil
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

// propagateToConfig uses a plain merge patch: config status writers are
// field-disjoint (lastPromotion* here, conditions and sequence in the config
// reconciler), so unlocked patches cannot clobber each other.

// propagateToConfig mirrors the promotion's conditions onto its config so the
// config always shows the outcome of the promotion currently acting on it.
// Only the executing promotion propagates; superseded losers stay silent.
func (r *VectorPromotionReconciler) propagateToConfig(ctx context.Context, config *konfidence.VectorPromotionConfig, vectorPromotion *konfidence.VectorPromotion) error {
	original := config.DeepCopy()
	config.Status.LastPromotionConditions = vectorPromotion.Status.Conditions
	if promotion.IsSucceeded(vectorPromotion) {
		config.Status.LastSuccessfulPromotionConditions = vectorPromotion.Status.Conditions
	}
	if reflect.DeepEqual(config.Status, original.Status) {
		return nil
	}
	if err := r.Status().Patch(ctx, config, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("failed to propagate promotion conditions to VectorPromotionConfig %q in namespace %q: %w",
			config.Name, config.Namespace, err)
	}
	return nil
}
