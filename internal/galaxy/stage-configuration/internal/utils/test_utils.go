//nolint:staticcheck // ST1001: allow dot-import for test utils using Gomega
package utils

import (
	"context"

	global "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	landscape "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/galaxy/stage-configuration/pkg/template"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
	. "github.com/onsi/gomega"
)

func CreateStageConfiguration(ctx context.Context, k8sClient client.Client, name string, namespace string,
	targetNamespace string, stageName string, vector string) {
	stageConfiguration := &global.StageConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "galaxy.konfidence.cloud/v1alpha1",
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

func CreateStageSync(ctx context.Context, k8sClient client.Client, name string, namespace string, stageConfigName string, targetNamespace string, stageName string, vectorName string) {
	stageConfiguration := global.StageConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "galaxy.konfidence.cloud/v1alpha1",
			Kind:       global.StageConfigurationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      stageConfigName,
			Namespace: namespace,
		},
		Spec: global.StageConfigurationSpec{
			Name:            stageName,
			Vector:          vectorName,
			TargetNamespace: targetNamespace,
		},
	}

	stageTemplate := CreateStageTemplate(stageConfiguration, vectorName)
	stageTemplateJSON, err := json.Marshal(stageTemplate)
	Expect(err).ToNot(HaveOccurred())

	stageSync := &global.StageSync{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "galaxy.konfidence.cloud/v1alpha1",
			Kind:       global.StageSyncKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: targetNamespace,
		},
		Spec: global.StageSyncSpec{
			StageTemplate: runtime.RawExtension{Raw: stageTemplateJSON},
		},
	}

	Expect(k8sClient.Create(ctx, stageSync)).To(Succeed())
}

func CreateNamespace(ctx context.Context, k8sClient client.Client, namespace string) {
	ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	Expect(k8sClient.Create(ctx, ns)).To(Succeed())
}

func CreateStageTemplate(stageConfiguration global.StageConfiguration, vector string) template.StageTemplate {
	return template.StageTemplate{
		TypeMeta: metav1.TypeMeta{
			Kind:       landscape.StageKind,
			APIVersion: "star.konfidence.cloud/v1alpha1",
		},
		Metadata: template.NamespacedName{
			Name:      stageConfiguration.Spec.Name,
			Namespace: stageConfiguration.Spec.TargetNamespace,
		},
		Spec: landscape.StageSpec{
			Vector: vector,
		},
	}
}

func CleanupResources(ctx context.Context, k8sClient client.Client, namespace string, targetNamespace string) {
	err := k8sClient.DeleteAllOf(ctx, &global.StageConfiguration{}, client.InNamespace(namespace))
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.DeleteAllOf(ctx, &global.StageSync{}, client.InNamespace(namespace))
	if !meta.IsNoMatchError(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	err = k8sClient.DeleteAllOf(ctx, &global.StageSync{}, client.InNamespace(targetNamespace))
	if !meta.IsNoMatchError(err) {
		Expect(err).ToNot(HaveOccurred())
	}
}
