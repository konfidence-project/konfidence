//nolint:staticcheck // ST1001: allow dot-import for test utils using Gomega
package utils

import (
	"context"

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateStageConfiguration(ctx context.Context, k8sClient client.Client, name string, namespace string,
	targetNamespace string, stageName string, vector string) {
	stageConfiguration := &global.StageConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "global.konfidence.cloud/v1alpha1",
			Kind:       global.StageConfigurationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: global.StageConfigurationSpec{
			Name:            stageName,
			Vector:          vector,
			TargetNamespace: targetNamespace,
		},
	}

	Expect(k8sClient.Create(ctx, stageConfiguration)).To(Succeed())
}

func GetStageConfiguration(ctx context.Context, k8sClient client.Client,
	name string, namespace string, opt bool) *global.StageConfiguration {
	stageConfiguration := &global.StageConfiguration{}
	stageConfigurationLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, stageConfigurationLookupKey, stageConfiguration)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch stageConfiguration: %s", name)
	return stageConfiguration
}

func CreateStage(ctx context.Context, k8sClient client.Client, name string, namespace string, vectorName string) {
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

func CreateNamespace(ctx context.Context, k8sClient client.Client, namespace string) {
	ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	Expect(k8sClient.Create(ctx, ns)).To(Succeed())
}

func CleanupResources(ctx context.Context, k8sClient client.Client, namespace string, targetNamespace string) {
	err := k8sClient.DeleteAllOf(ctx, &global.StageConfiguration{}, client.InNamespace(namespace))
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.DeleteAllOf(ctx, &common.Stage{}, client.InNamespace(namespace))
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.DeleteAllOf(ctx, &common.Stage{}, client.InNamespace(targetNamespace))
	Expect(err).ToNot(HaveOccurred())
}
