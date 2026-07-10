package controller

import (
	"fmt"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	testocm "github.com/konfidence-project/konfidence/pkg/testutil/ocm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"ocm.software/open-component-model/bindings/go/oci/compref"
)

const (
	badCredsSecretName = "bad-oci-credentials"
)

var _ = Describe("VectorPromotion PKI scenarios", Ordered, Serial, func() {

	BeforeEach(func() { cleanupPromotions() })

	It("should succeed when source vector is pre-signed with vectorSigningKey", func() {
		artifactRef := sourceRef("konfidence.io/pki/signed-artifact:v1.0.0")
		vectorRef := sourceRef("konfidence.io/pki/signed:v1.0.0")

		By("pushing artifact and pre-signed vector to source registry")
		testocm.PushComponent(ctx, ocmClient, artifactRef, nil)
		testocm.PushSignedVector(ctx, ocmClient, vectorRef, []compref.Ref{artifactRef}, "latest",
			testocm.SampleVectorConfig(),
			testocm.Bind(vectorSigName, vectorSigningKey),
		)

		By("creating VectorPromotionConfig with signing credentials and VerifyVector")
		source := fmt.Sprintf("http://%s//konfidence.io/pki/signed:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/pki/signed:promoted", targetRegistryEndpoint)
		config := createPKIConfig("pki-signed-config", source, target, &galaxy.Credentials{
			OCM: &galaxy.OCMCredentials{Refs: refsFromNames(credSecretNames...)},
		}, &galaxy.Verify{Signatures: []galaxy.Signature{{Name: vectorSigName}}})

		promotion := createPromotion("pki-signed-promotion", config.Name)

		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: promotion.Name, Namespace: testNamespace}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, galaxy.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(cond.Reason).To(Equal(galaxy.ReasonPromotionSucceeded))
		}, timeout, interval).Should(Succeed())
	})

	It("should fail with PromotionSourceVerificationFailed when source vector is unsigned", func() {
		artifactRef := sourceRef("konfidence.io/pki/unsigned-artifact:v1.0.0")
		vectorRef := sourceRef("konfidence.io/pki/unsigned:v1.0.0")

		By("pushing artifact and unsigned vector to source registry")
		testocm.PushComponent(ctx, ocmClient, artifactRef, nil)
		testocm.PushVector(ctx, ocmClient, vectorRef, []compref.Ref{artifactRef}, "latest", testocm.SampleVectorConfig())

		By("creating VectorPromotionConfig with VerifyVector enabled")
		source := fmt.Sprintf("http://%s//konfidence.io/pki/unsigned:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/pki/unsigned:promoted", targetRegistryEndpoint)
		config := createPKIConfig("pki-unsigned-config", source, target, &galaxy.Credentials{
			OCM: &galaxy.OCMCredentials{Refs: refsFromNames(credSecretNames...)},
		}, &galaxy.Verify{Signatures: []galaxy.Signature{{Name: vectorSigName}}})

		promotion := createPromotion("pki-unsigned-promotion", config.Name)

		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: promotion.Name, Namespace: testNamespace}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, galaxy.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(cond.Reason).To(Equal(galaxy.ReasonPromotionSourceVerificationFailed))
		}, timeout, interval).Should(Succeed())
	})

	It("should succeed when no verify is configured (plain copy)", func() {
		artifactRef := sourceRef("konfidence.io/pki/noop-artifact:v1.0.0")
		vectorRef := sourceRef("konfidence.io/pki/noop:v1.0.0")

		By("pushing artifact and unsigned vector to source registry")
		testocm.PushComponent(ctx, ocmClient, artifactRef, nil)
		testocm.PushVector(ctx, ocmClient, vectorRef, []compref.Ref{artifactRef}, "latest", testocm.SampleVectorConfig())

		source := fmt.Sprintf("http://%s//konfidence.io/pki/noop:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/pki/noop:promoted", targetRegistryEndpoint)
		config := createConfig("pki-noop-config", source, target)
		promotion := createPromotion("pki-noop-promotion", config.Name)

		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: promotion.Name, Namespace: testNamespace}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, galaxy.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(cond.Reason).To(Equal(galaxy.ReasonPromotionSucceeded))
		}, timeout, interval).Should(Succeed())
	})

	It("should fail with PromotionFailed when target registry credentials are wrong (bad password)", func() {
		By("creating a Secret with incorrect credentials for target registry")
		_ = k8sClient.Create(ctx, testocm.DockerConfigSecret(badCredsSecretName, testNamespace, "user", "wrong-password", targetRegistryEndpoint))

		artifactRef := sourceRef("konfidence.io/pki/badcreds-artifact:v1.0.0")
		vectorRef := sourceRef("konfidence.io/pki/badcreds:v1.0.0")

		By("pushing artifact and unsigned vector to source registry")
		testocm.PushComponent(ctx, ocmClient, artifactRef, nil)
		testocm.PushVector(ctx, ocmClient, vectorRef, []compref.Ref{artifactRef}, "latest", testocm.SampleVectorConfig())

		source := fmt.Sprintf("http://%s//konfidence.io/pki/badcreds:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/pki/badcreds:promoted", targetRegistryEndpoint)
		config := createConfigWithCredentials("pki-badcreds-config", source, target, &galaxy.Credentials{
			OCM: &galaxy.OCMCredentials{Refs: refsFromNames(sourceOnlyCredSecretName, badCredsSecretName)},
		})
		promotion := createPromotion("pki-badcreds-promotion", config.Name)

		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: promotion.Name, Namespace: testNamespace}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, galaxy.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(cond.Reason).To(Equal(galaxy.ReasonPromotionFailed))
		}, timeout, interval).Should(Succeed())
	})

})

// refsFromNames converts a list of Secret names into []galaxy.CredentialRef.
func refsFromNames(names ...string) []galaxy.CredentialRef {
	refs := make([]galaxy.CredentialRef, len(names))
	for i, n := range names {
		refs[i] = galaxy.CredentialRef{Name: n}
	}
	return refs
}
