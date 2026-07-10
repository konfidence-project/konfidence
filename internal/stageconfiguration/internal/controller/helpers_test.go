package controller

import (
	"context"
	"encoding/json"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
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

	stageTemplate := createStageTemplate(stageConfiguration, vectorName)
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

func createNamespace(ctx context.Context, k8sClient client.Client, namespace string) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	Expect(k8sClient.Create(ctx, ns)).To(Succeed())
}

func createStageTemplate(stageConfiguration galaxy.StageConfiguration, vector string) template.StageTemplate {
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

//nolint:unparam // namespace and targetNamespace are the same in every call, keep as params for consistency
func cleanupResources(ctx context.Context, k8sClient client.Client, namespace string, targetNamespace string) {
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

// createStageConfiguration creates a StageConfiguration with the suite's default credentials and no verification.
func createStageConfiguration(ctx context.Context, name, stageName, vector string) {
	createPKIStageConfiguration(ctx, name, stageName, vector, scCredentials(credSecretNames), nil)
}

// createPKIStageConfiguration creates a StageConfiguration with optional credentials and vector verification.
// The target namespace is always "target" (the namespace created in BeforeSuite).
func createPKIStageConfiguration(
	ctx context.Context,
	name, stageName, vector string,
	creds *galaxy.Credentials,
	verify *galaxy.Verify,
) {
	sc := &galaxy.StageConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "galaxy.konfidence.cloud/v1alpha1",
			Kind:       galaxy.StageConfigurationKind,
		},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: galaxy.StageConfigurationSpec{
			Name:            stageName,
			Vector:          vector,
			TargetNamespace: "target",
			Credentials:     creds,
			VerifyVector:    verify,
		},
	}
	ExpectWithOffset(1, k8sClient.Create(ctx, sc)).To(Succeed())
}
