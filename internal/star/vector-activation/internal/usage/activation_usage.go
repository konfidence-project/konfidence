package usage

import (
	"context"
	"fmt"

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func CreateActivationUsage(ctx context.Context, c client.Client, stage *common.Stage, activation *landscape.VectorActivation) (*landscape.StageVersionUsage, error) {
	log := logf.FromContext(ctx)

	stageVersionUsage := &landscape.StageVersionUsage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("activation-usage-%s-%s", stage.Name, activation.Name),
			Namespace: stage.Namespace,
			Labels:    map[string]string{ActivationStageVersionUsage: stage.Name},
		},
		Spec: landscape.StageVersionUsageSpec{
			Reason:          StageVersionUsageActivationType,
			StageVersionRef: &landscape.StageVersionReference{Name: activation.Spec.StageVersion},
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

func DeleteActivationUsage(ctx context.Context, c client.Client, activationUsage *landscape.StageVersionUsage) error {
	log := logf.FromContext(ctx)
	if err := c.Delete(ctx, activationUsage); err != nil {
		return fmt.Errorf("unable to delete activation stageVersionUsage: %w", err)
	}
	log.Info("Deleted activation stageVersionUsage", "stageVersionUsage", activationUsage)
	return nil
}
