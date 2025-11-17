package utils

import (
	"context"

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateVectorActivation(ctx context.Context, k8sClient client.Client, name string, namespace string, vectorName string, stageVersion string) {

	vectorActivation := &landscape.VectorActivation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "common.konfidence.cloud/v1alpha1",
			Kind:       "VectorActivation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: landscape.VectorActivationSpec{
			Vector:       vectorName,
			StageVersion: stageVersion,
		},
	}

	gomega.Expect(k8sClient.Create(ctx, vectorActivation)).To(gomega.Succeed())
}

func GetVectorActivation(ctx context.Context, k8sClient client.Client, name string, namespace string) *landscape.VectorActivation {
	vectorActivation := &landscape.VectorActivation{}
	objectKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, objectKey, vectorActivation)

	if err != nil && errors.IsNotFound(err) {
		return nil
	}

	gomega.Expect(err).ToNot(gomega.HaveOccurred(), "Failed to get VectorActivation: %s", name)
	return vectorActivation
}

func DeleteVectorActivation(ctx context.Context, k8sClient client.Client, vectorActivation *landscape.VectorActivation) {
	err := k8sClient.Delete(ctx, vectorActivation)
	gomega.Expect(err).ToNot(gomega.HaveOccurred(), "Failed to delete VectorActivation: %s", vectorActivation.Name)
}

func CleanupVectorActivation(k8sClient client.Client, vectorActivationName string, namespace string) {
	ctx := context.Background()
	vectorActivation := GetVectorActivation(ctx, k8sClient, vectorActivationName, namespace)

	if vectorActivation != nil {
		DeleteVectorActivation(ctx, k8sClient, vectorActivation)
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

	gomega.Expect(k8sClient.Create(ctx, stageVersion)).To(gomega.Succeed())
}

func GetStageVersion(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *landscape.StageVersion {
	stageVersion := &landscape.StageVersion{}
	stageVersionLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	gomega.Expect(err).ToNot(gomega.HaveOccurred(), "Failed to fetch stageVersion: %s", name)
	return stageVersion
}

func DeleteStageVersion(ctx context.Context, k8sClient client.Client, stageVersion *landscape.StageVersion) {
	err := k8sClient.Delete(ctx, stageVersion)
	gomega.Expect(err).ToNot(gomega.HaveOccurred(), "Failed to delete stageVersion: %s", stageVersion.Name)
}

func CleanupStageVersion(k8sClient client.Client, stageVersionName string, namespace string) {
	ctx := context.Background()
	stageVersion := GetStageVersion(ctx, k8sClient, stageVersionName, namespace, true)

	if stageVersion != nil {
		DeleteStageVersion(ctx, k8sClient, stageVersion)
	}
}

func GetActivationExecution(ctx context.Context, k8sClient client.Client, name string, namespace string) *landscape.ActivationExecution {
	activationExecution := &landscape.ActivationExecution{}
	activationExecutionLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, activationExecutionLookupKey, activationExecution)

	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	gomega.Expect(err).ToNot(gomega.HaveOccurred(), "Failed to fetch activation execution: %s", name)
	return activationExecution
}

func CleanupActivationExecution(k8sClient client.Client, executionName string, namespace string) {
	ctx := context.Background()
	activationExecution := GetActivationExecution(ctx, k8sClient, executionName, namespace)

	if activationExecution != nil {
		DeleteActivationExecution(ctx, k8sClient, activationExecution)
	}
}

func DeleteActivationExecution(ctx context.Context, k8sClient client.Client, activationExecution *landscape.ActivationExecution) {
	err := k8sClient.Delete(ctx, activationExecution)
	gomega.Expect(err).ToNot(gomega.HaveOccurred(), "Failed to delete activation execution: %s", activationExecution.Name)
}

func UpdateStageVersion(ctx context.Context, k8sClient client.Client, stageVersion *landscape.StageVersion) {
	err := k8sClient.Update(ctx, stageVersion)
	gomega.Expect(err).ToNot(gomega.HaveOccurred(), "Failed to update stageVersion: %s", stageVersion.Name)
}

func HasOwnerReference(ownerReferences []metav1.OwnerReference, ref metav1.OwnerReference) bool {
	for _, ownerReference := range ownerReferences {
		if ownerReference.Kind == ref.Kind && ownerReference.Name == ref.Name {
			return true
		}
	}
	return false
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

	gomega.Expect(k8sClient.Create(ctx, stage)).To(gomega.Succeed())
}

func GetStage(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *common.Stage {
	stage := &common.Stage{}
	stageLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, stageLookupKey, stage)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	gomega.Expect(err).ToNot(gomega.HaveOccurred(), "Failed to fetch stage: %s", name)
	return stage
}

func DeleteStage(ctx context.Context, k8sClient client.Client, stage *common.Stage) {
	err := k8sClient.Delete(ctx, stage)
	gomega.Expect(err).ToNot(gomega.HaveOccurred(), "Failed to delete stage: %s", stage.Name)
}

func CleanupStage(k8sClient client.Client, stageName string, namespace string) {
	ctx := context.Background()
	stage := GetStage(ctx, k8sClient, stageName, namespace, true)

	if stage != nil {
		DeleteStage(ctx, k8sClient, stage)
	}
}
