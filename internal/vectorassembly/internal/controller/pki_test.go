package controller

import (
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	testocm "github.com/konfidence-project/konfidence/pkg/testutil/ocm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("PKI sign/verify scenarios", Ordered, Serial, func() {

	BeforeEach(func() {
		// Clean up any existing VectorTemplate CRs before each test.
		Expect(k8sClient.DeleteAllOf(ctx, &konfidence.VectorTemplate{}, client.InNamespace(testNamespace))).To(Succeed())
		Eventually(func(g Gomega) {
			list := &konfidence.VectorTemplateList{}
			g.Expect(k8sClient.List(ctx, list, client.InNamespace(testNamespace))).To(Succeed())
			g.Expect(list.Items).To(BeEmpty())
		}, timeout, interval).Should(Succeed())
	})

	It("should sign the vector and verify pre-signed artifacts (all PKI paths active)", func() {
		svc1 := createReference("konfidence.io/pki/s1/service1:v1.0.0")
		svc1Alias := createReference("konfidence.io/pki/s1/service1:stable")
		svc2 := createReference("konfidence.io/pki/s1/service2:v1.0.0")
		svc2Alias := createReference("konfidence.io/pki/s1/service2:stable")
		vectorTarget := createBareReference("konfidence.io/pki/vectors/s1")

		By("pushing artifact components pre-signed with artifactSigningKey out-of-band")
		testocm.PushSignedComponent(ctx, ocmClient, svc1, new("stable"),
			testocm.Bind(artifactSigName, artifactSigningKey))
		testocm.PushSignedComponent(ctx, ocmClient, svc2, new("stable"),
			testocm.Bind(artifactSigName, artifactSigningKey))

		By("creating a VectorTemplate CR with sign + verify-vector + verify-artifacts")
		vectorTemplate := createPKIVectorTemplateCR(
			ctx, "pki-all-active", testNamespace,
			[]compref.Ref{svc1Alias, svc2Alias},
			vectorTarget, "",
			pkiVectorTemplateOptions{
				credSecretNames: credSecretNames,
				signVector:      signSpec(vectorSigName),
				verifyVector:    verifySpec(vectorSigName),
				verifyArtifacts: verifySpec(artifactSigName),
			},
		)

		By("asserting VectorReady=True with VectorCreated reason")
		expectReadyCondition(vectorTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)

		By("fetching the emitted vector descriptor from Zot and asserting v-sig-A is present")
		latest := waitForLatestVector(vectorTemplate)
		descriptor, err := ocmClient.Get(ctx, latest)
		Expect(err).NotTo(HaveOccurred(), "failed to get vector descriptor from registry")
		Expect(descriptor.Signatures).NotTo(BeEmpty(), "vector descriptor should carry at least one signature")
		sigNames := make([]string, len(descriptor.Signatures))
		for i, s := range descriptor.Signatures {
			sigNames[i] = s.Name
		}
		Expect(sigNames).To(ContainElement(vectorSigName), "descriptor should have signature named %q", vectorSigName)
	})

	It("should succeed with no PKI specs (noop path)", func() {
		svc1 := createReference("konfidence.io/pki/s2/service1:v1.0.0")
		svc1Alias := createReference("konfidence.io/pki/s2/service1:latest")
		vectorTarget := createBareReference("konfidence.io/pki/vectors/s2")

		By("pushing a plain unsigned component")
		testocm.PushComponent(ctx, ocmClient, svc1, new("latest"))

		By("creating a VectorTemplate CR with nil Credentials, nil SignVector, nil VerifyVector, nil VerifyArtifacts")
		vectorTemplate := createPKIVectorTemplateCR(
			ctx, "pki-noop", testNamespace,
			[]compref.Ref{svc1Alias},
			vectorTarget, "",
			pkiVectorTemplateOptions{
				credSecretNames: []string{ociCredSecretName},
			},
		)

		By("asserting VectorReady=True")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)).To(Succeed())
			cond := meta.FindStatusCondition(vectorTemplate.Status.Conditions, konfidence.VectorTemplateReadyCondition)
			g.Expect(cond).NotTo(BeNil(), "Ready condition should be set")
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(cond.ObservedGeneration).To(Equal(vectorTemplate.Generation))
		}, timeout, interval).Should(Succeed())
	})

	It("should fail when artifacts are signed with the wrong signature name", func() {
		svc1 := createReference("konfidence.io/pki/s4/service1:v1.0.0")
		svc1Alias := createReference("konfidence.io/pki/s4/service1:prod")
		vectorTarget := createBareReference("konfidence.io/pki/vectors/s4")

		By("pushing an artifact signed only with vectorSigName (v-sig-A), not artifact-sig-B")
		testocm.PushSignedComponent(ctx, ocmClient, svc1, new("prod"),
			testocm.Bind(vectorSigName, vectorSigningKey))

		By("creating a VectorTemplate CR whose VerifyArtifacts expects artifact-sig-B")
		vectorTemplate := createPKIVectorTemplateCR(
			ctx, "pki-wrong-sig", testNamespace,
			[]compref.Ref{svc1Alias},
			vectorTarget, "",
			pkiVectorTemplateOptions{
				credSecretNames: credSecretNames,
				verifyArtifacts: verifySpec(artifactSigName),
			},
		)

		By("asserting VectorReady=False (artifact verification rejected)")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)).To(Succeed())
			cond := meta.FindStatusCondition(vectorTemplate.Status.Conditions, konfidence.VectorTemplateReadyCondition)
			g.Expect(cond).NotTo(BeNil(), "Ready condition should be set")
			g.Expect(cond.Status).NotTo(Equal(metav1.ConditionTrue))
			g.Expect(cond.ObservedGeneration).To(Equal(vectorTemplate.Generation))
		}, timeout, interval).Should(Succeed())
	})
})
