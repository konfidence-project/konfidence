package controller

import (
	"context"
	"encoding/json"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/stageconfiguration/internal/template"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createStageSync(
	ctx context.Context, k8sClient client.Client, name string, namespace string,
	stageConfigName string, targetNamespace string, stageName string, vectorName string,
) {
	stageConfiguration := konfidence.StageConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "konfidence.cloud/v1alpha1",
			Kind:       konfidence.StageConfigurationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      stageConfigName,
			Namespace: namespace,
		},
		Spec: konfidence.StageConfigurationSpec{
			Name:            stageName,
			Vector:          vectorName,
			TargetNamespace: targetNamespace,
		},
	}

	stageTemplate := createStageTemplate(stageConfiguration, vectorName)
	stageTemplateJSON, err := json.Marshal(stageTemplate)
	Expect(err).ToNot(HaveOccurred())

	stageSync := &konfidence.StageSync{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "konfidence.cloud/v1alpha1",
			Kind:       konfidence.StageSyncKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: targetNamespace,
		},
		Spec: konfidence.StageSyncSpec{
			StageTemplate: runtime.RawExtension{Raw: stageTemplateJSON},
		},
	}

	Expect(k8sClient.Create(ctx, stageSync)).To(Succeed())
}

func createNamespace(ctx context.Context, k8sClient client.Client, namespace string) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	Expect(k8sClient.Create(ctx, ns)).To(Succeed())
}

func createStageTemplate(stageConfiguration konfidence.StageConfiguration, vector string) template.StageTemplate {
	return template.StageTemplate{
		TypeMeta: metav1.TypeMeta{
			Kind:       konfidence.StageKind,
			APIVersion: "konfidence.cloud/v1alpha1",
		},
		Metadata: template.NamespacedName{
			Name:      stageConfiguration.Spec.Name,
			Namespace: stageConfiguration.Spec.TargetNamespace,
		},
		Spec: konfidence.StageSpec{
			Vector: vector,
		},
	}
}

//nolint:unparam // namespace and targetNamespace are the same in every call, keep as params for consistency
func cleanupResources(ctx context.Context, k8sClient client.Client, namespace string, targetNamespace string) {
	err := k8sClient.DeleteAllOf(ctx, &konfidence.StageConfiguration{}, client.InNamespace(namespace))
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.DeleteAllOf(ctx, &konfidence.StageSync{}, client.InNamespace(namespace))
	if !meta.IsNoMatchError(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	err = k8sClient.DeleteAllOf(ctx, &konfidence.StageSync{}, client.InNamespace(targetNamespace))
	if !meta.IsNoMatchError(err) {
		Expect(err).ToNot(HaveOccurred())
	}
}

// createStageConfiguration creates a StageConfiguration with the suite's default credentials and no verification.
func createStageConfiguration(ctx context.Context, name, stageName, vector string) {
	createPKIStageConfiguration(ctx, name, stageName, vector, scCredentials(credSecretNames), nil)
}

// createPKIStageConfiguration creates a StageConfiguration with optional credentials and vector verification.
// The target namespace is always "target" (the namespace created in BeforeSuite).
func createPKIStageConfiguration(
	ctx context.Context,
	name, stageName, vector string,
	creds *konfidence.Credentials,
	verify *konfidence.Verify,
) {
	sc := &konfidence.StageConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "konfidence.cloud/v1alpha1",
			Kind:       konfidence.StageConfigurationKind,
		},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: konfidence.StageConfigurationSpec{
			Name:            stageName,
			Vector:          vector,
			TargetNamespace: "target",
			Credentials:     creds,
			VerifyVector:    verify,
		},
	}
	ExpectWithOffset(1, k8sClient.Create(ctx, sc)).To(Succeed())
}
