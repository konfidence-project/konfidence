package activation

import (
	"context"
	"fmt"

	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func InFinalStatusCondition(vectorActivation *landscape.VectorActivation) bool {
	if len(vectorActivation.Status.Conditions) == 0 {
		return false
	}
	return meta.IsStatusConditionTrue(vectorActivation.Status.Conditions, landscape.ActivationSucceeded) ||
		meta.IsStatusConditionTrue(vectorActivation.Status.Conditions, landscape.ActivationFailed) ||
		meta.IsStatusConditionTrue(vectorActivation.Status.Conditions, landscape.ActivationSkipped)
}

func PatchVectorActivationStatus(ctx context.Context, c client.Client, namespacedName types.NamespacedName, condition metav1.Condition) error {
	vectorActivation := &landscape.VectorActivation{}
	if err := c.Get(ctx, namespacedName, vectorActivation); err != nil {
		return fmt.Errorf("unable to fetch vectorActivation: %w", err)
	}

	// check if the condition already exists and is the same
	existingCondition := meta.FindStatusCondition(vectorActivation.Status.Conditions, condition.Type)
	if existingCondition != nil && existingCondition.Status == condition.Status &&
		existingCondition.Reason == condition.Reason &&
		existingCondition.Message == condition.Message {
		// No change, skip patch
		return nil
	}

	oldState := vectorActivation.DeepCopy()
	meta.SetStatusCondition(&vectorActivation.Status.Conditions, condition)
	if err := c.Status().Patch(ctx, vectorActivation, client.MergeFrom(oldState)); err != nil {
		return fmt.Errorf("unable to patch vectorActivation status: %w", err)
	}
	return nil
}
