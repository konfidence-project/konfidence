package usage

//go:generate go run go.uber.org/mock/mockgen -destination ./mocks/mock_client.go -package=mocks sigs.k8s.io/controller-runtime/pkg/client Client

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func GetCurrentActiveUsage(ctx context.Context, c client.Client, stage *konfidence.Stage) (*konfidence.StageVersionUsage, error) {
	log := logf.FromContext(ctx)
	name := getName(stage)
	stageVersionUsage := &konfidence.StageVersionUsage{}

	err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: stage.Namespace}, stageVersionUsage)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("no matching active stage version usage found")
			return nil, nil
		}
		return nil, fmt.Errorf("unable to get current active stageVersionUsage: %w", err)
	}
	return stageVersionUsage, nil
}

func IsNewerThanCurrentActiveUsage(
	ctx context.Context,
	c client.Client,
	stageVersion *konfidence.StageVersion,
	activeStageVersionUsage *konfidence.StageVersionUsage,
) (bool, error) {
	activeStageVersion := &konfidence.StageVersion{}
	err := c.Get(ctx,
		types.NamespacedName{Name: activeStageVersionUsage.Spec.StageVersionRef.Name, Namespace: activeStageVersionUsage.Namespace},
		activeStageVersion,
	)
	if err != nil {
		return false, fmt.Errorf("referenced stageVersion %s does not exist: %w", activeStageVersionUsage.Spec.StageVersionRef, err)
	}
	// TODO: clarify what should happen in case of equal generation.
	// Proposal: equal generation should only lead to a reconcile if the activation status is not successful (or failed?)
	if stageVersion.Spec.StageGeneration < activeStageVersion.Spec.StageGeneration {
		return false, nil
	}
	return true, nil
}

func UpdateActiveUsage(ctx context.Context, c client.Client, stageVersionUsage *konfidence.StageVersionUsage, stageVersion *konfidence.StageVersion) error {
	log := logf.FromContext(ctx)

	stageVersionUsage.Spec.StageVersionRef = &konfidence.StageVersionReference{Name: stageVersion.Name}

	if err := c.Update(ctx, stageVersionUsage); err != nil {
		return fmt.Errorf("unable to update active stageVersionUsage: %w", err)
	}
	log.Info("Updated active stageVersionUsage to stageVersion", "stageVersion", stageVersion, "stageVersionUsage", stageVersionUsage)
	return nil
}

func CreateActiveUsage(
	ctx context.Context, c client.Client, stage *konfidence.Stage, stageVersion *konfidence.StageVersion,
) (*konfidence.StageVersionUsage, error) {
	log := logf.FromContext(ctx)
	name := getName(stage)
	newActiveStageVersionUsage := &konfidence.StageVersionUsage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: stage.Namespace,
			Labels:    map[string]string{ActiveStageVersion: stage.Name},
		},
		Spec: konfidence.StageVersionUsageSpec{
			Reason:          StageVersionUsageActiveType,
			StageVersionRef: &konfidence.StageVersionReference{Name: stageVersion.Name},
		},
	}
	if err := controllerutil.SetControllerReference(stage, newActiveStageVersionUsage, c.Scheme()); err != nil {
		return nil, fmt.Errorf("unable to set controller reference for stageVersionUsage: %w", err)
	}

	if err := c.Create(ctx, newActiveStageVersionUsage); err != nil {
		return nil, err
	}
	log.Info("Created new active stageVersionUsage", "stageVersionUsage", newActiveStageVersionUsage)
	return newActiveStageVersionUsage, nil
}

func CreateOrUpdateActiveUsage(
	ctx context.Context,
	c client.Client,
	activeStageVersionUsage *konfidence.StageVersionUsage,
	stage *konfidence.Stage,
	stageVersion *konfidence.StageVersion,
) error {
	if activeStageVersionUsage == nil {
		_, err := CreateActiveUsage(ctx, c, stage, stageVersion)
		if err != nil {
			return fmt.Errorf("failed to create active usage: %w", err)
		}
		return nil
	}
	if err := UpdateActiveUsage(ctx, c, activeStageVersionUsage, stageVersion); err != nil {
		return fmt.Errorf("failed to update active usage: %w", err)
	}

	return nil
}

func getName(stage *konfidence.Stage) string {
	return fmt.Sprintf("%s-active-usage", stage.Name)
}
