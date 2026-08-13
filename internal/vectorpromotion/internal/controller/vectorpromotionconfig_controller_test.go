package controller

import (
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// These specs run against the manager-registered config reconciler: the value
// under test is the watch wiring itself, so everything is driven through API
// writes and observed with Eventually. The execution controller is not
// registered, so created promotions stay pending and assertions are stable.
var _ = Describe("VectorPromotionConfig drift detection", Ordered, Serial, func() {

	BeforeEach(func() { cleanupPromotions() })

	createTemplate := func(name string) *konfidence.VectorTemplate {
		template := &konfidence.VectorTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
			Spec: konfidence.VectorTemplateSpec{
				UploadTarget: "registry.example/upload",
				Components:   []konfidence.Component{{Name: "component"}},
			},
		}
		ExpectWithOffset(1, k8sClient.Create(ctx, template)).To(Succeed())
		DeferCleanup(func() {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, template))).To(Succeed())
		})
		return template
	}

	setLatestVector := func(template *konfidence.VectorTemplate, vector string) {
		EventuallyWithOffset(1, func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: template.Name, Namespace: testNamespace,
			}, template)).To(Succeed())
			original := template.DeepCopy()
			template.Status.LatestVector = vector
			g.Expect(k8sClient.Status().Patch(ctx, template, client.MergeFrom(original))).To(Succeed())
		}, timeout, interval).Should(Succeed())
	}

	promotionsOf := func(g Gomega, configName string) []konfidence.VectorPromotion {
		list := &konfidence.VectorPromotionList{}
		g.Expect(k8sClient.List(ctx, list, client.InNamespace(testNamespace))).To(Succeed())
		matching := []konfidence.VectorPromotion{}
		for _, item := range list.Items {
			if item.Spec.VectorPromotionConfigName == configName {
				matching = append(matching, item)
			}
		}
		return matching
	}

	It("creates an auto-approval promotion when the template's vector drifts", func() {
		createLandscapeWithNamespace("drift-auto-landscape", "kden-l-drift-auto")
		createStage("kden-l-drift-auto", "drift-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		template := createTemplate("drift-template")
		config := &konfidence.VectorPromotionConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "drift-auto-config", Namespace: testNamespace},
			Spec: konfidence.VectorPromotionConfigSpec{
				Source:           templateSource("drift-template"),
				Target:           stageTargetInLandscape("drift-stage", "drift-auto-landscape"),
				TTLAfterFinished: &metav1.Duration{Duration: time.Hour},
			},
		}
		Expect(k8sClient.Create(ctx, config)).To(Succeed())

		By("publishing a new latest vector on the template")
		setLatestVector(template, testVector)

		By("a sequence-stamped, owned, auto-approval promotion appears")
		Eventually(func(g Gomega) {
			promotions := promotionsOf(g, config.Name)
			g.Expect(promotions).To(HaveLen(1))
			created := promotions[0]
			g.Expect(created.Spec.Vector).To(Equal(testVector))
			g.Expect(created.Spec.RequireApproval).To(BeFalse())
			g.Expect(created.Spec.Sequence).To(Equal(int64(1)))
			g.Expect(created.Spec.TTLAfterFinished).NotTo(BeNil())
			g.Expect(created.Spec.TTLAfterFinished.Duration).To(Equal(time.Hour))
			g.Expect(created.Spec.Source).To(Equal(config.Spec.Source))
			g.Expect(created.Spec.Target).To(Equal(config.Spec.Target))
			g.Expect(metav1.IsControlledBy(&created, config)).To(BeTrue())
		}, timeout, interval).Should(Succeed())

		By("the config is Ready with the sequence recorded")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, config)).To(Succeed())
			g.Expect(config.Status.Sequence).To(Equal(int64(1)))
			ready := configReadyCondition(config)
			g.Expect(ready).NotTo(BeNil())
			g.Expect(ready.Status).To(Equal(metav1.ConditionTrue))
		}, timeout, interval).Should(Succeed())

		By("re-observing the same vector does not create a duplicate")
		setLatestVector(template, testVector)
		Consistently(func(g Gomega) {
			g.Expect(promotionsOf(g, config.Name)).To(HaveLen(1))
		}, 3*time.Second, interval).Should(Succeed())

		By("a second drift creates the next sequence")
		setLatestVector(template, "registry.example//konfidence.io/promo/app:2.0.0")
		Eventually(func(g Gomega) {
			promotions := promotionsOf(g, config.Name)
			g.Expect(promotions).To(HaveLen(2))
			sequences := []int64{promotions[0].Spec.Sequence, promotions[1].Spec.Sequence}
			g.Expect(sequences).To(ConsistOf(int64(1), int64(2)))
		}, timeout, interval).Should(Succeed())
	})

	It("creates an approval-gated promotion when the source is a stage", func() {
		createLandscapeWithNamespace("drift-src-landscape", "kden-l-drift-src")
		createLandscapeWithNamespace("drift-dst-landscape", "kden-l-drift-dst")
		createStage("kden-l-drift-src", "source-stage", testVector)
		createStage("kden-l-drift-dst", "target-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		config := createConfig("drift-manual-config",
			konfidence.PromotionSourceReference{
				Kind: konfidence.StageKind, Name: "source-stage", Landscape: "drift-src-landscape",
			},
			stageTargetInLandscape("target-stage", "drift-dst-landscape"))

		Eventually(func(g Gomega) {
			promotions := promotionsOf(g, config.Name)
			g.Expect(promotions).To(HaveLen(1))
			g.Expect(promotions[0].Spec.RequireApproval).To(BeTrue())
			g.Expect(promotions[0].Spec.Vector).To(Equal(testVector))
		}, timeout, interval).Should(Succeed())
	})

	It("recovers the Ready condition when the missing target appears", func() {
		createLandscapeWithNamespace("drift-ready-landscape", "kden-l-drift-ready")
		template := createTemplate("ready-template")
		setLatestVector(template, testVector)
		config := createConfig("drift-ready-config",
			templateSource("ready-template"),
			stageTargetInLandscape("late-ready-stage", "drift-ready-landscape"))

		By("the config reports the missing stage")
		Eventually(func(g Gomega) {
			ready := configReadyCondition(config)
			g.Expect(ready).NotTo(BeNil())
			g.Expect(ready.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(ready.Reason).To(Equal(konfidence.VectorPromotionConfigStageNotFoundReason))
		}, timeout, interval).Should(Succeed())

		By("creating the stage flips Ready without any config change")
		createStage("kden-l-drift-ready", "late-ready-stage", testVector)
		Eventually(func(g Gomega) {
			ready := configReadyCondition(config)
			g.Expect(ready).NotTo(BeNil())
			g.Expect(ready.Status).To(Equal(metav1.ConditionTrue))
		}, timeout, interval).Should(Succeed())

		By("no promotion was created because source and target match")
		Consistently(func(g Gomega) {
			g.Expect(promotionsOf(g, config.Name)).To(BeEmpty())
		}, 3*time.Second, interval).Should(Succeed())
	})

	It("reports a missing source on the config", func() {
		createLandscapeWithNamespace("drift-nosrc-landscape", "kden-l-drift-nosrc")
		createStage("kden-l-drift-nosrc", "nosrc-stage", testVector)
		config := createConfig("drift-nosrc-config",
			templateSource("absent-template"),
			stageTargetInLandscape("nosrc-stage", "drift-nosrc-landscape"))

		Eventually(func(g Gomega) {
			ready := configReadyCondition(config)
			g.Expect(ready).NotTo(BeNil())
			g.Expect(ready.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(ready.Reason).To(Equal(konfidence.VectorPromotionConfigSourceNotFoundReason))
		}, timeout, interval).Should(Succeed())
	})

	It("creates a new promotion when the target changes while a promotion for the same vector is still active", func() {
		createLandscapeWithNamespace("drift-retarget-landscape", "kden-l-drift-retarget")
		createStage("kden-l-drift-retarget", "old-target-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		createStage("kden-l-drift-retarget", "new-target-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		template := createTemplate("retarget-template")
		setLatestVector(template, testVector)

		config := createConfig("drift-retarget-config",
			templateSource("retarget-template"),
			stageTargetInLandscape("old-target-stage", "drift-retarget-landscape"))

		By("a promotion is created for the original target")
		Eventually(func(g Gomega) {
			promotions := promotionsOf(g, config.Name)
			g.Expect(promotions).To(HaveLen(1))
			g.Expect(promotions[0].Spec.Vector).To(Equal(testVector))
			g.Expect(promotions[0].Spec.Target.Name).To(Equal("old-target-stage"))
		}, timeout, interval).Should(Succeed())

		By("changing the config target to a different stage that also needs the vector")
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: config.Name, Namespace: testNamespace,
			}, config)).To(Succeed())
			original := config.DeepCopy()
			config.Spec.Target = stageTargetInLandscape("new-target-stage", "drift-retarget-landscape")
			g.Expect(k8sClient.Patch(ctx, config, client.MergeFrom(original))).To(Succeed())
		}, timeout, interval).Should(Succeed())

		By("a second promotion targeting the new stage must be created")
		Eventually(func(g Gomega) {
			promotions := promotionsOf(g, config.Name)
			g.Expect(promotions).To(HaveLen(2))
			targets := []string{promotions[0].Spec.Target.Name, promotions[1].Spec.Target.Name}
			g.Expect(targets).To(ConsistOf("old-target-stage", "new-target-stage"))
			g.Expect(promotions[0].Spec.Vector).To(Equal(testVector))
			g.Expect(promotions[1].Spec.Vector).To(Equal(testVector))
		}, timeout, interval).Should(Succeed())
	})

	It("triggers on target stage edits through the cross-namespace watch", func() {
		createLandscapeWithNamespace("drift-edit-landscape", "kden-l-drift-edit")
		stage := createStage("kden-l-drift-edit", "edit-stage", testVector)
		template := createTemplate("edit-template")
		setLatestVector(template, testVector)
		config := createConfig("drift-edit-config",
			templateSource("edit-template"),
			stageTargetInLandscape("edit-stage", "drift-edit-landscape"))

		By("no drift while source and target match")
		Consistently(func(g Gomega) {
			g.Expect(promotionsOf(g, config.Name)).To(BeEmpty())
		}, 3*time.Second, interval).Should(Succeed())

		By("hand-editing the target stage away from the source re-creates drift")
		original := stage.DeepCopy()
		stage.Spec.Vector = "registry.example//konfidence.io/promo/app:0.1.0"
		Expect(k8sClient.Patch(ctx, stage, client.MergeFrom(original))).To(Succeed())

		Eventually(func(g Gomega) {
			promotions := promotionsOf(g, config.Name)
			g.Expect(promotions).To(HaveLen(1))
			g.Expect(promotions[0].Spec.Vector).To(Equal(testVector))
		}, timeout, interval).Should(Succeed())
	})
})
