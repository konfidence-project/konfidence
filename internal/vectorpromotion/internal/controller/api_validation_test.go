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
		)

		It("accepts a Stage source targeting a different Stage", func() {
			config := newConfig(
				konfidence.PromotionSourceReference{Kind: konfidence.StageKind, Name: "stage-a"},
				stageTarget("stage-b"),
			)
			Expect(k8sClient.Create(ctx, config)).To(Succeed())
		})

		It("defaults apiGroup on both references", func() {
			config := newConfig(templateSource("some-template"), stageTarget("some-stage"))
			Expect(k8sClient.Create(ctx, config)).To(Succeed())
			Expect(config.Spec.Source.APIGroup).To(Equal("konfidence.cloud"))
			Expect(config.Spec.Target.APIGroup).To(Equal("konfidence.cloud"))
		})

		It("allows repointing source and target", func() {
			config := newConfig(templateSource("some-template"), stageTarget("some-stage"))
			Expect(k8sClient.Create(ctx, config)).To(Succeed())

			patch := client.MergeFrom(config.DeepCopy())
			config.Spec.Source = konfidence.PromotionSourceReference{Kind: konfidence.StageKind, Name: "other-stage"}
			config.Spec.Target = stageTarget("another-stage")
			Expect(k8sClient.Patch(ctx, config, patch)).To(Succeed())
		})
	})

	Describe("VectorPromotion", func() {
		It("rejects a vector that is not a component version reference", func() {
			promotion := &konfidence.VectorPromotion{
				ObjectMeta: metav1.ObjectMeta{Name: "schema-bad-vector", Namespace: testNamespace},
				Spec: konfidence.VectorPromotionSpec{
					VectorPromotionConfigRef: "some-config",
					Vector:                   "not-a-valid-reference",
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

		It("rejects updates to requireApproval", func() {
			promotion := &konfidence.VectorPromotion{
				ObjectMeta: metav1.ObjectMeta{Name: "schema-immutable-approval", Namespace: testNamespace},
				Spec: konfidence.VectorPromotionSpec{
					VectorPromotionConfigRef: "some-config",
					Vector:                   testVector,
					RequireApproval:          true,
				},
			}
			Expect(k8sClient.Create(ctx, promotion)).To(Succeed())

			patch := client.MergeFrom(promotion.DeepCopy())
			promotion.Spec.RequireApproval = false
			Expect(k8sClient.Patch(ctx, promotion, patch)).To(
				MatchError(ContainSubstring("requireApproval is immutable after it has been set")))
		})
	})
})
