//nolint:staticcheck // ST1001: allow dot-import for test utils using Gomega
package utils

import (
	"context"
	"fmt"

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateVectorMigration(ctx context.Context, k8sClient client.Client, name string, namespace string, stageVersionName string, vectorName string) {
	vectorMigration := &landscape.VectorMigration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "common.konfidence.cloud/v1alpha1",
			Kind:       "VectorMigration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: landscape.VectorMigrationSpec{
			StageVersion: stageVersionName,
			Vector:       vectorName,
		},
	}

	Expect(k8sClient.Create(ctx, vectorMigration)).To(Succeed())
}

func GetVectorMigration(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *landscape.VectorMigration {
	vectorMigration := &landscape.VectorMigration{}
	vectorMigrationLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch vectorMigration: %s", name)
	return vectorMigration
}

func DeleteVectorMigration(ctx context.Context, k8sClient client.Client, vectorMigration *landscape.VectorMigration) {
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

func CreateStageVersion(ctx context.Context, k8sClient client.Client, name string, namespace string, vectorName string) {
	stageVersion := &landscape.StageVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "landscape.konfidence.cloud/v1alpha1",
			Kind:       "StageVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: landscape.StageVersionSpec{
			Vector:          vectorName,
			StageGeneration: 1,
		},
	}

	Expect(k8sClient.Create(ctx, stageVersion)).To(Succeed())
}

func GetStageVersion(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *landscape.StageVersion {
	stageVersion := &landscape.StageVersion{}
	stageVersionLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch stageVersion: %s", name)
	return stageVersion
}

func DeleteStageVersion(ctx context.Context, k8sClient client.Client, stageVersion *landscape.StageVersion) {
	err := k8sClient.Delete(ctx, stageVersion)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete stageVersion: %s", stageVersion.Name)
}

func UpdateStageVersion(ctx context.Context, k8sClient client.Client, stageVersion *landscape.StageVersion) {
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

func GetStageVersionUsage(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *landscape.StageVersionUsage {
	stageVersionUsage := &landscape.StageVersionUsage{}
	stageVersionUsageLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch stageVersionUsage: %s", name)
	return stageVersionUsage
}

func DeleteStageVersionUsage(ctx context.Context, k8sClient client.Client, stageVersionUsage *landscape.StageVersionUsage) {
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

func CreateArtifactDeployment(ctx context.Context, k8sClient client.Client, name string, namespace string, taskManifest []landscape.TaskManifest) {
	artifactDeployment := &landscape.ArtifactDeployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "common.konfidence.cloud/v1alpha1",
			Kind:       "ArtifactDeployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: landscape.ArtifactDeploymentSpec{
			Manifest: landscape.ArtifactManifest{
				Type:       "image",
				AllowReuse: true,
			},
			TaskManifests: taskManifest,
			Component: landscape.OCMComponent{
				Name: "service",
			},
		},
	}

	Expect(k8sClient.Create(ctx, artifactDeployment)).To(Succeed())
}

func GetArtifactDeployment(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *landscape.ArtifactDeployment {
	artifactDeployment := &landscape.ArtifactDeployment{}
	artifactDeploymentLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, artifactDeploymentLookupKey, artifactDeployment)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch artifactDeployment: %s", name)
	return artifactDeployment
}

func DeleteArtifactDeployment(ctx context.Context, k8sClient client.Client, artifactDeployment *landscape.ArtifactDeployment) {
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

func CreateVectorDeployment(ctx context.Context, k8sClient client.Client, name string, namespace string, vector string) {
	vectorDeployment := &landscape.VectorDeployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "common.konfidence.cloud/v1alpha1",
			Kind:       "VectorDeployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: landscape.VectorDeploymentSpec{
			Vector: vector,
		},
	}

	Expect(k8sClient.Create(ctx, vectorDeployment)).To(Succeed())
}

func GetVectorDeployment(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *landscape.VectorDeployment {
	vectorDeployment := &landscape.VectorDeployment{}
	vectorDeploymentLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch vectorDeployment: %s", name)
	return vectorDeployment
}

func DeleteVectorDeployment(ctx context.Context, k8sClient client.Client, vectorDeployment *landscape.VectorDeployment) {
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

func UpdateVectorDeploymentStatus(ctx context.Context, k8sClient client.Client, vectorDeployment *landscape.VectorDeployment) {
	err := k8sClient.Status().Update(ctx, vectorDeployment)
	Expect(err).ToNot(HaveOccurred(), "Failed to update status of vectorDeployment: %s", vectorDeployment.Name)
}

func SetTaskExecutionStatus(ctx context.Context, k8sClient client.Client, taskExecution *landscape.TaskExecution, status string) {
	meta.SetStatusCondition(&taskExecution.Status.Conditions, metav1.Condition{Type: status,
		Status: metav1.ConditionTrue, Reason: status,
		Message: fmt.Sprintf("Successfully executed Task %s", taskExecution.Name)})

	err := k8sClient.Status().Update(ctx, taskExecution)
	Expect(err).ToNot(HaveOccurred(), "Failed to update status of taskExecution: %s", taskExecution.Name)
}

func CreateStage(ctx context.Context, k8sClient client.Client, name string, namespace string, specName string, vectorName string) {
	stage := &common.Stage{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "common.konfidence.cloud/v1alpha1",
			Kind:       "Stage",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: common.StageSpec{
			Name:   specName,
			Vector: vectorName,
		},
	}

	Expect(k8sClient.Create(ctx, stage)).To(Succeed())
}

func GetStage(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *common.Stage {
	stage := &common.Stage{}
	stageLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, stageLookupKey, stage)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch stage: %s", name)
	return stage
}

func DeleteStage(ctx context.Context, k8sClient client.Client, stage *common.Stage) {
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
