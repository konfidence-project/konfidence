package controller

import (
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Promotion CRD schema validation", Ordered, Serial, func() {

	BeforeEach(func() { cleanupPromotions() })

	Describe("VectorPromotionConfig", func() {
		newConfig := func(source konfidence.PromotionSourceReference, target konfidence.PromotionTargetReference) *konfidence.VectorPromotionConfig {
			return &konfidence.VectorPromotionConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "schema-config", Namespace: testNamespace},
				Spec:       konfidence.VectorPromotionConfigSpec{Source: source, Target: target},
			}
		}

		DescribeTable("rejects invalid specs",
			func(config *konfidence.VectorPromotionConfig, expectedErr string) {
				Expect(k8sClient.Create(ctx, config)).To(MatchError(ContainSubstring(expectedErr)))
			},
			Entry("target kind other than Stage",
				&konfidence.VectorPromotionConfig{
					ObjectMeta: metav1.ObjectMeta{Name: "schema-bad-target-kind", Namespace: testNamespace},
					Spec: konfidence.VectorPromotionConfigSpec{
						Source: templateSource("some-template"),
						Target: konfidence.PromotionTargetReference{Kind: "VectorTemplate", Name: "some-template"},
					},
				}, "Unsupported value"),
			Entry("source kind other than VectorTemplate or Stage",
				&konfidence.VectorPromotionConfig{
					ObjectMeta: metav1.ObjectMeta{Name: "schema-bad-source-kind", Namespace: testNamespace},
					Spec: konfidence.VectorPromotionConfigSpec{
						Source: konfidence.PromotionSourceReference{Kind: "Deployment", Name: "some-deployment"},
						Target: stageTarget("some-stage"),
					},
				}, "Unsupported value"),
			Entry("Stage source without landscape",
				&konfidence.VectorPromotionConfig{
					ObjectMeta: metav1.ObjectMeta{Name: "schema-stage-source-no-landscape", Namespace: testNamespace},
					Spec: konfidence.VectorPromotionConfigSpec{
						Source: konfidence.PromotionSourceReference{Kind: konfidence.StageKind, Name: "stage-a"},
						Target: stageTarget("stage-b"),
					},
				}, "landscape is required for Stage references"),
			Entry("VectorTemplate source with landscape",
				&konfidence.VectorPromotionConfig{
					ObjectMeta: metav1.ObjectMeta{Name: "schema-template-source-landscape", Namespace: testNamespace},
					Spec: konfidence.VectorPromotionConfigSpec{
						Source: konfidence.PromotionSourceReference{
							Kind: konfidence.VectorTemplateKind, Name: "some-template", Landscape: testLandscape,
						},
						Target: stageTarget("some-stage"),
					},
				}, "must be omitted for VectorTemplate references"),
			Entry("target without landscape",
				&konfidence.VectorPromotionConfig{
					ObjectMeta: metav1.ObjectMeta{Name: "schema-target-no-landscape", Namespace: testNamespace},
					Spec: konfidence.VectorPromotionConfigSpec{
						Source: templateSource("some-template"),
						Target: konfidence.PromotionTargetReference{Kind: konfidence.StageKind, Name: "some-stage"},
					},
				}, "spec.target.landscape"),
		)

		It("accepts a Stage source targeting a different Stage", func() {
			config := newConfig(stageSource("stage-a"), stageTarget("stage-b"))
			Expect(k8sClient.Create(ctx, config)).To(Succeed())
		})

		It("allows repointing source and target", func() {
			config := newConfig(templateSource("some-template"), stageTarget("some-stage"))
			Expect(k8sClient.Create(ctx, config)).To(Succeed())

			patch := client.MergeFrom(config.DeepCopy())
			config.Spec.Source = stageSource("other-stage")
			config.Spec.Target = stageTarget("another-stage")
			Expect(k8sClient.Patch(ctx, config, patch)).To(Succeed())
		})
	})

	Describe("VectorPromotion", func() {
		It("rejects an empty vector", func() {
			promotion := &konfidence.VectorPromotion{
				ObjectMeta: metav1.ObjectMeta{Name: "schema-empty-vector", Namespace: testNamespace},
				Spec: konfidence.VectorPromotionSpec{
					VectorPromotionConfigName: "some-config",
					Source:                    templateSource("some-template"),
					Target:                    stageTarget("some-stage"),
				},
			}
			Expect(k8sClient.Create(ctx, promotion)).To(MatchError(ContainSubstring("spec.vector")))
		})

		It("rejects updates to vector", func() {
			promotion := createPromotion("schema-immutable-vector", "some-config")

			patch := client.MergeFrom(promotion.DeepCopy())
			promotion.Spec.Vector = "registry.example//konfidence.io/promo/app:2.0.0"
			Expect(k8sClient.Patch(ctx, promotion, patch)).To(
				MatchError(ContainSubstring("vector is immutable after it has been set")))
		})

		It("rejects changes to gates", func() {
			promotion := createPromotionRequiringApproval("schema-immutable-gates", "some-config")

			By("changing a gate type")
			patch := client.MergeFrom(promotion.DeepCopy())
			promotion.Spec.Gates = []konfidence.PromotionGate{{Type: "e2e-tests.example.com"}}
			Expect(k8sClient.Patch(ctx, promotion, patch)).To(
				MatchError(ContainSubstring("gates are immutable after they have been set")))

			By("adding a gate")
			refreshPromotion(promotion)
			patch = client.MergeFrom(promotion.DeepCopy())
			promotion.Spec.Gates = append(promotion.Spec.Gates, konfidence.PromotionGate{Type: "e2e-tests.example.com"})
			Expect(k8sClient.Patch(ctx, promotion, patch)).To(
				MatchError(ContainSubstring("gates are immutable after they have been set")))

			By("emptying the gate list")
			refreshPromotion(promotion)
			patch = client.MergeFrom(promotion.DeepCopy())
			promotion.Spec.Gates = []konfidence.PromotionGate{}
			Expect(k8sClient.Patch(ctx, promotion, patch)).To(
				MatchError(ContainSubstring("gates are immutable after they have been set")))
		})

		It("rejects adding gates to a gateless promotion", func() {
			promotion := createPromotion("schema-immutable-no-gates", "some-config")

			patch := client.MergeFrom(promotion.DeepCopy())
			promotion.Spec.Gates = approvalGate()
			Expect(k8sClient.Patch(ctx, promotion, patch)).To(
				MatchError(ContainSubstring("gates are immutable after they have been set")))
		})

		DescribeTable("validates gate types",
			func(gateType string, valid bool) {
				promotion := &konfidence.VectorPromotion{
					ObjectMeta: metav1.ObjectMeta{GenerateName: "schema-gate-type-", Namespace: testNamespace},
					Spec: konfidence.VectorPromotionSpec{
						VectorPromotionConfigName: "some-config",
						Source:                    templateSource("some-template"),
						Target:                    stageTarget("some-stage"),
						Vector:                    testVector,
						Gates:                     []konfidence.PromotionGate{{Type: gateType}},
					},
				}
				err := k8sClient.Create(ctx, promotion)
				if valid {
					Expect(err).NotTo(HaveOccurred())
				} else {
					Expect(err).To(MatchError(ContainSubstring("spec.gates[0].type")))
				}
			},
			Entry("core gate", konfidence.GateTypeExplicitApproval, true),
			Entry("third-party gate", "e2e-tests.example.com", true),
			Entry("missing owner domain", "explicit-approval", false),
			Entry("domain without gate name", "konfidence.cloud", false),
			Entry("legacy slash notation", "konfidence.cloud/ExplicitApproval", false),
			Entry("uppercase", "Explicit-Approval.konfidence.cloud", false),
		)

		It("rejects duplicate gate types", func() {
			promotion := &konfidence.VectorPromotion{
				ObjectMeta: metav1.ObjectMeta{Name: "schema-duplicate-gates", Namespace: testNamespace},
				Spec: konfidence.VectorPromotionSpec{
					VectorPromotionConfigName: "some-config",
					Source:                    templateSource("some-template"),
					Target:                    stageTarget("some-stage"),
					Vector:                    testVector,
					Gates:                     append(approvalGate(), approvalGate()...),
				},
			}
			Expect(k8sClient.Create(ctx, promotion)).To(MatchError(ContainSubstring("Duplicate value")))
		})
	})
})
