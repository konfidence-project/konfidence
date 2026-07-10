package controller

import (
	"context"
	"fmt"
	"time"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"github.com/konfidence-project/konfidence/pkg/testutil/ocm"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// sourceRef creates a compref.Ref pointing to the source registry.
func sourceRef(component string) compref.Ref {
	return ocm.ParseRef(sourceRegistryEndpoint, component)
}

// targetRef creates a compref.Ref pointing to the target registry.
func targetRef(component string) compref.Ref {
	return ocm.ParseRef(targetRegistryEndpoint, component)
}

// sourceRefWithSubPath creates a compref.Ref for the source registry with a sub path.
func sourceRefWithSubPath(subPath, component string) compref.Ref {
	return ocm.ParseRef(fmt.Sprintf("%s/%s", sourceRegistryEndpoint, subPath), component)
}

// cleanupPromotions deletes all VectorPromotion and VectorPromotionConfig objects and waits for them to be gone.
func cleanupPromotions() {
	Expect(k8sClient.DeleteAllOf(ctx, &galaxy.VectorPromotion{}, client.InNamespace(testNamespace))).To(Succeed())
	Expect(k8sClient.DeleteAllOf(ctx, &galaxy.VectorPromotionConfig{}, client.InNamespace(testNamespace))).To(Succeed())
	Eventually(func(g Gomega) {
		promotions := &galaxy.VectorPromotionList{}
		g.Expect(k8sClient.List(ctx, promotions, client.InNamespace(testNamespace))).To(Succeed())
		g.Expect(promotions.Items).To(BeEmpty())
		configs := &galaxy.VectorPromotionConfigList{}
		g.Expect(k8sClient.List(ctx, configs, client.InNamespace(testNamespace))).To(Succeed())
		g.Expect(configs.Items).To(BeEmpty())
	}, timeout, interval).Should(Succeed())
}

// pushComponent pushes a minimal OCM component to the source registry.
func pushComponent(ctx context.Context, ref compref.Ref, alias *string) {
	ocm.PushComponent(ctx, ocmClient, ref, alias)
}

func createConfig(name, source, target string) *galaxy.VectorPromotionConfig {
	refs := make([]galaxy.CredentialRef, len(credSecretNames))
	for i, n := range credSecretNames {
		refs[i] = galaxy.CredentialRef{Name: n}
	}
	return createConfigWithCredentials(name, source, target, &galaxy.Credentials{
		OCM: &galaxy.OCMCredentials{Refs: refs},
	})
}

// createConfigWithCredentials creates a VectorPromotionConfig with credentials in the test namespace.
func createConfigWithCredentials(
	name, source, target string,
	creds *galaxy.Credentials,
) *galaxy.VectorPromotionConfig {
	return createPKIConfig(name, source, target, creds, nil)
}

// createPKIConfig creates a VectorPromotionConfig with credentials and optional verification in the test namespace.
func createPKIConfig(
	name, source, target string,
	creds *galaxy.Credentials,
	verify *galaxy.Verify,
) *galaxy.VectorPromotionConfig {
	config := &galaxy.VectorPromotionConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: galaxy.VectorPromotionConfigSpec{
			Source:       source,
			Target:       target,
			Credentials:  creds,
			VerifyVector: verify,
		},
	}
	ExpectWithOffset(1, k8sClient.Create(ctx, config)).To(Succeed())
	return config
}

// createPromotion creates a VectorPromotion in the test namespace referencing a config.
func createPromotion(name, configRef string) *galaxy.VectorPromotion {
	promotion := &galaxy.VectorPromotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: galaxy.VectorPromotionSpec{
			VectorPromotionConfigRef: configRef,
		},
	}
	ExpectWithOffset(1, k8sClient.Create(ctx, promotion)).To(Succeed())
	return promotion
}

// createPromotionWithTTL creates a VectorPromotion with TTLAfterFinished set.
func createPromotionWithTTL(name, configRef string, ttl time.Duration) *galaxy.VectorPromotion {
	promotion := &galaxy.VectorPromotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: galaxy.VectorPromotionSpec{
			VectorPromotionConfigRef: configRef,
			TTLAfterFinished:         &metav1.Duration{Duration: ttl},
		},
	}
	ExpectWithOffset(1, k8sClient.Create(ctx, promotion)).To(Succeed())
	return promotion
}
