package controller

import (
	"fmt"
	"time"

	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNamespace = "default"
	timeout       = 30 * time.Second
	interval      = 250 * time.Millisecond
)

var _ = Describe("VectorTemplate controller tests", Ordered, Serial, func() {

	BeforeEach(func() {
		// Clean up any existing VectorTemplate CRs before each test
		Expect(k8sClient.DeleteAllOf(ctx, &global.VectorTemplate{}, client.InNamespace(testNamespace))).To(Succeed())
		// Wait until all VectorTemplates are gone
		Eventually(func(g Gomega) {
			list := &global.VectorTemplateList{}
			g.Expect(k8sClient.List(ctx, list, client.InNamespace(testNamespace))).To(Succeed())
			g.Expect(list.Items).To(BeEmpty())
		}, timeout, interval).Should(Succeed())
	})

	It("should create a new vector when drift is detected against an existing vector", func() {
		svc1 := createReference("konfidence.io/sample/drift/service1:v5.0.0")
		svc1Alias := createReference("konfidence.io/sample/drift/service1:edge")
		svc2 := createReference("konfidence.io/sample/drift/service2:v5.0.0")
		svc2Alias := createReference("konfidence.io/sample/drift/service2:edge")
		versionVector := createReference(fmt.Sprintf("konfidence.io/sample/vectors/drift-test:%s", oldTestVersion))
		aliasVector := createReference("konfidence.io/sample/vectors/drift-test:dev-eu10")

		By("creating mock component descriptors in oci")
		pushComponent(ctx, ocmClient, svc1, new("edge"))
		pushComponent(ctx, ocmClient, svc2, new("edge"))

		By("creating a mock vector with older versions")
		pushVector(ctx, ocmClient, versionVector, []compref.Ref{
			createReference("konfidence.io/sample/drift/service1:v5.0.0"),
			createReference("konfidence.io/sample/drift/service2:v3.2.1"), // older version: drift
		}, "dev-eu10")

		By("creating a VectorTemplate CR")
		vectorTemplate := createVectorTemplateCR(ctx, "drift-test", testNamespace, []compref.Ref{svc1Alias, svc2Alias}, aliasVector, nil)

		By("verifying CR status shows VectorCreated")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)).To(Succeed())
			statusCondition := meta.FindStatusCondition(vectorTemplate.Status.Conditions, global.VectorTemplateReadyCondition)
			g.Expect(statusCondition).NotTo(BeNil(), "Ready condition should be set")
			g.Expect(statusCondition.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(statusCondition.Reason).To(Equal(global.VectorTemplateVectorCreatedReason))
			g.Expect(statusCondition.ObservedGeneration).To(Equal(vectorTemplate.Generation))
		}, timeout, interval).Should(Succeed())

		By("verifying vector in oci contains updated artifact versions")
		descriptor, err := ocmClient.Get(ctx, aliasVector)
		Expect(err).NotTo(HaveOccurred(), "failed to get vector descriptor from oci")
		Expect(descriptor.Component.References).To(HaveLen(2))
		Expect(descriptor.Component.Version).To(Equal(testVersion), "new vector should have the static test version")

		refVersions := make(map[string]string, len(descriptor.Component.References))
		for _, ref := range descriptor.Component.References {
			refVersions[ref.Component] = ref.Version
		}
		Expect(refVersions).To(HaveKeyWithValue(svc1.Component, svc1.Version))
		Expect(refVersions).To(HaveKeyWithValue(svc2.Component, svc2.Version))
	})

	It("should report NoDriftDetected when the vector already matches", func() {
		svc1 := createReference("konfidence.io/sample/nodrift/service1:1.0.0")
		svc1Alias := createReference("konfidence.io/sample/nodrift/service1:stable")
		svc2 := createReference("konfidence.io/sample/nodrift/service2:2.0.0")
		svc2Alias := createReference("konfidence.io/sample/nodrift/service2:stable")
		versionVector := createReference(fmt.Sprintf("konfidence.io/sample/vectors/nodrift-test:%s", oldTestVersion))
		aliasVector := createReference("konfidence.io/sample/vectors/nodrift-test:stable")

		By("creating mock component descriptors in OCI registry")
		pushComponent(ctx, ocmClient, svc1, new("stable"))
		pushComponent(ctx, ocmClient, svc2, new("stable"))

		By("creating a mock vector with matching versions")
		pushVector(ctx, ocmClient, versionVector, []compref.Ref{svc1, svc2}, "stable")

		By("creating a VectorTemplate CR")
		vectorTemplate := createVectorTemplateCR(ctx, "nodrift-test", testNamespace, []compref.Ref{svc1Alias, svc2Alias}, aliasVector, nil)

		By("verifying CR status shows NoDriftDetected")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)).To(Succeed())
			statusCondition := meta.FindStatusCondition(vectorTemplate.Status.Conditions, global.VectorTemplateReadyCondition)
			g.Expect(statusCondition).NotTo(BeNil(), "Ready condition should be set")
			g.Expect(statusCondition.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(statusCondition.Reason).To(Equal(global.VectorTemplateNoDriftDetectedReason))
			g.Expect(statusCondition.ObservedGeneration).To(Equal(vectorTemplate.Generation))
		}, timeout, interval).Should(Succeed())

		By("verifying vector in OCI registry still has the original version")
		descriptor, err := ocmClient.Get(ctx, aliasVector)
		Expect(err).NotTo(HaveOccurred(), "failed to get vector descriptor from registry")
		Expect(descriptor.Component.Version).To(Equal(oldTestVersion), "vector version should not have changed")
	})

	It("should create a new vector when no vector exists yet", func() {
		svc1 := createReference("konfidence.io/sample/firstcreate/service1:1.0.0")
		svc1Alias := createReference("konfidence.io/sample/firstcreate/service1:latest")
		svc2 := createReference("konfidence.io/sample/firstcreate/service2:2.0.0")
		svc2Alias := createReference("konfidence.io/sample/firstcreate/service2:latest")
		aliasVector := createReference("konfidence.io/sample/vectors/first-test:latest")

		By("creating mock component descriptors (no existing vector)")
		pushComponent(ctx, ocmClient, svc1, new("latest"))
		pushComponent(ctx, ocmClient, svc2, new("latest"))

		By("creating a VectorTemplate CR")
		vectorTemplate := createVectorTemplateCR(ctx, "first-create-test", testNamespace, []compref.Ref{svc1Alias, svc2Alias}, aliasVector, nil)

		By("verifying CR status shows VectorCreated")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)).To(Succeed())
			statusCondition := meta.FindStatusCondition(vectorTemplate.Status.Conditions, global.VectorTemplateReadyCondition)
			g.Expect(statusCondition).NotTo(BeNil(), "Ready condition should be set")
			g.Expect(statusCondition.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(statusCondition.Reason).To(Equal(global.VectorTemplateVectorCreatedReason))
			g.Expect(statusCondition.ObservedGeneration).To(Equal(vectorTemplate.Generation))
		}, timeout, interval).Should(Succeed())

		By("verifying vector exists in OCI registry")
		descriptor, err := ocmClient.Get(ctx, aliasVector)
		Expect(err).NotTo(HaveOccurred(), "failed to get vector descriptor from registry")
		Expect(descriptor.Component.Name).To(Equal("konfidence.io/sample/vectors/first-test"))
		Expect(descriptor.Component.Version).To(Equal(testVersion), "newly created vector should have the static test version")
		Expect(descriptor.Component.References).To(HaveLen(2))
	})

	It("should create a vector with base vector artifacts merged in", func() {
		svc1 := createReference("konfidence.io/sample/inherit/service1:1.2.0")
		svc1Alias := createReference("konfidence.io/sample/inherit/service1:prod")
		svc2 := createReference("konfidence.io/sample/inherit/service2:3.1.0")
		svc2Alias := createReference("konfidence.io/sample/inherit/service2:prod")
		svc3 := createReference("konfidence.io/sample/inherit/service3:0.9.0")
		aliasVector := createReference("konfidence.io/sample/vectors/inherit-test:prod")
		versionBaseVector := createReference(fmt.Sprintf("konfidence.io/sample/vectors/base-vector:%s", oldTestVersion))
		aliasBaseVector := createReference("konfidence.io/sample/vectors/base-vector:base")

		By("creating a mock base vector with service3")
		pushComponent(ctx, ocmClient, svc3, new("prod"))
		pushVector(ctx, ocmClient, versionBaseVector, []compref.Ref{svc3}, "base")

		By("creating mock component descriptors for service1 and service2")
		pushComponent(ctx, ocmClient, svc1, new("prod"))
		pushComponent(ctx, ocmClient, svc2, new("prod"))

		By("creating a VectorTemplate CR with base")
		vectorTemplate := createVectorTemplateCR(ctx, "inherit-test", testNamespace, []compref.Ref{svc1Alias, svc2Alias}, aliasVector, &aliasBaseVector)

		By("verifying CR status shows VectorCreated")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)).To(Succeed())
			statusCondition := meta.FindStatusCondition(vectorTemplate.Status.Conditions, global.VectorTemplateReadyCondition)
			g.Expect(statusCondition).NotTo(BeNil(), "Ready condition should be set")
			g.Expect(statusCondition.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(statusCondition.Reason).To(Equal(global.VectorTemplateVectorCreatedReason))
			g.Expect(statusCondition.ObservedGeneration).To(Equal(vectorTemplate.Generation))
		}, timeout, interval).Should(Succeed())

		By("verifying vector in OCI registry contains 3 artifacts (base + components)")
		desc, err := ocmClient.Get(ctx, aliasVector)
		Expect(err).NotTo(HaveOccurred(), "failed to get vector descriptor from registry")
		Expect(desc.Component.Version).To(Equal(testVersion), "newly created vector should have the static test version")
		Expect(desc.Component.References).To(HaveLen(3))

		// Verify all three services are present with correct versions
		refVersions := make(map[string]string, len(desc.Component.References))
		for _, ref := range desc.Component.References {
			refVersions[ref.Component] = ref.Version
		}
		Expect(refVersions).To(HaveKeyWithValue(svc1.Component, svc1.Version))
		Expect(refVersions).To(HaveKeyWithValue(svc2.Component, svc2.Version))
		Expect(refVersions).To(HaveKeyWithValue(svc3.Component, svc3.Version))
	})

	It("should set DriftDetectionFailed when a component does not exist in the registry", func() {
		aliasVector := createReference("konfidence.io/sample/notfound/vectors/notfound-test:broken")
		nonExistentAlias := createReference("konfidence.io/sample/notfound/does-not-exist:latest")

		By("creating a VectorTemplate CR referencing a non-existent component (no mock data)")
		vectorTemplate := createVectorTemplateCR(ctx, "notfound-test", testNamespace, []compref.Ref{nonExistentAlias}, aliasVector, nil)

		By("verifying CR status shows DriftDetectionFailed")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)).To(Succeed())
			statusCondition := meta.FindStatusCondition(vectorTemplate.Status.Conditions, global.VectorTemplateReadyCondition)
			g.Expect(statusCondition).NotTo(BeNil(), "Ready condition should be set")
			g.Expect(statusCondition.Status).NotTo(Equal(metav1.ConditionTrue))
			g.Expect(statusCondition.Reason).To(Equal(global.VectorTemplateDriftDetectionFailedReason))
			g.Expect(statusCondition.ObservedGeneration).To(Equal(vectorTemplate.Generation))
		}, timeout, interval).Should(Succeed())
	})

	It("should deduplicate components listed multiple times", func() {
		svc1 := createReference("konfidence.io/sample/dedup/service1:1.0.0")
		svc1Alias := createReference("konfidence.io/sample/dedup/service1:dedup")
		svc2 := createReference("konfidence.io/sample/dedup/service2:2.0.0")
		svc2Alias := createReference("konfidence.io/sample/dedup/service2:dedup")
		aliasVector := createReference("konfidence.io/sample/vectors/dedup-test:dedup")

		By("creating mock component descriptors")
		pushComponent(ctx, ocmClient, svc1, new("dedup"))
		pushComponent(ctx, ocmClient, svc2, new("dedup"))

		By("creating a VectorTemplate CR with duplicate components")
		components := []compref.Ref{svc1Alias, svc1Alias, svc2Alias, svc1Alias, svc2Alias}
		vectorTemplate := createVectorTemplateCR(ctx, "dedup-test", testNamespace, components, aliasVector, nil)

		By("verifying CR status shows VectorCreated")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)).To(Succeed())
			statusCondition := meta.FindStatusCondition(vectorTemplate.Status.Conditions, global.VectorTemplateReadyCondition)
			g.Expect(statusCondition).NotTo(BeNil(), "Ready condition should be set")
			g.Expect(statusCondition.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(statusCondition.Reason).To(Equal(global.VectorTemplateVectorCreatedReason))
			g.Expect(statusCondition.ObservedGeneration).To(Equal(vectorTemplate.Generation))
		}, timeout, interval).Should(Succeed())

		By("verifying vector has deduplicated artifacts (2, not 5)")
		desc, err := ocmClient.Get(ctx, aliasVector)
		Expect(err).NotTo(HaveOccurred(), "failed to get vector descriptor from registry")
		Expect(desc.Component.Version).To(Equal(testVersion), "newly created vector should have the static test version")
		Expect(desc.Component.References).To(HaveLen(2), fmt.Sprintf("expected 2 deduplicated references, got %d", len(desc.Component.References)))
	})

	It("should create a vector with mixed semver and alias-based component versions", func() {
		svc1 := createReference("konfidence.io/sample/mixed/service1:v1.2.3")
		svc2 := createReference("konfidence.io/sample/mixed/service2:v2.0.1")
		svc2Alias := createReference("konfidence.io/sample/mixed/service2:edge")
		svc3 := createReference("konfidence.io/sample/mixed/service3:v3.1.0")
		aliasVector := createReference("konfidence.io/sample/vectors/mixed-test:latest")
		versionBaseVector := createReference(fmt.Sprintf("konfidence.io/sample/vectors/mixed-base:%s", oldTestVersion))

		By("creating a mock base vector with semver component")
		pushComponent(ctx, ocmClient, svc3, nil)
		pushVector(ctx, ocmClient, versionBaseVector, []compref.Ref{svc3}, "stable")

		By("creating mock component descriptors with mixed semver and tag versions")
		pushComponent(ctx, ocmClient, svc1, nil)         // semver component, no alias
		pushComponent(ctx, ocmClient, svc2, new("edge")) // tag-based component with alias

		By("creating a VectorTemplate CR referencing semver directly and tag via alias")
		vectorTemplate := createVectorTemplateCR(ctx,
			"mixed-test",
			testNamespace,
			[]compref.Ref{svc1, svc2Alias},
			aliasVector,
			&versionBaseVector)

		By("verifying CR status shows VectorCreated")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)).To(Succeed())
			statusCondition := meta.FindStatusCondition(vectorTemplate.Status.Conditions, global.VectorTemplateReadyCondition)
			g.Expect(statusCondition).NotTo(BeNil(), "Ready condition should be set")
			g.Expect(statusCondition.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(statusCondition.Reason).To(Equal(global.VectorTemplateVectorCreatedReason))
			g.Expect(statusCondition.ObservedGeneration).To(Equal(vectorTemplate.Generation))
		}, timeout, interval).Should(Succeed())

		By("verifying vector in OCI registry contains all components with correct versions")
		desc, err := ocmClient.Get(ctx, aliasVector)
		Expect(err).NotTo(HaveOccurred(), "failed to get vector descriptor from registry")
		Expect(desc.Component.Version).To(Equal(testVersion), "newly created vector should have the static test version")
		Expect(desc.Component.References).To(HaveLen(3))

		// Verify all three services are present with correct versions (mix of semver and tag)
		refVersions := make(map[string]string, len(desc.Component.References))
		for _, ref := range desc.Component.References {
			refVersions[ref.Component] = ref.Version
		}
		Expect(refVersions).To(HaveKeyWithValue(svc1.Component, "v1.2.3"))
		Expect(refVersions).To(HaveKeyWithValue(svc2.Component, "v2.0.1"))
		Expect(refVersions).To(HaveKeyWithValue(svc3.Component, "v3.1.0"))
	})

	It("should fail when uploadTarget uses a semver version instead of an alias", func() {
		svc1 := createReference("konfidence.io/sample/semver-fail/service1:v1.0.0")
		svc1Alias := createReference("konfidence.io/sample/semver-fail/service1:latest")
		semverVector := createReference("konfidence.io/sample/vectors/semver-fail:v1.2.3")

		By("creating mock component descriptor")
		pushComponent(ctx, ocmClient, svc1, new("latest"))

		By("creating a VectorTemplate CR with semver uploadTarget")
		vectorTemplate := createVectorTemplateCR(ctx, "semver-fail-test", testNamespace, []compref.Ref{svc1Alias}, semverVector, nil)

		By("verifying CR status shows failure due to semver uploadTarget")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)).To(Succeed())
			statusCondition := meta.FindStatusCondition(vectorTemplate.Status.Conditions, global.VectorTemplateReadyCondition)
			g.Expect(statusCondition).NotTo(BeNil(), "Ready condition should be set")
			g.Expect(statusCondition.Status).NotTo(Equal(metav1.ConditionTrue))
			// The reason should indicate validation failure or similar
			g.Expect(statusCondition.Reason).To(Equal(global.VectorTemplateDriftDetectionFailedReason))
			g.Expect(statusCondition.Message).To(ContainSubstring("semver"))
			g.Expect(statusCondition.ObservedGeneration).To(Equal(vectorTemplate.Generation))
		}, timeout, interval).Should(Succeed())
	})
})
