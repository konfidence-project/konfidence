/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"fmt"
	"time"

	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		svc1 := "konfidence.io/sample/drift/service1"
		svc2 := "konfidence.io/sample/drift/service2"
		vectorName := "konfidence.io/sample/vectors/drift-test"

		By("creating mock component descriptors in Zot")
		pushComponent(ctx, ocmClient, registryEndpoint, svc1, "1.2.0")
		pushComponent(ctx, ocmClient, registryEndpoint, svc2, "3.1.0")

		By("creating a mock vector with older versions")
		pushVector(ctx, ocmClient, registryEndpoint, vectorName, "2026.1.1-000000000Z", []vectorArtifact{
			{Name: svc1, Version: "1.2.0"},
			{Name: svc2, Version: "2.99.0"}, // older version → drift
		})

		By("creating a VectorTemplate CR")
		vectorTemplate := newVectorTemplateCR("drift-test", testNamespace, registryEndpoint, vectorName, []string{svc1, svc2}, nil)
		Expect(k8sClient.Create(ctx, vectorTemplate)).To(Succeed())

		By("verifying CR status shows VectorCreated")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)).To(Succeed())
			statusCondition := meta.FindStatusCondition(vectorTemplate.Status.Conditions, global.VectorTemplateReadyCondition)
			g.Expect(statusCondition).NotTo(BeNil(), "Ready condition should be set")
			g.Expect(statusCondition.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(statusCondition.Reason).To(Equal(global.VectorTemplateVectorCreatedReason))
			g.Expect(statusCondition.ObservedGeneration).To(Equal(vectorTemplate.Generation))
		}, timeout, interval).Should(Succeed())

		By("verifying vector in Zot contains updated artifact versions")
		descriptor, err := getDescriptorFromRegistry(ctx, ocmClient, registryEndpoint, vectorName)
		Expect(err).NotTo(HaveOccurred(), "failed to get vector descriptor from registry")
		Expect(descriptor.Component.References).To(HaveLen(2))

		refVersions := make(map[string]string, len(descriptor.Component.References))
		for _, ref := range descriptor.Component.References {
			refVersions[ref.Component] = ref.Version
		}
		Expect(refVersions).To(HaveKeyWithValue(svc1, "1.2.0"))
		Expect(refVersions).To(HaveKeyWithValue(svc2, "3.1.0"))
	})

	It("should report NoDriftDetected when the vector already matches", func() {
		svc1 := "konfidence.io/sample/nodrift/service1"
		svc2 := "konfidence.io/sample/nodrift/service2"
		vectorName := "konfidence.io/sample/vectors/nodrift-test"

		By("creating mock component descriptors in Zot")
		pushComponent(ctx, ocmClient, registryEndpoint, svc1, "1.0.0")
		pushComponent(ctx, ocmClient, registryEndpoint, svc2, "2.0.0")

		By("creating a mock vector with matching versions")
		pushVector(ctx, ocmClient, registryEndpoint, vectorName, "2026.1.1-000000000Z", []vectorArtifact{
			{Name: svc1, Version: "1.0.0"},
			{Name: svc2, Version: "2.0.0"},
		})

		By("creating a VectorTemplate CR")
		vectorTemplate := newVectorTemplateCR("nodrift-test", testNamespace, registryEndpoint, vectorName, []string{svc1, svc2}, nil)
		Expect(k8sClient.Create(ctx, vectorTemplate)).To(Succeed())

		By("verifying CR status shows NoDriftDetected")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)).To(Succeed())
			statusCondition := meta.FindStatusCondition(vectorTemplate.Status.Conditions, global.VectorTemplateReadyCondition)
			g.Expect(statusCondition).NotTo(BeNil(), "Ready condition should be set")
			g.Expect(statusCondition.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(statusCondition.Reason).To(Equal(global.VectorTemplateNoDriftDetectedReason))
			g.Expect(statusCondition.ObservedGeneration).To(Equal(vectorTemplate.Generation))
		}, timeout, interval).Should(Succeed())

		By("verifying vector in Zot still has the original version")
		descriptor, err := getDescriptorFromRegistry(ctx, ocmClient, registryEndpoint, vectorName)
		Expect(err).NotTo(HaveOccurred(), "failed to get vector descriptor from registry")
		Expect(descriptor.Component.Version).To(Equal("2026.1.1-000000000Z"), "vector version should not have changed")
	})

	It("should create a new vector when no vector exists yet", func() {
		svc1 := "konfidence.io/sample/firstcreate/service1"
		svc2 := "konfidence.io/sample/firstcreate/service2"
		vectorName := "konfidence.io/sample/vectors/first-test"

		By("creating mock component descriptors (no existing vector)")
		pushComponent(ctx, ocmClient, registryEndpoint, svc1, "1.0.0")
		pushComponent(ctx, ocmClient, registryEndpoint, svc2, "2.0.0")

		By("creating a VectorTemplate CR")
		vectorTemplate := newVectorTemplateCR("first-create-test", testNamespace, registryEndpoint, vectorName, []string{svc1, svc2}, nil)
		Expect(k8sClient.Create(ctx, vectorTemplate)).To(Succeed())

		By("verifying CR status shows VectorCreated")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)).To(Succeed())
			statusCondition := meta.FindStatusCondition(vectorTemplate.Status.Conditions, global.VectorTemplateReadyCondition)
			g.Expect(statusCondition).NotTo(BeNil(), "Ready condition should be set")
			g.Expect(statusCondition.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(statusCondition.Reason).To(Equal(global.VectorTemplateVectorCreatedReason))
			g.Expect(statusCondition.ObservedGeneration).To(Equal(vectorTemplate.Generation))
		}, timeout, interval).Should(Succeed())

		By("verifying vector exists in Zot")
		descriptor, err := getDescriptorFromRegistry(ctx, ocmClient, registryEndpoint, vectorName)
		Expect(err).NotTo(HaveOccurred(), "failed to get vector descriptor from registry")
		Expect(descriptor.Component.Name).To(Equal(vectorName))
		Expect(descriptor.Component.References).To(HaveLen(2))
	})

	It("should create a vector with base vector artifacts merged in", func() {
		svc1 := "konfidence.io/sample/inherit/service1"
		svc2 := "konfidence.io/sample/inherit/service2"
		svc3 := "konfidence.io/sample/inherit/service3"
		vectorName := "konfidence.io/sample/vectors/inherit-test"
		baseName := "konfidence.io/sample/vectors/base-vector"

		By("creating a mock base vector with service3")
		pushComponent(ctx, ocmClient, registryEndpoint, svc3, "0.9.0")
		pushVector(ctx, ocmClient, registryEndpoint, baseName, "2026.1.1-000000000Z", []vectorArtifact{
			{Name: svc3, Version: "0.9.0"},
		})

		By("creating mock component descriptors for service1 and service2")
		pushComponent(ctx, ocmClient, registryEndpoint, svc1, "1.2.0")
		pushComponent(ctx, ocmClient, registryEndpoint, svc2, "3.1.0")

		By("creating a VectorTemplate CR with base")
		base := baseName
		vectorTemplate := newVectorTemplateCR("inherit-test", testNamespace, registryEndpoint, vectorName, []string{svc1, svc2}, &base)
		Expect(k8sClient.Create(ctx, vectorTemplate)).To(Succeed())

		By("verifying CR status shows VectorCreated")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)).To(Succeed())
			statusCondition := meta.FindStatusCondition(vectorTemplate.Status.Conditions, global.VectorTemplateReadyCondition)
			g.Expect(statusCondition).NotTo(BeNil(), "Ready condition should be set")
			g.Expect(statusCondition.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(statusCondition.Reason).To(Equal(global.VectorTemplateVectorCreatedReason))
			g.Expect(statusCondition.ObservedGeneration).To(Equal(vectorTemplate.Generation))
		}, timeout, interval).Should(Succeed())

		By("verifying vector in Zot contains 3 artifacts (base + components)")
		desc, err := getDescriptorFromRegistry(ctx, ocmClient, registryEndpoint, vectorName)
		Expect(err).NotTo(HaveOccurred(), "failed to get vector descriptor from registry")
		Expect(desc.Component.References).To(HaveLen(3))

		// Verify all three services are present
		refComponents := make([]string, 0, len(desc.Component.References))
		for _, ref := range desc.Component.References {
			refComponents = append(refComponents, ref.Component)
		}
		Expect(refComponents).To(ContainElements(svc1, svc2, svc3))
	})

	It("should set DriftDetectionFailed when a component does not exist in the registry", func() {
		vectorName := "konfidence.io/sample/notfound/vectors/notfound-test"
		nonExistent := "konfidence.io/sample/notfound/does-not-exist"

		By("creating a VectorTemplate CR referencing a non-existent component (no mock data)")
		vectorTemplate := newVectorTemplateCR("notfound-test", testNamespace, registryEndpoint, vectorName, []string{nonExistent}, nil)
		Expect(k8sClient.Create(ctx, vectorTemplate)).To(Succeed())

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
		svc1 := "konfidence.io/sample/dedup/service1"
		svc2 := "konfidence.io/sample/dedup/service2"
		vectorName := "konfidence.io/sample/vectors/dedup-test"

		By("creating mock component descriptors")
		pushComponent(ctx, ocmClient, registryEndpoint, svc1, "1.0.0")
		pushComponent(ctx, ocmClient, registryEndpoint, svc2, "2.0.0")

		By("creating a VectorTemplate CR with duplicate components")
		components := []string{svc1, svc1, svc2, svc1, svc2}
		vectorTemplate := newVectorTemplateCR("dedup-test", testNamespace, registryEndpoint, vectorName, components, nil)
		Expect(k8sClient.Create(ctx, vectorTemplate)).To(Succeed())

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
		desc, err := getDescriptorFromRegistry(ctx, ocmClient, registryEndpoint, vectorName)
		Expect(err).NotTo(HaveOccurred(), "failed to get vector descriptor from registry")
		Expect(desc.Component.References).To(HaveLen(2), fmt.Sprintf("expected 2 deduplicated references, got %d", len(desc.Component.References)))
	})
})
