//nolint:staticcheck // ST1001: allow dot-import for test utils using Gomega
package controller

import (
	"context"
	"fmt"

	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateVectorMigration(ctx context.Context, k8sClient client.Client, name string, namespace string, stageVersionName string, vectorName string) {
	vectorMigration := &star.VectorMigration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: star.GroupVersion.String(),
			Kind:       "VectorMigration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: star.VectorMigrationSpec{
			StageVersion: stageVersionName,
			Vector:       vectorName,
		},
	}

	Expect(k8sClient.Create(ctx, vectorMigration)).To(Succeed())
}

func GetVectorMigration(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *star.VectorMigration {
	vectorMigration := &star.VectorMigration{}
	vectorMigrationLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch vectorMigration: %s", name)
	return vectorMigration
}

func DeleteVectorMigration(ctx context.Context, k8sClient client.Client, vectorMigration *star.VectorMigration) {
	err := k8sClient.Delete(ctx, vectorMigration)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete vectorMigration: %s", vectorMigration.Name)
}

func CleanupVectorMigration(k8sClient client.Client, vectorMigrationName string, namespace string) {
	ctx := context.Background()
	vectorMigration := GetVectorMigration(ctx, k8sClient, vectorMigrationName, namespace, true)

	if vectorMigration != nil {
		DeleteVectorMigration(ctx, k8sClient, vectorMigration)
	}
}

func CreateStageVersion(ctx context.Context, k8sClient client.Client, name string, namespace string, vectorName string, stageName string) {
	stageVersion := &star.StageVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: star.GroupVersion.String(),
			Kind:       "StageVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: star.StageVersionSpec{
			Vector:          vectorName,
			StageGeneration: 1,
			StageRef: &star.StageReference{
				Name: stageName,
			},
		},
	}

	Expect(k8sClient.Create(ctx, stageVersion)).To(Succeed())
}

func GetStageVersion(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *star.StageVersion {
	stageVersion := &star.StageVersion{}
	stageVersionLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch stageVersion: %s", name)
	return stageVersion
}

func DeleteStageVersion(ctx context.Context, k8sClient client.Client, stageVersion *star.StageVersion) {
	err := k8sClient.Delete(ctx, stageVersion)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete stageVersion: %s", stageVersion.Name)
}

func UpdateStageVersion(ctx context.Context, k8sClient client.Client, stageVersion *star.StageVersion) {
	err := k8sClient.Update(ctx, stageVersion)
	Expect(err).ToNot(HaveOccurred(), "Failed to update stageVersion: %s", stageVersion.Name)
}

func CleanupStageVersion(k8sClient client.Client, stageVersionName string, namespace string) {
	ctx := context.Background()
	stageVersion := GetStageVersion(ctx, k8sClient, stageVersionName, namespace, true)

	if stageVersion != nil {
		DeleteStageVersion(ctx, k8sClient, stageVersion)
	}
}

func GetStageVersionUsage(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *star.StageVersionUsage {
	stageVersionUsage := &star.StageVersionUsage{}
	stageVersionUsageLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch stageVersionUsage: %s", name)
	return stageVersionUsage
}

func DeleteStageVersionUsage(ctx context.Context, k8sClient client.Client, stageVersionUsage *star.StageVersionUsage) {
	err := k8sClient.Delete(ctx, stageVersionUsage)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete stageVersionUsage: %s", stageVersionUsage.Name)
}

func CleanupStageVersionUsage(k8sClient client.Client, stageVersionUsageName string, namespace string) {
	ctx := context.Background()
	stageVersionUsage := GetStageVersionUsage(ctx, k8sClient, stageVersionUsageName, namespace, true)

	if stageVersionUsage != nil {
		DeleteStageVersionUsage(ctx, k8sClient, stageVersionUsage)
	}
}

func CreateArtifactDeployment(ctx context.Context, k8sClient client.Client, name string, namespace string, taskManifest []star.TaskManifest) {
	artifactDeployment := &star.ArtifactDeployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: star.GroupVersion.String(),
			Kind:       "ArtifactDeployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: star.ArtifactDeploymentSpec{
			Manifest: star.ArtifactManifest{
				Type:       "image",
				AllowReuse: true,
			},
			TaskManifests: taskManifest,
			Component: star.OCMComponent{
				Name: "service",
			},
		},
	}

	Expect(k8sClient.Create(ctx, artifactDeployment)).To(Succeed())
}

func GetArtifactDeployment(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *star.ArtifactDeployment {
	artifactDeployment := &star.ArtifactDeployment{}
	artifactDeploymentLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, artifactDeploymentLookupKey, artifactDeployment)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch artifactDeployment: %s", name)
	return artifactDeployment
}

func DeleteArtifactDeployment(ctx context.Context, k8sClient client.Client, artifactDeployment *star.ArtifactDeployment) {
	err := k8sClient.Delete(ctx, artifactDeployment)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete artifactDeployment: %s", artifactDeployment.Name)
}

func CleanupArtifactDeployment(k8sClient client.Client, artifactDeploymentName string, namespace string) {
	ctx := context.Background()
	artifactDeployment := GetArtifactDeployment(ctx, k8sClient, artifactDeploymentName, namespace, true)

	if artifactDeployment != nil {
		DeleteArtifactDeployment(ctx, k8sClient, artifactDeployment)
	}
}

func CreateVectorDeployment(ctx context.Context, k8sClient client.Client, name string, namespace string, vector string, stageVersion string) {
	vectorDeployment := &star.VectorDeployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: star.GroupVersion.String(),
			Kind:       "VectorDeployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				pkgctrl.StageVersionNameLabel: stageVersion,
			},
		},
		Spec: star.VectorDeploymentSpec{
			Vector: vector,
		},
	}

	Expect(k8sClient.Create(ctx, vectorDeployment)).To(Succeed())
}

func GetVectorDeployment(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *star.VectorDeployment {
	vectorDeployment := &star.VectorDeployment{}
	vectorDeploymentLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch vectorDeployment: %s", name)
	return vectorDeployment
}

func DeleteVectorDeployment(ctx context.Context, k8sClient client.Client, vectorDeployment *star.VectorDeployment) {
	err := k8sClient.Delete(ctx, vectorDeployment)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete vectorDeployment: %s", vectorDeployment.Name)
}

func CleanupVectorDeployment(k8sClient client.Client, vectorDeploymentName string, namespace string) {
	ctx := context.Background()
	vectorDeployment := GetVectorDeployment(ctx, k8sClient, vectorDeploymentName, namespace, true)

	if vectorDeployment != nil {
		DeleteVectorDeployment(ctx, k8sClient, vectorDeployment)
	}
}

func UpdateVectorDeploymentStatus(ctx context.Context, k8sClient client.Client, vectorDeployment *star.VectorDeployment) {
	err := k8sClient.Status().Update(ctx, vectorDeployment)
	Expect(err).ToNot(HaveOccurred(), "Failed to update status of vectorDeployment: %s", vectorDeployment.Name)
}

func SetTaskExecutionStatus(ctx context.Context, k8sClient client.Client, taskExecution *star.TaskExecution, status string) {
	meta.SetStatusCondition(&taskExecution.Status.Conditions, metav1.Condition{
		Type:               status,
		Status:             metav1.ConditionTrue,
		Reason:             status,
		Message:            fmt.Sprintf("Successfully executed Task %s", taskExecution.Name),
		ObservedGeneration: taskExecution.Generation,
		LastTransitionTime: metav1.Now(),
	})

	err := k8sClient.Status().Update(ctx, taskExecution)
	Expect(err).ToNot(HaveOccurred(), "Failed to update status of taskExecution: %s", taskExecution.Name)
}

func CreateStage(ctx context.Context, k8sClient client.Client, name string, namespace string, specName string, vectorName string) {
	stage := &star.Stage{
		TypeMeta: metav1.TypeMeta{
			APIVersion: star.GroupVersion.String(),
			Kind:       "Stage",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: star.StageSpec{
			Vector: vectorName,
		},
	}

	Expect(k8sClient.Create(ctx, stage)).To(Succeed())
}

func GetStage(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *star.Stage {
	stage := &star.Stage{}
	stageLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, stageLookupKey, stage)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch stage: %s", name)
	return stage
}

func DeleteStage(ctx context.Context, k8sClient client.Client, stage *star.Stage) {
	err := k8sClient.Delete(ctx, stage)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete stage: %s", stage.Name)
}

func CleanupStage(k8sClient client.Client, stageName string, namespace string) {
	ctx := context.Background()
	stage := GetStage(ctx, k8sClient, stageName, namespace, true)

	if stage != nil {
		DeleteStage(ctx, k8sClient, stage)
	}
}

func HasOwnerReference(ownerReferences []metav1.OwnerReference, ref metav1.OwnerReference) bool {
	for _, ownerReference := range ownerReferences {
		if ownerReference.Kind == ref.Kind && ownerReference.Name == ref.Name {
			return true
		}
	}

	return false
}

func GetTaskExecutionWithTaskName(name string, items []star.TaskExecution) *star.TaskExecution {
	for _, exec := range items {
		if name == exec.Spec.Name {
			return &exec
		}
	}

	return nil
}

func DeleteTaskExecution(ctx context.Context, k8sClient client.Client, taskExecution *star.TaskExecution) {
	err := k8sClient.Delete(ctx, taskExecution)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete taskExecution: %s", taskExecution.Name)
}

func GetTaskExecutions(ctx context.Context, k8sClient client.Client) *star.TaskExecutionList {
	taskExecutions := &star.TaskExecutionList{}
	err := k8sClient.List(ctx, taskExecutions)

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch taskExecutions")
	return taskExecutions
}
func CleanupTaskExecutions(k8sClient client.Client) {
	ctx := context.Background()
	taskExecutions := GetTaskExecutions(ctx, k8sClient)

	for _, taskExecution := range taskExecutions.Items {
		DeleteTaskExecution(ctx, k8sClient, &taskExecution)
	}
}
