package controller

import (
	"bytes"
	"fmt"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	ocm2 "github.com/konfidence-project/konfidence/internal/vectorassembly/internal/ocm"
	konfcompref "github.com/konfidence-project/konfidence/pkg/ocm/compref"
	"github.com/konfidence-project/konfidence/pkg/testutil/ocm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"ocm.software/open-component-model/bindings/go/blob"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNamespace = "default"
	timeout       = 30 * time.Second
	interval      = 250 * time.Millisecond
)

// expectReadyCondition polls the VectorTemplate until its Ready condition matches the
// given status and reason for the current generation.
func expectReadyCondition(vt *konfidence.VectorTemplate, status metav1.ConditionStatus, reason string) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vt), vt)).To(Succeed())
		cond := meta.FindStatusCondition(vt.Status.Conditions, konfidence.VectorTemplateReadyCondition)
		g.Expect(cond).NotTo(BeNil(), "Ready condition should be set")
		g.Expect(cond.Status).To(Equal(status))
		g.Expect(cond.Reason).To(Equal(reason))
		g.Expect(cond.ObservedGeneration).To(Equal(vt.Generation))
	}, timeout, interval).Should(Succeed())
}

// waitForLatestVector polls the VectorTemplate until status.latestVector is non-empty and
// returns the parsed concrete reference. Use this after the first assembly; to wait for a
// subsequent assembly to advance the version, use waitForLatestVectorChange.
func waitForLatestVector(vt *konfidence.VectorTemplate) compref.Ref {
	GinkgoHelper()
	var latest string
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vt), vt)).To(Succeed())
		g.Expect(vt.Status.LatestVector).NotTo(BeEmpty(), "status.latestVector should be populated")
		latest = vt.Status.LatestVector
	}, timeout, interval).Should(Succeed())

	return parseLatestVector(latest)
}

// parseLatestVector parses a status.latestVector value as a concrete versioned reference.
func parseLatestVector(latest string) compref.Ref {
	GinkgoHelper()
	ref, err := konfcompref.Parse(latest, konfcompref.WithVersionValidation(konfcompref.VersionValidationSemverOnly))
	Expect(err).NotTo(HaveOccurred(), "status.latestVector must be a concrete versioned reference")
	return *ref
}

// waitForLatestVectorChange polls until status.latestVector differs from previousRaw (a new
// assembly produced a new concrete version) and returns the parsed new reference. previousRaw
// is the raw status.latestVector string captured before the change, compared verbatim to
// avoid reference re-serialization mismatches.
func waitForLatestVectorChange(vt *konfidence.VectorTemplate, previousRaw string) compref.Ref {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vt), vt)).To(Succeed())
		g.Expect(vt.Status.LatestVector).NotTo(BeEmpty(), "status.latestVector should be populated")
		g.Expect(vt.Status.LatestVector).NotTo(Equal(previousRaw), "status.latestVector should advance to a new version")
	}, timeout, interval).Should(Succeed())

	return parseLatestVector(vt.Status.LatestVector)
}

var _ = Describe("VectorTemplate controller tests", Ordered, Serial, func() {

	BeforeEach(func() {
		// Clean up any existing VectorTemplate CRs before each test
		Expect(k8sClient.DeleteAllOf(ctx, &konfidence.VectorTemplate{}, client.InNamespace(testNamespace))).To(Succeed())
		// Wait until all VectorTemplates are gone
		Eventually(func(g Gomega) {
			list := &konfidence.VectorTemplateList{}
			g.Expect(k8sClient.List(ctx, list, client.InNamespace(testNamespace))).To(Succeed())
			g.Expect(list.Items).To(BeEmpty())
		}, timeout, interval).Should(Succeed())
	})

	It("should create a new vector when drift is detected against the recorded latest vector", func() {
		svc1v1 := createReference("konfidence.io/sample/drift/service1:v5.0.0")
		svc2v1 := createReference("konfidence.io/sample/drift/service2:v3.2.1")
		svc2v2 := createReference("konfidence.io/sample/drift/service2:v5.0.0")
		svc1Alias := createReference("konfidence.io/sample/drift/service1:edge")
		svc2Alias := createReference("konfidence.io/sample/drift/service2:edge")
		vectorTarget := createBareReference("konfidence.io/sample/vectors/drift-test")

		By("creating initial component descriptors reachable via the edge alias")
		ocm.PushComponent(ctx, ocmClient, svc1v1, new("edge"))
		ocm.PushComponent(ctx, ocmClient, svc2v1, new("edge"))

		By("phase 1: creating the VectorTemplate CR and letting it assemble the initial vector")
		vectorTemplate := createVectorTemplateCR(ctx, "drift-test", testNamespace, []compref.Ref{svc1Alias, svc2Alias}, vectorTarget, "", nil)
		expectReadyCondition(vectorTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)
		firstVector := waitForLatestVector(vectorTemplate)
		firstLatestRaw := vectorTemplate.Status.LatestVector

		firstDescriptor, err := ocmClient.Get(ctx, firstVector)
		Expect(err).NotTo(HaveOccurred())
		firstRefVersions := make(map[string]string, len(firstDescriptor.Component.References))
		for _, ref := range firstDescriptor.Component.References {
			firstRefVersions[ref.Component] = ref.Version
		}
		Expect(firstRefVersions).To(HaveKeyWithValue(svc2v1.Component, "v3.2.1"))

		By("inducing drift by moving service2's edge alias to a newer version")
		ocm.PushComponent(ctx, ocmClient, svc2v2, new("edge"))

		By("phase 2: nudging a reconcile and verifying a new vector is assembled")
		nudgeReconcile(vectorTemplate)
		expectReadyCondition(vectorTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)

		By("verifying status.latestVector advanced to a new concrete version")
		latest := waitForLatestVectorChange(vectorTemplate, firstLatestRaw)

		By("verifying the assembled vector in oci reflects the updated artifact version")
		descriptor, err := ocmClient.Get(ctx, latest)
		Expect(err).NotTo(HaveOccurred(), "failed to get vector descriptor from oci")
		Expect(descriptor.Component.References).To(HaveLen(2))
		Expect(descriptor.Component.Version).To(Equal(latest.Version), "descriptor version should match status.latestVector")

		refVersions := make(map[string]string, len(descriptor.Component.References))
		for _, ref := range descriptor.Component.References {
			refVersions[ref.Component] = ref.Version
		}
		Expect(refVersions).To(HaveKeyWithValue(svc1v1.Component, "v5.0.0"))
		Expect(refVersions).To(HaveKeyWithValue(svc2v2.Component, "v5.0.0"))
	})

	It("should report NoDriftDetected when nothing changed since the last assembly", func() {
		svc1 := createReference("konfidence.io/sample/nodrift/service1:1.0.0")
		svc1Alias := createReference("konfidence.io/sample/nodrift/service1:stable")
		svc2 := createReference("konfidence.io/sample/nodrift/service2:2.0.0")
		svc2Alias := createReference("konfidence.io/sample/nodrift/service2:stable")
		vectorTarget := createBareReference("konfidence.io/sample/vectors/nodrift-test")

		By("creating mock component descriptors in OCI registry")
		ocm.PushComponent(ctx, ocmClient, svc1, new("stable"))
		ocm.PushComponent(ctx, ocmClient, svc2, new("stable"))

		By("phase 1: creating the VectorTemplate CR and letting it assemble the initial vector")
		vectorTemplate := createVectorTemplateCR(ctx, "nodrift-test", testNamespace, []compref.Ref{svc1Alias, svc2Alias}, vectorTarget, "", nil)
		expectReadyCondition(vectorTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)
		waitForLatestVector(vectorTemplate)
		firstLatest := vectorTemplate.Status.LatestVector

		By("phase 2: nudging a reconcile without changing anything and expecting NoDriftDetected")
		nudgeReconcile(vectorTemplate)
		expectReadyCondition(vectorTemplate, metav1.ConditionTrue, konfidence.VectorTemplateNoDriftDetectedReason)

		By("verifying status.latestVector is unchanged (same concrete version)")
		Consistently(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)).To(Succeed())
			g.Expect(vectorTemplate.Status.LatestVector).To(Equal(firstLatest))
		}, 2*time.Second, interval).Should(Succeed())
	})

	It("should create a new vector when no latest vector has been recorded yet", func() {
		svc1 := createReference("konfidence.io/sample/firstcreate/service1:1.0.0")
		svc1Alias := createReference("konfidence.io/sample/firstcreate/service1:latest")
		svc2 := createReference("konfidence.io/sample/firstcreate/service2:2.0.0")
		svc2Alias := createReference("konfidence.io/sample/firstcreate/service2:latest")
		vectorTarget := createBareReference("konfidence.io/sample/vectors/first-test")

		By("creating mock component descriptors (no existing vector, empty status.latestVector)")
		ocm.PushComponent(ctx, ocmClient, svc1, new("latest"))
		ocm.PushComponent(ctx, ocmClient, svc2, new("latest"))

		By("creating a VectorTemplate CR")
		vectorTemplate := createVectorTemplateCR(ctx, "first-create-test", testNamespace, []compref.Ref{svc1Alias, svc2Alias}, vectorTarget, "", nil)

		By("verifying CR status shows VectorCreated")
		expectReadyCondition(vectorTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)

		By("verifying status.latestVector holds the newly assembled concrete version")
		latest := waitForLatestVector(vectorTemplate)
		Expect(latest.Component).To(Equal(vectorTarget.Component))

		By("verifying the assembled vector exists in OCI registry")
		descriptor, err := ocmClient.Get(ctx, latest)
		Expect(err).NotTo(HaveOccurred(), "failed to get vector descriptor from registry")
		Expect(descriptor.Component.Name).To(Equal("konfidence.io/sample/vectors/first-test"))
		Expect(descriptor.Component.Version).To(Equal(latest.Version), "descriptor version should match status.latestVector")
		Expect(descriptor.Component.References).To(HaveLen(2))
	})

	It("should create a vector with base vector artifacts merged in", func() {
		svc1 := createReference("konfidence.io/sample/inherit/service1:1.2.0")
		svc1Alias := createReference("konfidence.io/sample/inherit/service1:prod")
		svc2 := createReference("konfidence.io/sample/inherit/service2:3.1.0")
		svc2Alias := createReference("konfidence.io/sample/inherit/service2:prod")
		svc3 := createReference("konfidence.io/sample/inherit/service3:0.9.0")
		svc3Alias := createReference("konfidence.io/sample/inherit/service3:prod")
		baseTarget := createBareReference("konfidence.io/sample/vectors/base-vector")
		vectorTarget := createBareReference("konfidence.io/sample/vectors/inherit-test")

		By("creating mock component descriptors for the base (service3) and the dependent (service1, service2)")
		ocm.PushComponent(ctx, ocmClient, svc3, new("prod"))
		ocm.PushComponent(ctx, ocmClient, svc1, new("prod"))
		ocm.PushComponent(ctx, ocmClient, svc2, new("prod"))

		By("creating a base VectorTemplate CR and waiting for it to assemble a vector")
		baseTemplate := createVectorTemplateCR(ctx, "base-vector-test", testNamespace, []compref.Ref{svc3Alias}, baseTarget, "", nil)
		expectReadyCondition(baseTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)
		waitForLatestVector(baseTemplate)

		By("creating a dependent VectorTemplate CR that references the base by name")
		vectorTemplate := createVectorTemplateCR(ctx, "inherit-test", testNamespace, []compref.Ref{svc1Alias, svc2Alias}, vectorTarget, "base-vector-test", nil)

		By("verifying CR status shows VectorCreated")
		expectReadyCondition(vectorTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)

		By("verifying assembled vector contains 3 artifacts (base + components)")
		latest := waitForLatestVector(vectorTemplate)
		desc, err := ocmClient.Get(ctx, latest)
		Expect(err).NotTo(HaveOccurred(), "failed to get vector descriptor from registry")
		Expect(desc.Component.Version).To(Equal(latest.Version), "descriptor version should match status.latestVector")
		Expect(desc.Component.References).To(HaveLen(3))

		refVersions := make(map[string]string, len(desc.Component.References))
		for _, ref := range desc.Component.References {
			refVersions[ref.Component] = ref.Version
		}
		Expect(refVersions).To(HaveKeyWithValue(svc1.Component, svc1.Version))
		Expect(refVersions).To(HaveKeyWithValue(svc2.Component, svc2.Version))
		Expect(refVersions).To(HaveKeyWithValue(svc3.Component, svc3.Version))
	})

	It("should set DriftDetectionFailed when a component does not exist in the registry", func() {
		vectorTarget := createBareReference("konfidence.io/sample/notfound/vectors/notfound-test")
		nonExistentAlias := createReference("konfidence.io/sample/notfound/does-not-exist:latest")

		By("creating a VectorTemplate CR referencing a non-existent component (no mock data)")
		vectorTemplate := createVectorTemplateCR(ctx, "notfound-test", testNamespace, []compref.Ref{nonExistentAlias}, vectorTarget, "", nil)

		By("verifying CR status shows DriftDetectionFailed")
		expectReadyCondition(vectorTemplate, metav1.ConditionUnknown, konfidence.VectorTemplateDriftDetectionFailedReason)
	})

	It("should deduplicate components listed multiple times", func() {
		svc1 := createReference("konfidence.io/sample/dedup/service1:1.0.0")
		svc1Alias := createReference("konfidence.io/sample/dedup/service1:dedup")
		svc2 := createReference("konfidence.io/sample/dedup/service2:2.0.0")
		svc2Alias := createReference("konfidence.io/sample/dedup/service2:dedup")
		vectorTarget := createBareReference("konfidence.io/sample/vectors/dedup-test")

		By("creating mock component descriptors")
		ocm.PushComponent(ctx, ocmClient, svc1, new("dedup"))
		ocm.PushComponent(ctx, ocmClient, svc2, new("dedup"))

		By("creating a VectorTemplate CR with duplicate components")
		components := []compref.Ref{svc1Alias, svc1Alias, svc2Alias, svc1Alias, svc2Alias}
		vectorTemplate := createVectorTemplateCR(ctx, "dedup-test", testNamespace, components, vectorTarget, "", nil)

		By("verifying CR status shows VectorCreated")
		expectReadyCondition(vectorTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)

		By("verifying vector has deduplicated artifacts (2, not 5)")
		latest := waitForLatestVector(vectorTemplate)
		desc, err := ocmClient.Get(ctx, latest)
		Expect(err).NotTo(HaveOccurred(), "failed to get vector descriptor from registry")
		Expect(desc.Component.Version).To(Equal(latest.Version), "descriptor version should match status.latestVector")
		Expect(desc.Component.References).To(HaveLen(2), fmt.Sprintf("expected 2 deduplicated references, got %d", len(desc.Component.References)))
	})

	It("should create a vector with mixed semver and alias-based component versions", func() {
		svc1 := createReference("konfidence.io/sample/mixed/service1:v1.2.3")
		svc2 := createReference("konfidence.io/sample/mixed/service2:v2.0.1")
		svc2Alias := createReference("konfidence.io/sample/mixed/service2:edge")
		svc3 := createReference("konfidence.io/sample/mixed/service3:v3.1.0")
		svc3Alias := createReference("konfidence.io/sample/mixed/service3:prod")
		baseTarget := createBareReference("konfidence.io/sample/vectors/mixed-base")
		vectorTarget := createBareReference("konfidence.io/sample/vectors/mixed-test")

		By("creating mock component descriptors with mixed semver and tag versions")
		ocm.PushComponent(ctx, ocmClient, svc3, new("prod")) // component reachable via alias for the base
		ocm.PushComponent(ctx, ocmClient, svc1, nil)         // semver component, no alias
		ocm.PushComponent(ctx, ocmClient, svc2, new("edge")) // tag-based component with alias

		By("creating a base VectorTemplate CR (service3) and waiting for it to assemble")
		baseTemplate := createVectorTemplateCR(ctx, "mixed-base-test", testNamespace, []compref.Ref{svc3Alias}, baseTarget, "", nil)
		expectReadyCondition(baseTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)
		waitForLatestVector(baseTemplate)

		By("creating a VectorTemplate CR referencing semver directly, a tag via alias, and the base by name")
		vectorTemplate := createVectorTemplateCR(ctx,
			"mixed-test",
			testNamespace,
			[]compref.Ref{svc1, svc2Alias},
			vectorTarget,
			"mixed-base-test", nil)

		By("verifying CR status shows VectorCreated")
		expectReadyCondition(vectorTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)

		By("verifying vector in OCI registry contains all components with correct versions")
		latest := waitForLatestVector(vectorTemplate)
		desc, err := ocmClient.Get(ctx, latest)
		Expect(err).NotTo(HaveOccurred(), "failed to get vector descriptor from registry")
		Expect(desc.Component.Version).To(Equal(latest.Version), "descriptor version should match status.latestVector")
		Expect(desc.Component.References).To(HaveLen(3))

		refVersions := make(map[string]string, len(desc.Component.References))
		for _, ref := range desc.Component.References {
			refVersions[ref.Component] = ref.Version
		}
		Expect(refVersions).To(HaveKeyWithValue(svc1.Component, "v1.2.3"))
		Expect(refVersions).To(HaveKeyWithValue(svc2.Component, "v2.0.1"))
		Expect(refVersions).To(HaveKeyWithValue(svc3.Component, "v3.1.0"))
	})

	It("should set DriftDetectionFailed when the credential Secret does not exist", func() {
		svc1 := createReference("konfidence.io/sample/missing-creds/service1:v1.0.0")
		svc1Alias := createReference("konfidence.io/sample/missing-creds/service1:edge")
		vectorTarget := createBareReference("konfidence.io/sample/vectors/missing-creds")

		By("pushing a plain component")
		ocm.PushComponent(ctx, ocmClient, svc1, new("edge"))

		By("creating a VectorTemplate CR referencing a non-existent credential Secret")
		vectorTemplate := createPKIVectorTemplateCR(
			ctx, "missing-creds-test", testNamespace,
			[]compref.Ref{svc1Alias},
			vectorTarget, "",
			pkiVectorTemplateOptions{credSecretNames: []string{"non-existent-secret"}},
		)

		By("verifying CR status shows DriftDetectionFailed mentioning the missing secret")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)).To(Succeed())
			cond := meta.FindStatusCondition(vectorTemplate.Status.Conditions, konfidence.VectorTemplateReadyCondition)
			g.Expect(cond).NotTo(BeNil(), "Ready condition should be set")
			g.Expect(cond.Status).NotTo(Equal(metav1.ConditionTrue))
			g.Expect(cond.Reason).To(Equal(konfidence.VectorTemplateDriftDetectionFailedReason))
			g.Expect(cond.Message).To(ContainSubstring("non-existent-secret"))
			g.Expect(cond.ObservedGeneration).To(Equal(vectorTemplate.Generation))
		}, timeout, interval).Should(Succeed())
	})

	It("should fail when uploadTarget carries a version instead of a bare component", func() {
		svc1 := createReference("konfidence.io/sample/version-fail/service1:v1.0.0")
		svc1Alias := createReference("konfidence.io/sample/version-fail/service1:latest")
		// uploadTarget must be a bare component; a versioned target must be rejected.
		versionedTarget := createReference("konfidence.io/sample/vectors/version-fail:v1.2.3")

		By("creating mock component descriptor")
		ocm.PushComponent(ctx, ocmClient, svc1, new("latest"))

		By("creating a VectorTemplate CR with a versioned uploadTarget")
		vectorTemplate := createVectorTemplateCR(ctx, "version-fail-test", testNamespace, []compref.Ref{svc1Alias}, versionedTarget, "", nil)

		By("verifying CR status shows failure because uploadTarget must not carry a version")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)).To(Succeed())
			cond := meta.FindStatusCondition(vectorTemplate.Status.Conditions, konfidence.VectorTemplateReadyCondition)
			g.Expect(cond).NotTo(BeNil(), "Ready condition should be set")
			g.Expect(cond.Status).NotTo(Equal(metav1.ConditionTrue))
			g.Expect(cond.Reason).To(Equal(konfidence.VectorTemplateDriftDetectionFailedReason))
			g.Expect(cond.Message).To(ContainSubstring("must not carry a version"))
			g.Expect(cond.ObservedGeneration).To(Equal(vectorTemplate.Generation))
		}, timeout, interval).Should(Succeed())
	})

	It("should report no drift when vector configuration matches", func() {
		svc1 := createReference("konfidence.io/sample/conf-no-drift/service1:1.0.0")
		svc1Alias := createReference("konfidence.io/sample/conf-no-drift/service1:stable")
		svc2 := createReference("konfidence.io/sample/conf-no-drift/service2:2.0.0")
		svc2Alias := createReference("konfidence.io/sample/conf-no-drift/service2:stable")
		vectorTarget := createBareReference("konfidence.io/sample/vectors/conf-no-drift-test")

		By("creating mock component descriptors in OCI registry")
		ocm.PushComponent(ctx, ocmClient, svc1, new("stable"))
		ocm.PushComponent(ctx, ocmClient, svc2, new("stable"))

		vectorConfig := konfidence.VectorConfig{
			Features: &runtime.RawExtension{Raw: []byte(`{"test":"1234"}`)},
			Authored: &runtime.RawExtension{Raw: []byte(`{"cfg":"abc"}`)},
		}

		By("phase 1: creating the VectorTemplate CR with a vector config and letting it assemble")
		vectorTemplate := createVectorTemplateCR(ctx, "conf-nodrift-test", testNamespace, []compref.Ref{svc1Alias, svc2Alias}, vectorTarget, "", &vectorConfig)
		expectReadyCondition(vectorTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)
		waitForLatestVector(vectorTemplate)
		firstLatest := vectorTemplate.Status.LatestVector

		By("phase 2: nudging a reconcile without changing anything and expecting NoDriftDetected")
		nudgeReconcile(vectorTemplate)
		expectReadyCondition(vectorTemplate, metav1.ConditionTrue, konfidence.VectorTemplateNoDriftDetectedReason)

		By("verifying status.latestVector is unchanged")
		Consistently(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)).To(Succeed())
			g.Expect(vectorTemplate.Status.LatestVector).To(Equal(firstLatest))
		}, 2*time.Second, interval).Should(Succeed())
	})

	It("should report drift when vector configuration has changed", func() {
		svc1 := createReference("konfidence.io/sample/conf-drift/service1:1.0.0")
		svc1Alias := createReference("konfidence.io/sample/conf-drift/service1:stable")
		svc2 := createReference("konfidence.io/sample/conf-drift/service2:2.0.0")
		svc2Alias := createReference("konfidence.io/sample/conf-drift/service2:stable")
		vectorTarget := createBareReference("konfidence.io/sample/vectors/conf-drift-test")

		By("creating mock component descriptors in OCI registry")
		ocm.PushComponent(ctx, ocmClient, svc1, new("stable"))
		ocm.PushComponent(ctx, ocmClient, svc2, new("stable"))

		initialConfig := konfidence.VectorConfig{
			Features: &runtime.RawExtension{Raw: []byte(`{"test":"1234"}`)},
			Authored: &runtime.RawExtension{Raw: []byte(`{"cfg":"abc"}`)},
		}

		By("phase 1: creating the VectorTemplate CR with an initial vector config and letting it assemble")
		vectorTemplate := createVectorTemplateCR(ctx, "conf-drift-test", testNamespace, []compref.Ref{svc1Alias, svc2Alias}, vectorTarget, "", &initialConfig)
		expectReadyCondition(vectorTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)
		waitForLatestVector(vectorTemplate)

		By("phase 2: changing the vector configuration on the spec")
		newVectorConfig := konfidence.VectorConfig{
			Features: &runtime.RawExtension{Raw: []byte(`{"label":"Test"}`)},
			Authored: &runtime.RawExtension{Raw: []byte(`{"cfg":"abc", "dbPort":3306}`)},
		}
		newVectorConfigContent, err := getVectorConfigurationContent(newVectorConfig)
		Expect(err).NotTo(HaveOccurred())
		updateVectorConfig(vectorTemplate, &newVectorConfig)

		By("verifying CR status shows VectorCreated")
		expectReadyCondition(vectorTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)

		By("verifying the assembled vector contains the updated vector config")
		latest := waitForLatestVector(vectorTemplate)
		readBlob, _, err := ocmClient.GetLocalResource(ctx, latest, map[string]string{
			"name":    ocm2.DefaultVectorConfigName,
			"version": ocm2.DefaultVectorConfigVersion,
		})
		Expect(err).NotTo(HaveOccurred(), "failed to get vector configuration local resource from oci")

		var buf bytes.Buffer
		err = blob.Copy(&buf, readBlob)
		Expect(err).NotTo(HaveOccurred(), "failed to read vector configuration local resource")
		Expect(newVectorConfigContent).To(Equal(buf.Bytes()), "new vector should have updated vector config")
	})

	It("should report drift when existing vector has no configuration and a new config is added", func() {
		svc1 := createReference("konfidence.io/sample/conf-drift-2/service1:1.0.0")
		svc1Alias := createReference("konfidence.io/sample/conf-drift-2/service1:stable")
		svc2 := createReference("konfidence.io/sample/conf-drift-2/service2:2.0.0")
		svc2Alias := createReference("konfidence.io/sample/conf-drift-2/service2:stable")
		vectorTarget := createBareReference("konfidence.io/sample/vectors/conf-drift-2-test")

		By("creating mock component descriptors in OCI registry")
		ocm.PushComponent(ctx, ocmClient, svc1, new("stable"))
		ocm.PushComponent(ctx, ocmClient, svc2, new("stable"))

		By("phase 1: creating the VectorTemplate CR with NO vector config and letting it assemble")
		vectorTemplate := createVectorTemplateCR(ctx, "conf-drift-2-test", testNamespace, []compref.Ref{svc1Alias, svc2Alias}, vectorTarget, "", nil)
		expectReadyCondition(vectorTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)
		waitForLatestVector(vectorTemplate)

		By("phase 2: adding a vector configuration to the spec")
		newVectorConfig := konfidence.VectorConfig{
			Features: &runtime.RawExtension{Raw: []byte(`{"label":"Test"}`)},
			Authored: &runtime.RawExtension{Raw: []byte(`{"cfg":"abc", "dbPort":3306}`)},
		}
		newVectorConfigContent, err := getVectorConfigurationContent(newVectorConfig)
		Expect(err).NotTo(HaveOccurred())
		updateVectorConfig(vectorTemplate, &newVectorConfig)

		By("verifying CR status shows VectorCreated")
		expectReadyCondition(vectorTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)

		By("verifying the assembled vector contains the updated vector config")
		latest := waitForLatestVector(vectorTemplate)
		readBlob, _, err := ocmClient.GetLocalResource(ctx, latest, map[string]string{
			"name":    ocm2.DefaultVectorConfigName,
			"version": ocm2.DefaultVectorConfigVersion,
		})
		Expect(err).NotTo(HaveOccurred(), "failed to get vector configuration local resource from oci")

		var buf bytes.Buffer
		err = blob.Copy(&buf, readBlob)
		Expect(err).NotTo(HaveOccurred(), "failed to read vector configuration local resource")
		Expect(newVectorConfigContent).To(Equal(buf.Bytes()), "new vector should have updated vector config")
	})

	It("should report drift when existing vector has vector config but new vector does not", func() {
		svc1 := createReference("konfidence.io/sample/conf-drift-3/service1:1.0.0")
		svc1Alias := createReference("konfidence.io/sample/conf-drift-3/service1:stable")
		svc2 := createReference("konfidence.io/sample/conf-drift-3/service2:2.0.0")
		svc2Alias := createReference("konfidence.io/sample/conf-drift-3/service2:stable")
		vectorTarget := createBareReference("konfidence.io/sample/vectors/conf-drift-3-test")

		By("creating mock component descriptors in OCI registry")
		ocm.PushComponent(ctx, ocmClient, svc1, new("stable"))
		ocm.PushComponent(ctx, ocmClient, svc2, new("stable"))

		initialConfig := konfidence.VectorConfig{
			Features: &runtime.RawExtension{Raw: []byte(`{"test":"1234"}`)},
			Authored: &runtime.RawExtension{Raw: []byte(`{"cfg":"abc"}`)},
		}

		By("phase 1: creating the VectorTemplate CR WITH a vector config and letting it assemble")
		vectorTemplate := createVectorTemplateCR(ctx, "conf-drift-3-test", testNamespace, []compref.Ref{svc1Alias, svc2Alias}, vectorTarget, "", &initialConfig)
		expectReadyCondition(vectorTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)
		waitForLatestVector(vectorTemplate)

		By("phase 2: removing the vector configuration from the spec")
		updateVectorConfig(vectorTemplate, nil)

		By("verifying CR status shows VectorCreated")
		expectReadyCondition(vectorTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)

		By("verifying the assembled vector contains no vector config")
		latest := waitForLatestVector(vectorTemplate)
		_, _, err := ocmClient.GetLocalResource(ctx, latest, map[string]string{
			"name":    ocm2.DefaultVectorConfigName,
			"version": ocm2.DefaultVectorConfigVersion,
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("found 0 candidates"))
	})

	It("should set WaitingForBase and not assemble while the base has no latest vector", func() {
		svc1 := createReference("konfidence.io/sample/wait-base/service1:1.0.0")
		svc1Alias := createReference("konfidence.io/sample/wait-base/service1:stable")
		baseTarget := createBareReference("konfidence.io/sample/vectors/wait-base")
		vectorTarget := createBareReference("konfidence.io/sample/vectors/wait-base-dependent")

		By("creating the dependent component")
		ocm.PushComponent(ctx, ocmClient, svc1, new("stable"))

		By("creating a base VectorTemplate CR that cannot assemble yet (missing component)")
		// The base references a component that does not exist, so it never populates
		// status.latestVector - it stays with an empty latest vector.
		missingBaseComponent := createReference("konfidence.io/sample/wait-base/missing:latest")
		createVectorTemplateCR(ctx, "wait-base-base", testNamespace, []compref.Ref{missingBaseComponent}, baseTarget, "", nil)

		By("creating a dependent VectorTemplate CR referencing the not-yet-ready base")
		dependent := createVectorTemplateCR(ctx, "wait-base-dependent", testNamespace, []compref.Ref{svc1Alias}, vectorTarget, "wait-base-base", nil)

		By("verifying the dependent reports WaitingForBase and does not assemble")
		expectReadyCondition(dependent, metav1.ConditionFalse, konfidence.VectorTemplateWaitingForBaseReason)
		Consistently(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(dependent), dependent)).To(Succeed())
			g.Expect(dependent.Status.LatestVector).To(BeEmpty(), "dependent must not assemble while base is not ready")
		}, 2*time.Second, interval).Should(Succeed())
	})

	It("should reconcile the dependent automatically when the base assembles (watch propagation)", func() {
		svc1 := createReference("konfidence.io/sample/watch/service1:1.0.0")
		svc1Alias := createReference("konfidence.io/sample/watch/service1:stable")
		baseComponent := createReference("konfidence.io/sample/watch/base-service:1.0.0")
		baseComponentAlias := createReference("konfidence.io/sample/watch/base-service:stable")
		baseTarget := createBareReference("konfidence.io/sample/vectors/watch-base")
		vectorTarget := createBareReference("konfidence.io/sample/vectors/watch-dependent")

		By("creating the dependent's own component, but NOT the base's component yet")
		ocm.PushComponent(ctx, ocmClient, svc1, new("stable"))

		By("creating a base VectorTemplate CR that cannot assemble yet (its component is missing)")
		baseTemplate := createVectorTemplateCR(ctx, "watch-base", testNamespace, []compref.Ref{baseComponentAlias}, baseTarget, "", nil)

		By("creating a dependent VectorTemplate CR that references the base by name")
		dependent := createVectorTemplateCR(ctx, "watch-dependent", testNamespace, []compref.Ref{svc1Alias}, vectorTarget, "watch-base", nil)

		By("verifying the dependent initially waits for the base")
		expectReadyCondition(dependent, metav1.ConditionFalse, konfidence.VectorTemplateWaitingForBaseReason)

		By("recording the dependent's observed generation before the base becomes ready")
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(dependent), dependent)).To(Succeed())
		dependentGenerationBefore := dependent.Generation

		By("making the base assemblable by pushing its component, unblocking the base")
		ocm.PushComponent(ctx, ocmClient, baseComponent, new("stable"))
		expectReadyCondition(baseTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)
		waitForLatestVector(baseTemplate)

		By("verifying the dependent is re-reconciled by the watch (no spec change) and assembles")
		expectReadyCondition(dependent, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)
		latest := waitForLatestVector(dependent)

		By("verifying the wake-up was watch-driven: the dependent's spec/generation never changed")
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(dependent), dependent)).To(Succeed())
		Expect(dependent.Generation).To(Equal(dependentGenerationBefore),
			"dependent must not have been re-applied; the base watch is what re-enqueued it")

		By("verifying the dependent's assembled vector merged the base's component")
		desc, err := ocmClient.Get(ctx, latest)
		Expect(err).NotTo(HaveOccurred())
		refComponents := make([]string, 0, len(desc.Component.References))
		for _, ref := range desc.Component.References {
			refComponents = append(refComponents, ref.Component)
		}
		Expect(refComponents).To(ContainElement(svc1.Component))
		Expect(refComponents).To(ContainElement(baseComponent.Component))
	})

	It("should keep the dependent's vector when the base latest vector is cleared and then repopulated", func() {
		svc1 := createReference("konfidence.io/sample/roundtrip/service1:1.0.0")
		svc1Alias := createReference("konfidence.io/sample/roundtrip/service1:stable")
		baseComponent := createReference("konfidence.io/sample/roundtrip/base-service:1.0.0")
		baseComponentAlias := createReference("konfidence.io/sample/roundtrip/base-service:stable")
		baseTarget := createBareReference("konfidence.io/sample/vectors/roundtrip-base")
		vectorTarget := createBareReference("konfidence.io/sample/vectors/roundtrip-dependent")

		By("creating both components")
		ocm.PushComponent(ctx, ocmClient, svc1, new("stable"))
		ocm.PushComponent(ctx, ocmClient, baseComponent, new("stable"))

		By("creating a base VectorTemplate CR and waiting for it to assemble")
		baseTemplate := createVectorTemplateCR(ctx, "roundtrip-base", testNamespace, []compref.Ref{baseComponentAlias}, baseTarget, "", nil)
		expectReadyCondition(baseTemplate, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)
		waitForLatestVector(baseTemplate)
		baseLatestRaw := baseTemplate.Status.LatestVector

		By("creating a dependent that assembles against the ready base")
		dependent := createVectorTemplateCR(ctx, "roundtrip-dependent", testNamespace, []compref.Ref{svc1Alias}, vectorTarget, "roundtrip-base", nil)
		expectReadyCondition(dependent, metav1.ConditionTrue, konfidence.VectorTemplateVectorCreatedReason)
		waitForLatestVector(dependent)
		dependentLatestBefore := dependent.Status.LatestVector

		By("clearing the base's status.latestVector (simulating a user edit)")
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(baseTemplate), baseTemplate)).To(Succeed())
		baseTemplate.Status.LatestVector = ""
		Expect(k8sClient.Status().Update(ctx, baseTemplate)).To(Succeed())

		By("verifying the dependent keeps its previously assembled vector (not rolled back)")
		Consistently(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(dependent), dependent)).To(Succeed())
			g.Expect(dependent.Status.LatestVector).To(Equal(dependentLatestBefore),
				"dependent's existing vector must survive a transient base clear")
		}, 2*time.Second, interval).Should(Succeed())

		By("repopulating the base's status.latestVector")
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(baseTemplate), baseTemplate)).To(Succeed())
		baseTemplate.Status.LatestVector = baseLatestRaw
		Expect(k8sClient.Status().Update(ctx, baseTemplate)).To(Succeed())

		By("verifying the dependent recovers to Ready once the base is restored")
		// The dependent's content did not change, so it converges to NoDriftDetected
		// (still Ready=True) rather than re-assembling.
		expectReadyCondition(dependent, metav1.ConditionTrue, konfidence.VectorTemplateNoDriftDetectedReason)
	})
})
