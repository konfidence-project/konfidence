package controller

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
)

// createPromotionForDrift creates the next sequence-stamped promotion for the
// drifted source vector, unless a live promotion already pins it.
func (r *VectorPromotionConfigReconciler) createPromotionForDrift(ctx context.Context, config *konfidence.VectorPromotionConfig, sourceVector string) error {
	log := logf.FromContext(ctx)

	list := &konfidence.VectorPromotionList{}

	// we need all pre-existing vector promotions in order to supersede existing ones
	err := r.List(ctx, list,
		client.InNamespace(config.Namespace),
		client.MatchingFields{promotionConfigRefField: config.Name})
	if err != nil {
		return fmt.Errorf("failed to list promotions of config %q: %w", config.Name, err)
	}
	for i := range list.Items {
		// if there is a promotion already running at any point in time and this promotion caters towards the vector we're targeting we can assume that the promotion already exists
		if !promotion.IsTerminal(&list.Items[i]) && list.Items[i].Spec.Vector == sourceVector {
			return nil
		}
	}

	// The sequence is committed before the create: a crash in between leaves a
	// gap in the sequence, never a duplicate.
	//
	// We use a sequence here in order to get a true order and not just a pseudo-order based on a timestamp, that might not be monotonic
	if err := r.patchConfigStatus(ctx, config, func() { config.Status.Sequence++ }); err != nil {
		return err
	}

	vectorPromotion := &konfidence.VectorPromotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      promotionName(config.Name, config.Status.Sequence),
			Namespace: config.Namespace,
		},
		Spec: konfidence.VectorPromotionSpec{
			VectorPromotionConfigRef: config.Name,
			Vector:                   sourceVector,
			RequireApproval:          config.Spec.Source.Kind == konfidence.StageKind,
			TTLAfterFinished:         config.Spec.TTLAfterFinished,
			Sequence:                 config.Status.Sequence,
		},
	}
	if err := controllerutil.SetControllerReference(config, vectorPromotion, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on promotion %q: %w", vectorPromotion.Name, err)
	}
	if err := r.Create(ctx, vectorPromotion); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create promotion %q: %w", vectorPromotion.Name, err)
	}

	log.Info("created promotion for drifted source",
		"promotion", vectorPromotion.Name,
		"vector", sourceVector,
		"sequence", vectorPromotion.Spec.Sequence,
		"requireApproval", vectorPromotion.Spec.RequireApproval)
	r.Recorder.Eventf(config, vectorPromotion, corev1.EventTypeNormal, "VectorPromotionCreated",
		EventActionDriftDetection,
		fmt.Sprintf("created promotion %q for vector %q", vectorPromotion.Name, sourceVector))
	return nil
}
