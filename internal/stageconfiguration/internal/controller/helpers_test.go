package controller

import (
	"context"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createNamespace(ctx context.Context, k8sClient client.Client, namespace string) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	Expect(k8sClient.Create(ctx, ns)).To(Succeed())
}

//nolint:unparam // namespace and targetNamespace are the same in every call, keep as params for consistency
func cleanupResources(ctx context.Context, k8sClient client.Client, namespace string, targetNamespace string) {
	err := k8sClient.DeleteAllOf(ctx, &konfidence.StageConfiguration{}, client.InNamespace(namespace))
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.DeleteAllOf(ctx, &konfidence.Stage{}, client.InNamespace(namespace))
	if !meta.IsNoMatchError(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	err = k8sClient.DeleteAllOf(ctx, &konfidence.Stage{}, client.InNamespace(targetNamespace))
	if !meta.IsNoMatchError(err) {
		Expect(err).ToNot(HaveOccurred())
	}
}

// createStageConfiguration creates a StageConfiguration with the suite's default credentials and no verification.
func createStageConfiguration(ctx context.Context, name, stageName, vector string) {
	createPKIStageConfiguration(ctx, name, stageName, vector, scCredentials(credSecretNames), nil)
}

// updateStageConfigurationVector fetches the named StageConfiguration and updates its vector reference.
func updateStageConfigurationVector(ctx context.Context, name, vector string) {
	sc := &konfidence.StageConfiguration{}
	ExpectWithOffset(1, k8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: "default"}, sc)).To(Succeed())
	sc.Spec.Vector = vector
	ExpectWithOffset(1, k8sClient.Update(ctx, sc)).To(Succeed())
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
