package controller

import (
	"fmt"
	"time"

	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNamespace = "default"
	timeout       = 30 * time.Second
	interval      = 250 * time.Millisecond
)

var _ = Describe("VectorPromotion controller tests", Ordered, Serial, func() {

	BeforeEach(func() {
		Expect(k8sClient.DeleteAllOf(ctx, &global.VectorPromotion{}, client.InNamespace(testNamespace))).To(Succeed())
		Expect(k8sClient.DeleteAllOf(ctx, &global.VectorPromotionConfig{}, client.InNamespace(testNamespace))).To(Succeed())
		Eventually(func(g Gomega) {
			list := &global.VectorPromotionList{}
			g.Expect(k8sClient.List(ctx, list, client.InNamespace(testNamespace))).To(Succeed())
			g.Expect(list.Items).To(BeEmpty())
		}, timeout, interval).Should(Succeed())
	})

	//nolint:dupl // Test cases are intentionally similar but test different scenarios
	It("should successfully promote cross-registry", func() {
		By("pushing component to source registry")
		ref := sourceRef("konfidence.io/promo/svc1:v1.0.0")
		pushComponent(ctx, ref, new("latest"))

		By("creating VectorPromotionConfig")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/svc1:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/svc1:promoted", targetRegistryEndpoint)
		config := createConfig("cross-registry-config", source, target)

		By("creating VectorPromotion")
		promotion := createPromotion("cross-registry-promotion", config.Name)

		By("asserting promotion succeeded")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(cond.Reason).To(Equal(global.ReasonPromotionSucceeded))
		}, timeout, interval).Should(Succeed())

		By("verifying component exists in target registry with alias")
		targetCompRef := targetRef("konfidence.io/promo/svc1:promoted")
		desc, err := ocmClient.Get(ctx, targetCompRef)
		Expect(err).NotTo(HaveOccurred(), "component should be accessible in target registry")
		Expect(desc.Component.Version).To(Equal("v1.0.0"))
	})

	//nolint:dupl // Test cases are intentionally similar but test different scenarios
	It("should successfully promote within the same registry (alias-only, no copy)", func() {
		By("pushing component to source registry")
		ref := sourceRef("konfidence.io/promo/same:v2.0.0")
		pushComponent(ctx, ref, new("latest"))

		By("creating VectorPromotionConfig with same registry for source and target")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/same:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/same:stable", sourceRegistryEndpoint)
		config := createConfig("same-registry-config", source, target)

		By("creating VectorPromotion")
		promotion := createPromotion("same-registry-promotion", config.Name)

		By("asserting promotion succeeded")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(cond.Reason).To(Equal(global.ReasonPromotionSucceeded))
		}, timeout, interval).Should(Succeed())

		By("verifying component exists in source registry with new alias")
		stableRef := sourceRef("konfidence.io/promo/same:stable")
		desc, err := ocmClient.Get(ctx, stableRef)
		Expect(err).NotTo(HaveOccurred(), "component should be accessible with new alias")
		Expect(desc.Component.Version).To(Equal("v2.0.0"))
	})

	It("should successfully promote within same registry but different sub path (copy required)", func() {
		By("pushing component to source registry")
		ref := sourceRef("konfidence.io/promo/subpath:v1.0.0")
		pushComponent(ctx, ref, new("latest"))

		By("creating VectorPromotionConfig with different sub path")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/subpath:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s/different-subpath//konfidence.io/promo/subpath:promoted", sourceRegistryEndpoint)
		config := createConfig("subpath-config", source, target)

		By("creating VectorPromotion")
		promotion := createPromotion("subpath-promotion", config.Name)

		By("asserting promotion succeeded")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(cond.Reason).To(Equal(global.ReasonPromotionSucceeded))
		}, timeout, interval).Should(Succeed())

		By("verifying component exists at target with different sub path")
		targetCompRef := sourceRefWithSubPath("different-subpath", "konfidence.io/promo/subpath:promoted")
		desc, err := ocmClient.Get(ctx, targetCompRef)
		Expect(err).NotTo(HaveOccurred(), "component should be accessible at different sub path")
		Expect(desc.Component.Version).To(Equal("v1.0.0"))
	})

	It("should set PromotionSourceNotFound when source component does not exist", func() {
		By("creating VectorPromotionConfig pointing to non-existent component")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/nonexistent:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/nonexistent:promoted", targetRegistryEndpoint)
		config := createConfig("source-not-found-config", source, target)

		By("creating VectorPromotion")
		promotion := createPromotion("source-not-found-promotion", config.Name)

		By("asserting PromotionSourceNotFound condition")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(cond.Reason).To(Equal(global.ReasonPromotionSourceNotFound))
		}, timeout, interval).Should(Succeed())
	})

	It("should set PromotionFailed when OCM client creation fails", func() {
		By("creating VectorPromotionConfig with credentials that trigger client creation failure")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/client-fail:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/client-fail:promoted", targetRegistryEndpoint)
		config := createConfigWithCredentials("client-fail-config", source, target, []global.CredentialsConfig{
			{Kind: "Secret", APIVersion: "v1", Name: failClientCreationSecret},
		})

		By("creating VectorPromotion")
		promotion := createPromotion("client-fail-promotion", config.Name)

		By("asserting PromotionFailed condition due to client creation failure")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(cond.Reason).To(Equal(global.ReasonPromotionFailed))
			g.Expect(cond.Message).To(ContainSubstring("failed to create OCM client"))
		}, timeout, interval).Should(Succeed())
	})

	It("should set PromotionConfigurationNotFound when config does not exist", func() {
		By("creating VectorPromotion referencing non-existent config")
		promotion := createPromotion("config-not-found-promotion", "non-existent-config")

		By("asserting PromotionConfigurationNotFound condition")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(cond.Reason).To(Equal(global.ReasonPromotionConfigurationNotFound))
		}, timeout, interval).Should(Succeed())
	})

	It("should set InvalidPromotionConfiguration when target uses semver version", func() {
		By("pushing component to source registry")
		ref := sourceRef("konfidence.io/promo/invalid:v1.0.0")
		pushComponent(ctx, ref, new("latest"))

		By("creating VectorPromotionConfig with semver target version")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/invalid:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/invalid:v1.0.0", targetRegistryEndpoint)
		config := createConfig("invalid-target-config", source, target)

		By("creating VectorPromotion")
		promotion := createPromotion("invalid-target-promotion", config.Name)

		By("asserting InvalidPromotionConfiguration condition")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(cond.Reason).To(Equal(global.ReasonInvalidPromotionConfiguration))
		}, timeout, interval).Should(Succeed())
	})

	It("should set InvalidPromotionConfiguration when source and target component names do not match", func() {
		By("creating VectorPromotionConfig with mismatched component names")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/app-a:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/app-b:promoted", targetRegistryEndpoint)
		config := createConfig("mismatched-components-config", source, target)

		By("creating VectorPromotion")
		promotion := createPromotion("mismatched-components-promotion", config.Name)

		By("asserting InvalidPromotionConfiguration condition")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(cond.Reason).To(Equal(global.ReasonInvalidPromotionConfiguration))
			g.Expect(cond.Message).To(ContainSubstring("source and target component names do not match"))
		}, timeout, interval).Should(Succeed())
	})

	It("should remain stable after reaching terminal state even when spec is updated", func() {
		By("pushing component to source registry")
		ref := sourceRef("konfidence.io/promo/terminal:v4.0.0")
		pushComponent(ctx, ref, new("latest"))

		By("creating VectorPromotionConfig")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/terminal:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/terminal:promoted", targetRegistryEndpoint)
		config := createConfig("terminal-config", source, target)

		By("creating VectorPromotion and waiting for success")
		promotion := createPromotion("terminal-promotion", config.Name)
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		}, timeout, interval).Should(Succeed())

		By("recording the condition's LastTransitionTime")
		cond := meta.FindStatusCondition(promotion.Status.Conditions, global.ConditionTypeSucceeded)
		originalTransitionTime := cond.LastTransitionTime

		By("patching the promotion spec to add TTLAfterFinished (triggers an Update event)")
		patch := client.MergeFrom(promotion.DeepCopy())
		promotion.Spec.TTLAfterFinished = &metav1.Duration{Duration: time.Hour}
		Expect(k8sClient.Patch(ctx, promotion, patch)).To(Succeed())

		By("asserting condition remains unchanged after the spec update")
		Consistently(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(cond.Reason).To(Equal(global.ReasonPromotionSucceeded))
			g.Expect(cond.LastTransitionTime).To(Equal(originalTransitionTime))
		}, 3*time.Second, interval).Should(Succeed())
	})

	It("should succeed when source and target aliases already point to the same version (idempotent)", func() {
		By("pushing component with both source and target aliases pointing to the same version")
		ref := sourceRef("konfidence.io/promo/idempotent:v1.0.0")
		pushComponent(ctx, ref, new("latest"))
		Expect(ocmClient.AddAlias(ctx, ref, "promoted")).To(Succeed())

		By("creating VectorPromotionConfig where source and target reference the same version via different aliases")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/idempotent:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/idempotent:promoted", sourceRegistryEndpoint)
		config := createConfig("idempotent-config", source, target)

		By("creating VectorPromotion")
		promotion := createPromotion("idempotent-promotion", config.Name)

		By("asserting promotion succeeded even though target already had the same version")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(cond.Reason).To(Equal(global.ReasonPromotionSucceeded))
		}, timeout, interval).Should(Succeed())

		By("verifying target alias still points to the same version")
		promotedRef := sourceRef("konfidence.io/promo/idempotent:promoted")
		desc, err := ocmClient.Get(ctx, promotedRef)
		Expect(err).NotTo(HaveOccurred())
		Expect(desc.Component.Version).To(Equal("v1.0.0"))
	})

	It("should re-promote when source alias is updated to a newer version", func() {
		By("pushing initial version with aliases")
		initialRef := sourceRef("konfidence.io/promo/repromote:v1.0.0")
		pushComponent(ctx, initialRef, new("latest"))

		By("creating VectorPromotionConfig")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/repromote:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/repromote:stable", sourceRegistryEndpoint)
		config := createConfig("repromote-config", source, target)

		By("creating first VectorPromotion to establish initial state")
		promotion1 := createPromotion("repromote-initial", config.Name)

		By("asserting first promotion succeeded")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion1.Name, Namespace: testNamespace,
			}, promotion1)).To(Succeed())
			cond := meta.FindStatusCondition(promotion1.Status.Conditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		}, timeout, interval).Should(Succeed())

		By("verifying stable alias points to v1.0.0")
		stableRef := sourceRef("konfidence.io/promo/repromote:stable")
		desc, err := ocmClient.Get(ctx, stableRef)
		Expect(err).NotTo(HaveOccurred())
		Expect(desc.Component.Version).To(Equal("v1.0.0"))

		By("pushing newer version and updating the latest alias")
		newerRef := sourceRef("konfidence.io/promo/repromote:v2.0.0")
		pushComponent(ctx, newerRef, new("latest"))

		By("creating second VectorPromotion to promote the newer version")
		promotion2 := createPromotion("repromote-update", config.Name)

		By("asserting second promotion succeeded")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion2.Name, Namespace: testNamespace,
			}, promotion2)).To(Succeed())
			cond := meta.FindStatusCondition(promotion2.Status.Conditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(cond.Reason).To(Equal(global.ReasonPromotionSucceeded))
		}, timeout, interval).Should(Succeed())

		By("verifying stable alias now points to the newer version v2.0.0")
		desc, err = ocmClient.Get(ctx, stableRef)
		Expect(err).NotTo(HaveOccurred())
		Expect(desc.Component.Version).To(Equal("v2.0.0"))
	})

	It("should succeed when source and target are identical (same registry, component, and alias)", func() {
		By("pushing component with alias")
		ref := sourceRef("konfidence.io/promo/identical:v1.0.0")
		pushComponent(ctx, ref, new("latest"))

		By("creating VectorPromotionConfig where source and target are 100% identical")
		source := fmt.Sprintf("http://%s//konfidence.io/promo/identical:latest", sourceRegistryEndpoint)
		target := fmt.Sprintf("http://%s//konfidence.io/promo/identical:latest", sourceRegistryEndpoint)
		config := createConfig("identical-config", source, target)

		By("creating VectorPromotion")
		promotion := createPromotion("identical-promotion", config.Name)

		By("asserting promotion succeeded (no-op but valid)")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: promotion.Name, Namespace: testNamespace,
			}, promotion)).To(Succeed())
			cond := meta.FindStatusCondition(promotion.Status.Conditions, global.ConditionTypeSucceeded)
			g.Expect(cond).NotTo(BeNil())
			g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			g.Expect(cond.Reason).To(Equal(global.ReasonPromotionSucceeded))
		}, timeout, interval).Should(Succeed())

		By("verifying alias still points to the original version")
		latestRef := sourceRef("konfidence.io/promo/identical:latest")
		desc, err := ocmClient.Get(ctx, latestRef)
		Expect(err).NotTo(HaveOccurred())
		Expect(desc.Component.Version).To(Equal("v1.0.0"))
	})
})
