package usage

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func CreateActivationUsage(
	ctx context.Context, c client.Client, stage *konfidence.Stage, activation *konfidence.VectorActivation,
) (*konfidence.StageVersionUsage, error) {
	log := logf.FromContext(ctx)

	stageVersionUsage := &konfidence.StageVersionUsage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      activationUsageName(stage, activation),
			Namespace: stage.Namespace,
			Labels:    map[string]string{ActivationStageVersionUsage: stage.Name},
		},
		Spec: konfidence.StageVersionUsageSpec{
			Reason:          StageVersionUsageActivationType,
			StageVersionRef: &konfidence.StageVersionReference{Name: activation.Spec.StageVersion},
		},
	}
	if err := controllerutil.SetOwnerReference(activation, stageVersionUsage, c.Scheme()); err != nil {
		return nil, fmt.Errorf("failed to set owner reference on activation stageVersionUsage: %w", err)
	}
	if err := c.Create(ctx, stageVersionUsage); err != nil {
		if errors.IsAlreadyExists(err) {
			return stageVersionUsage, nil
		}
		return nil, fmt.Errorf("failed to create activation stageVersionUsage: %w", err)
	}
	log.Info("Created activation stageVersionUsage", "stageVersionUsage", stageVersionUsage)
	return stageVersionUsage, nil
}

func DeleteActivationUsage(
	ctx context.Context, c client.Client, stage *konfidence.Stage, activation *konfidence.VectorActivation,
) error {
	log := logf.FromContext(ctx)
	activationUsage := &konfidence.StageVersionUsage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      activationUsageName(stage, activation),
			Namespace: stage.Namespace,
		},
	}
	if err := c.Delete(ctx, activationUsage); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("unable to delete activation stageVersionUsage: %w", err)
	}
	log.Info("Deleted activation stageVersionUsage", "stageVersionUsage", activationUsage)
	return nil
}

func activationUsageName(stage *konfidence.Stage, activation *konfidence.VectorActivation) string {
	return fmt.Sprintf("%s-%s-activation", stage.Name, activation.Name)
}
