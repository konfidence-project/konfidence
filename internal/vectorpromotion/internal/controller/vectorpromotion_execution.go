package controller

import (
	"context"
	"encoding/json"
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

var errNewerPromotionExecuted = errors.New("a newer promotion has already executed")

// execute resolves the promotion's own target snapshot, writes the pinned
// vector to that Stage, and marks the promotion succeeded. The config is
// never read: the promotion is self-contained, and the config reconciler
// aggregates promotion outcomes on its side.
//
// startTime is only set once. Re-runs after a crash are idempotent, and the
// Stage's sequence watermark prevents a stale promotion from rolling it back.
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
		startedMsg := fmt.Sprintf("started promoting vector %q to stage %q", vectorPromotion.Spec.Vector, stage.Name)
		r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionExecutionStarted", EventActionExecution, startedMsg)
	}

	if err := r.promoteStage(ctx, stage, vectorPromotion); err != nil {
		if errors.Is(err, errNewerPromotionExecuted) {
			return r.markSuperseded(ctx, vectorPromotion, err.Error())
		}
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
// records provenance and the config's highest executed sequence in the same
// optimistic-locked patch. A stale Stage snapshot conflicts; a fresh snapshot
// with a newer sequence rejects execution. Neither can roll the Stage back.
func (r *VectorPromotionReconciler) promoteStage(ctx context.Context, stage *konfidence.Stage, vectorPromotion *konfidence.VectorPromotion) error {
	log := logf.FromContext(ctx)
	sequences := map[string]int64{}
	if raw := stage.Annotations[utils.PromotionSequencesAnnotation]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &sequences); err != nil {
			return fmt.Errorf("invalid promotion sequences on Stage %s/%s: %w", stage.Namespace, stage.Name, err)
		}
		if sequences == nil {
			return fmt.Errorf("invalid promotion sequences on Stage %s/%s: expected an object", stage.Namespace, stage.Name)
		}
		for _, sequence := range sequences {
			if sequence < 1 {
				return fmt.Errorf("invalid promotion sequences on Stage %s/%s: sequences must be positive", stage.Namespace, stage.Name)
			}
		}
	}
	configKey := promotionConfigKey(vectorPromotion)
	if sequence := sequences[configKey]; sequence > vectorPromotion.Spec.Sequence {
		return fmt.Errorf("%w: Stage %s/%s records sequence %d for config %s; promotion %s has sequence %d",
			errNewerPromotionExecuted, stage.Namespace, stage.Name, sequence, configKey, vectorPromotion.Name, vectorPromotion.Spec.Sequence)
	}

	promotedBy := fmt.Sprintf("%s/%s", vectorPromotion.Namespace, vectorPromotion.Name)
	if stage.Spec.Vector == vectorPromotion.Spec.Vector && stage.Annotations[utils.PromotedByAnnotation] == promotedBy &&
		sequences[configKey] == vectorPromotion.Spec.Sequence {
		return nil
	}
	sequences[configKey] = vectorPromotion.Spec.Sequence
	encoded, err := json.Marshal(sequences)
	if err != nil {
		return fmt.Errorf("encoding promotion sequences: %w", err)
	}
	original := stage.DeepCopy()
	previousVector := stage.Spec.Vector
	stage.Spec.Vector = vectorPromotion.Spec.Vector
	if stage.Annotations == nil {
		stage.Annotations = map[string]string{}
	}
	stage.Annotations[utils.PromotedByAnnotation] = promotedBy
	stage.Annotations[utils.PromotionSequencesAnnotation] = string(encoded)
	if err := r.Patch(ctx, stage, client.MergeFromWithOptions(original, client.MergeFromWithOptimisticLock{})); err != nil {
		return fmt.Errorf("failed to patch vector onto stage %q in namespace %q: %w", stage.Name, stage.Namespace, err)
	}
	log.Info("promoted vector to stage",
		"stage", stage.Namespace+"/"+stage.Name,
		"vector", vectorPromotion.Spec.Vector,
		"previousVector", previousVector)
	return nil
}

// Config UIDs distinguish recreated configs whose sequence starts over. The
// name-based fallback also supports manually created, unowned promotions.
func promotionConfigKey(p *konfidence.VectorPromotion) string {
	key := p.Namespace + "/" + p.Spec.VectorPromotionConfigName
	if owner := metav1.GetControllerOf(p); owner != nil && owner.APIVersion == konfidence.GroupVersion.String() &&
		owner.Kind == konfidence.VectorPromotionConfigKind && owner.Name == p.Spec.VectorPromotionConfigName && owner.UID != "" {
		key += "/" + string(owner.UID)
	}
	return key
}
