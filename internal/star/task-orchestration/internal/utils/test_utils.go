//nolint:staticcheck // ST1001: allow dot-import for test utils using Gomega
package utils

import (
	"context"

	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
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

func CleanupStageVersion(k8sClient client.Client, stageVersionName string, namespace string) {
	ctx := context.Background()
	stageVersion := GetStageVersion(ctx, k8sClient, stageVersionName, namespace, true)

	if stageVersion != nil {
		DeleteStageVersion(ctx, k8sClient, stageVersion)
	}
}

func ContainsReference(references []metav1.OwnerReference, name string, kind string) bool {
	for _, ref := range references {
		if ref.Kind == kind && ref.Name == name {
			return true
		}
	}

	return false
}

func CreateStageVersionUsage(ctx context.Context, k8sClient client.Client, name string, namespace string) {
	usage := &landscape.StageVersionUsage{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "landscape.konfidence.cloud/v1alpha1",
			Kind:       "StageVersionUsage",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: landscape.StageVersionUsageSpec{},
	}

	Expect(k8sClient.Create(ctx, usage)).To(Succeed())
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
