package controller

import (
	"context"
	"encoding/json"

	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/galaxy/stageconfiguration/internal/template"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateStageSync(
	ctx context.Context, k8sClient client.Client, name string, namespace string,
	stageConfigName string, targetNamespace string, stageName string, vectorName string,
) {
	stageConfiguration := galaxy.StageConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "galaxy.konfidence.cloud/v1alpha1",
			Kind:       galaxy.StageConfigurationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      stageConfigName,
			Namespace: namespace,
		},
		Spec: galaxy.StageConfigurationSpec{
			Name:            stageName,
			Vector:          vectorName,
			TargetNamespace: targetNamespace,
		},
	}

	stageTemplate := CreateStageTemplate(stageConfiguration, vectorName)
	stageTemplateJSON, err := json.Marshal(stageTemplate)
	Expect(err).ToNot(HaveOccurred())

	stageSync := &galaxy.StageSync{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "galaxy.konfidence.cloud/v1alpha1",
			Kind:       galaxy.StageSyncKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: targetNamespace,
		},
		Spec: galaxy.StageSyncSpec{
			StageTemplate: runtime.RawExtension{Raw: stageTemplateJSON},
		},
	}

	Expect(k8sClient.Create(ctx, stageSync)).To(Succeed())
}

func CreateNamespace(ctx context.Context, k8sClient client.Client, namespace string) {
	ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	Expect(k8sClient.Create(ctx, ns)).To(Succeed())
}

func CreateStageTemplate(stageConfiguration galaxy.StageConfiguration, vector string) template.StageTemplate {
	return template.StageTemplate{
		TypeMeta: metav1.TypeMeta{
			Kind:       star.StageKind,
			APIVersion: "star.konfidence.cloud/v1alpha1",
		},
		Metadata: template.NamespacedName{
			Name:      stageConfiguration.Spec.Name,
			Namespace: stageConfiguration.Spec.TargetNamespace,
		},
		Spec: star.StageSpec{
			Vector: vector,
		},
	}
}

func CleanupResources(ctx context.Context, k8sClient client.Client, namespace string, targetNamespace string) {
	err := k8sClient.DeleteAllOf(ctx, &galaxy.StageConfiguration{}, client.InNamespace(namespace))
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.DeleteAllOf(ctx, &galaxy.StageSync{}, client.InNamespace(namespace))
	if !meta.IsNoMatchError(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	err = k8sClient.DeleteAllOf(ctx, &galaxy.StageSync{}, client.InNamespace(targetNamespace))
	if !meta.IsNoMatchError(err) {
		Expect(err).ToNot(HaveOccurred())
	}
}
