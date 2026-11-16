package controller

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// resolveReferences resolves the target Stage, the source vector, and the source
// VectorTemplate (nil for Stage sources). A definitive miss is returned as a
// resolutionError for the Ready condition; an empty source vector with a nil error
// means the source exists but has not assembled a vector yet.
func (r *VectorPromotionConfigReconciler) resolveReferences(ctx context.Context, config *konfidence.VectorPromotionConfig) (*konfidence.Stage, string, *konfidence.VectorTemplate, error) {
	target, err := resolveTargetStage(ctx, r.Client, config.Namespace, config.Spec.Target)
	if err != nil {
		return nil, "", nil, err
	}
	sourceVector, sourceTemplate, err := r.resolveSource(ctx, config)
	if err != nil {
		return nil, "", nil, err
	}
	return target, sourceVector, sourceTemplate, nil
}

// resolveSource reads the vector currently offered by the source and returns the
// VectorTemplate when the source kind is VectorTemplate (nil for Stage sources).
func (r *VectorPromotionConfigReconciler) resolveSource(ctx context.Context, config *konfidence.VectorPromotionConfig) (string, *konfidence.VectorTemplate, error) {
	source := config.Spec.Source
	if source.Kind == konfidence.VectorTemplateKind {
		template := &konfidence.VectorTemplate{}
		key := types.NamespacedName{Namespace: config.Namespace, Name: source.Name}
		err := r.Get(ctx, key, template)
		if apierrors.IsNotFound(err) {
			return "", nil, &resolutionError{
				reason:  konfidence.VectorPromotionConfigSourceNotFoundReason,
				message: fmt.Sprintf("source vector template %q does not exist in namespace %q", key.Name, key.Namespace),
			}
		}
		if err != nil {
			return "", nil, fmt.Errorf("failed to fetch source vector template %q: %w", key.Name, err)
		}
		return template.Status.LatestVector, template, nil
	}

	namespace, err := resolveLandscapeNamespace(ctx, r.Client, config.Namespace, source.Landscape)
	if err != nil {
		return "", nil, err
	}
	stage := &konfidence.Stage{}
	key := types.NamespacedName{Namespace: namespace, Name: source.Name}
	err = r.Get(ctx, key, stage)
	if apierrors.IsNotFound(err) {
		return "", nil, &resolutionError{
			reason:  konfidence.VectorPromotionConfigSourceNotFoundReason,
			message: fmt.Sprintf("source stage %q does not exist in landscape namespace %q", key.Name, key.Namespace),
		}
	}
	if err != nil {
		return "", nil, fmt.Errorf("failed to fetch source stage %q in landscape namespace %q: %w", key.Name, key.Namespace, err)
	}
	return stage.Spec.Vector, nil, nil
}
