package activation

import (
	"context"
	"fmt"

	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func InFinalStatusCondition(vectorActivation *star.VectorActivation) bool {
	if len(vectorActivation.Status.Conditions) == 0 {
		return false
	}
	return meta.IsStatusConditionTrue(vectorActivation.Status.Conditions, star.ActivationSucceeded) ||
		meta.IsStatusConditionTrue(vectorActivation.Status.Conditions, star.ActivationFailed) ||
		meta.IsStatusConditionTrue(vectorActivation.Status.Conditions, star.ActivationSkipped)
}

func UpdateVectorActivationStatus(ctx context.Context, c client.Client, activation *star.VectorActivation, condition metav1.Condition) error {
	if condition.Type == "" || condition.Status == "" {
		return fmt.Errorf("unable to update vectorActivation status: condition type and status must be set")
	}

	condition.LastTransitionTime = metav1.Now()
	activation.Status.Conditions = []metav1.Condition{condition}

	if err := c.Status().Update(ctx, activation); err != nil {
		return fmt.Errorf("unable to update vectorActivation status: %w", err)
	}
	return nil
}
