package controller

import (
	"context"
	"fmt"
	"time"

	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	testutilOcm "github.com/konfidence-project/pkg/testutil/ocm"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"ocm.software/open-component-model/bindings/go/oci/compref"
)

// sourceRef creates a compref.Ref pointing to the source registry.
func sourceRef(component string) compref.Ref {
	return testutilOcm.ParseRef(sourceRegistryEndpoint, component)
}

// targetRef creates a compref.Ref pointing to the target registry.
func targetRef(component string) compref.Ref {
	return testutilOcm.ParseRef(targetRegistryEndpoint, component)
}

// sourceRefWithSubPath creates a compref.Ref for the source registry with a sub path.
func sourceRefWithSubPath(subPath, component string) compref.Ref {
	return testutilOcm.ParseRef(fmt.Sprintf("%s/%s", sourceRegistryEndpoint, subPath), component)
}

// pushComponent pushes a minimal OCM component to the registry.
func pushComponent(ctx context.Context, ref compref.Ref, alias *string) {
	testutilOcm.PushComponent(ctx, ocmClient, ref, alias)
}

// createConfig creates a VectorPromotionConfig in the test namespace.
func createConfig(name, source, target string) *global.VectorPromotionConfig {
	config := &global.VectorPromotionConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: global.VectorPromotionConfigSpec{
			Source: source,
			Target: target,
		},
	}
	ExpectWithOffset(1, k8sClient.Create(ctx, config)).To(Succeed())
	return config
}

// createPromotion creates a VectorPromotion in the test namespace referencing a config.
func createPromotion(name, configRef string) *global.VectorPromotion {
	promotion := &global.VectorPromotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: global.VectorPromotionSpec{
			VectorPromotionConfigRef: configRef,
		},
	}
	ExpectWithOffset(1, k8sClient.Create(ctx, promotion)).To(Succeed())
	return promotion
}

// createPromotionWithTTL creates a VectorPromotion with TTLAfterFinished set.
func createPromotionWithTTL(name, configRef string, ttl time.Duration) *global.VectorPromotion {
	promotion := &global.VectorPromotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: global.VectorPromotionSpec{
			VectorPromotionConfigRef: configRef,
			TTLAfterFinished:         &metav1.Duration{Duration: ttl},
		},
	}
	ExpectWithOffset(1, k8sClient.Create(ctx, promotion)).To(Succeed())
	return promotion
}
