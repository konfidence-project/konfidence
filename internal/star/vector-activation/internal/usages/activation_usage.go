package usages

import (
	"context"
	"fmt"

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func CreateOrUpdateActivationUsage(ctx context.Context, c client.Client, stage *common.Stage, stageVersion *landscape.StageVersion, activation *landscape.VectorActivation) (*landscape.StageVersionUsage, error) {
	log := logf.FromContext(ctx)
	name := fmt.Sprintf("%s-activation-usage", stageVersion.Name)
	stageVersionUsage := &landscape.StageVersionUsage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: stage.Namespace,
			Labels:    map[string]string{ActivationStageVersionUsage: stage.Name},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: activation.APIVersion,
					Kind:       activation.Kind,
					Name:       activation.Name,
					UID:        activation.UID,
				},
			},
		},
		Spec: landscape.StageVersionUsageSpec{
			Reason:          StageVersionUsageActivationType,
			StageVersionRef: &landscape.StageVersionRef{Name: stageVersion.Name},
		},
	}
	if err := c.Create(ctx, stageVersionUsage); err != nil {
		if errors.IsAlreadyExists(err) {
			log.Info("activation usage already exists, update it")
			return UpdateActivationUsage(ctx, c, stageVersionUsage)
		}
		return nil, fmt.Errorf("failed to create activation stageVersionUsage: %w", err)
	}
	log.Info("Created activation stageVersionUsage", "stageVersionUsage", stageVersionUsage)
	return stageVersionUsage, nil
}

func UpdateActivationUsage(ctx context.Context, c client.Client, activationUsage *landscape.StageVersionUsage) (*landscape.StageVersionUsage, error) {
	log := logf.FromContext(ctx)
	if err := c.Update(ctx, activationUsage); err != nil {
		return nil, fmt.Errorf("unable to update activation stageVersionUsage: %w", err)
	}
	log.Info("Updated activation stageVersionUsage", "stageVersionUsage", activationUsage)
	return activationUsage, nil
}

func DeleteActivationUsage(ctx context.Context, c client.Client, activationUsage *landscape.StageVersionUsage) error {
	log := logf.FromContext(ctx)
	if err := c.Delete(ctx, activationUsage); err != nil {
		return fmt.Errorf("unable to delete activation stageVersionUsage: %w", err)
	}
	log.Info("Deleted activation stageVersionUsage", "stageVersionUsage", activationUsage)
	return nil
}
