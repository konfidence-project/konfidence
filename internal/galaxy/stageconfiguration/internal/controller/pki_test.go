package controller

import (
	"context"
	"fmt"
	"time"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"ocm.software/open-component-model/bindings/go/oci/compref"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	testocm "github.com/konfidence-project/konfidence/pkg/testutil/ocm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	pkiTimeout  = 30 * time.Second
	pkiInterval = 250 * time.Millisecond
)

func scCredentials(names []string) *galaxy.Credentials {
	refs := make([]galaxy.CredentialRef, len(names))
	for i, n := range names {
		refs[i] = galaxy.CredentialRef{Name: n}
	}
	return &galaxy.Credentials{OCM: &galaxy.OCMCredentials{Refs: refs}}
}

func scVerify(sigName string) *galaxy.Verify {
	return &galaxy.Verify{Signatures: []galaxy.Signature{{Name: sigName}}}
}

func scVectorRef(version string) compref.Ref {
	return testocm.ParseRef(registryEndpoint, fmt.Sprintf("konfidence.cloud/pki/sc-vector:%s", version))
}

func scVectorAlias(alias string) string {
	return fmt.Sprintf("http://%s//konfidence.cloud/pki/sc-vector:%s", registryEndpoint, alias)
}

func assertSCReady(ctx context.Context, name string, expectedStatus metav1.ConditionStatus) {
	sc := &galaxy.StageConfiguration{}
	key := types.NamespacedName{Name: name, Namespace: "default"}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, key, sc)).To(Succeed())
		cond := apimeta.FindStatusCondition(sc.Status.Conditions, galaxy.StageConfigurationReadyCondition)
		g.Expect(cond).NotTo(BeNil())
		g.Expect(cond.Status).To(Equal(expectedStatus))
	}, pkiTimeout, pkiInterval).Should(Succeed())
}

var _ = Describe("PKI sign/verify scenarios", Ordered, func() {
	BeforeEach(func() {
		cleanupResources(context.Background(), k8sClient, "default", "target")
	})

	AfterEach(func() {
		cleanupResources(context.Background(), k8sClient, "default", "target")
	})

	It("should mark ready when the vector is correctly signed and credentials match", func() {
		ctx := context.Background()
		ref := scVectorRef("1.0.0-sc1")
		artifactRef := testocm.ParseRef(registryEndpoint, "konfidence.cloud/pki/sc-artifact-1:0.0.1")

		By("pushing a signed vector to Zot")
		testocm.PushSignedVector(ctx, ocmClient, ref, []compref.Ref{artifactRef}, "sc1-latest",
			testocm.SampleVectorConfig(),
			testocm.Bind(vectorSigName, vectorSigningKey))

		createPKIStageConfiguration(ctx, "pki-sc-signed-ok", "pki-stage-1",
			scVectorAlias("sc1-latest"),
			scCredentials(credSecretNames), scVerify(vectorSigName))

		assertSCReady(ctx, "pki-sc-signed-ok", metav1.ConditionTrue)
	})

	It("should mark ready when verification is disabled and vector is unsigned", func() {
		ctx := context.Background()
		ref := scVectorRef("1.0.0-sc2")
		artifactRef := testocm.ParseRef(registryEndpoint, "konfidence.cloud/pki/sc-artifact-2:0.0.1")

		By("pushing an unsigned vector to Zot")
		testocm.PushVector(ctx, ocmClient, ref, []compref.Ref{artifactRef}, "sc2-latest", testocm.SampleVectorConfig())

		createPKIStageConfiguration(ctx, "pki-sc-noop-ok", "pki-stage-2",
			scVectorAlias("sc2-latest"), scCredentials([]string{ociCredSecretName}), nil)

		assertSCReady(ctx, "pki-sc-noop-ok", metav1.ConditionTrue)
	})

	It("should mark not ready when the expected signature name does not match the descriptor", func() {
		ctx := context.Background()
		ref := scVectorRef("1.0.0-sc3")
		artifactRef := testocm.ParseRef(registryEndpoint, "konfidence.cloud/pki/sc-artifact-3:0.0.1")

		By("pushing a vector signed under the canonical name")
		testocm.PushSignedVector(ctx, ocmClient, ref, []compref.Ref{artifactRef}, "sc3-latest",
			testocm.SampleVectorConfig(),
			testocm.Bind(vectorSigName, vectorSigningKey))

		createPKIStageConfiguration(ctx, "pki-sc-wrong-sig", "pki-stage-3",
			scVectorAlias("sc3-latest"),
			scCredentials(credSecretNames), scVerify("wrong-sig-name"))

		assertSCReady(ctx, "pki-sc-wrong-sig", metav1.ConditionFalse)
	})

	It("should mark not ready when the vector carries no signature but verification is enabled", func() {
		ctx := context.Background()
		ref := scVectorRef("1.0.0-sc5")
		artifactRef := testocm.ParseRef(registryEndpoint, "konfidence.cloud/pki/sc-artifact-5:0.0.1")

		By("pushing a plain unsigned vector to Zot")
		testocm.PushVector(ctx, ocmClient, ref, []compref.Ref{artifactRef}, "sc5-latest", testocm.SampleVectorConfig())

		createPKIStageConfiguration(ctx, "pki-sc-unsigned-fail", "pki-stage-5",
			scVectorAlias("sc5-latest"),
			scCredentials(credSecretNames), scVerify(vectorSigName))

		assertSCReady(ctx, "pki-sc-unsigned-fail", metav1.ConditionFalse)
	})
})
