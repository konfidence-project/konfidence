package controller

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Shared landscape/stage resolution for the promotion controllers: the
// execution controller resolves a promotion's target snapshot, the config
// reconciler resolves the config's references.

// resolutionError is a definitive target-resolution failure that is surfaced
// on the VectorPromotionConfig, as opposed to a transient API error.
type resolutionError struct {
	reason  string
	message string
}

func (e *resolutionError) Error() string { return e.message }

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

// resolveTargetStage resolves a target reference through the Landscape in the
// given namespace. The Stage is never created here: a missing landscape or
// stage is reported as a resolutionError for the user to act on.
func resolveTargetStage(ctx context.Context, c client.Client, namespace string, target konfidence.PromotionTargetReference) (*konfidence.Stage, error) {
	landscapeNamespace, err := resolveLandscapeNamespace(ctx, c, namespace, target.Landscape)
	if err != nil {
		return nil, err
	}

	stage := &konfidence.Stage{}
	key := types.NamespacedName{Namespace: landscapeNamespace, Name: target.Name}
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
