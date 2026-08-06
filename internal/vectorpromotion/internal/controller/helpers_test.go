package controller

import (
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	testNamespace = "default"
	timeout       = 30 * time.Second
	interval      = 250 * time.Millisecond

	testVector    = "registry.example//konfidence.io/promo/app:1.0.0"
	testLandscape = "test-landscape"
)

func templateSource(name string) konfidence.PromotionSourceReference {
	return konfidence.PromotionSourceReference{Kind: konfidence.VectorTemplateKind, Name: name}
}

func stageSource(name string) konfidence.PromotionSourceReference {
	return konfidence.PromotionSourceReference{Kind: konfidence.StageKind, Name: name, Landscape: testLandscape}
}

func stageTarget(name string) konfidence.PromotionTargetReference {
	return stageTargetInLandscape(name, testLandscape)
}

func stageTargetInLandscape(name, landscape string) konfidence.PromotionTargetReference {
	return konfidence.PromotionTargetReference{Kind: konfidence.StageKind, Name: name, Landscape: landscape}
}

// createLandscapeWithNamespace creates a namespace, a Landscape in the test
// namespace, and points the Landscape's status at the created namespace, as
// the landscape controller (not running here) would.
func createLandscapeWithNamespace(name, namespace string) {
	ExpectWithOffset(1, k8sClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespace},
	})).To(Succeed())
	landscape := &konfidence.Landscape{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
	}
	ExpectWithOffset(1, k8sClient.Create(ctx, landscape)).To(Succeed())
	original := landscape.DeepCopy()
	landscape.Status.Namespace = namespace
	ExpectWithOffset(1, k8sClient.Status().Patch(ctx, landscape, client.MergeFrom(original))).To(Succeed())
	DeferCleanup(func() {
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, landscape))).To(Succeed())
	})
}

// createStage creates a Stage with the given vector in the given namespace.
func createStage(namespace, name, vector string) *konfidence.Stage {
	stage := &konfidence.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       konfidence.StageSpec{Vector: vector},
	}
	ExpectWithOffset(1, k8sClient.Create(ctx, stage)).To(Succeed())
	DeferCleanup(func() {
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, stage))).To(Succeed())
	})
	return stage
}

// cleanupPromotions deletes all VectorPromotion and VectorPromotionConfig objects and waits for them to be gone.
func cleanupPromotions() {
	Expect(k8sClient.DeleteAllOf(ctx, &konfidence.VectorPromotion{}, client.InNamespace(testNamespace))).To(Succeed())
	Expect(k8sClient.DeleteAllOf(ctx, &konfidence.VectorPromotionConfig{}, client.InNamespace(testNamespace))).To(Succeed())
	Eventually(func(g Gomega) {
		promotions := &konfidence.VectorPromotionList{}
		g.Expect(k8sClient.List(ctx, promotions, client.InNamespace(testNamespace))).To(Succeed())
		g.Expect(promotions.Items).To(BeEmpty())
		configs := &konfidence.VectorPromotionConfigList{}
		g.Expect(k8sClient.List(ctx, configs, client.InNamespace(testNamespace))).To(Succeed())
		g.Expect(configs.Items).To(BeEmpty())
	}, timeout, interval).Should(Succeed())
}

// createConfig creates a VectorPromotionConfig with structured references in the test namespace.
func createConfig(
	name string,
	source konfidence.PromotionSourceReference,
	target konfidence.PromotionTargetReference,
) *konfidence.VectorPromotionConfig {
	config := &konfidence.VectorPromotionConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: konfidence.VectorPromotionConfigSpec{
			Source: source,
			Target: target,
		},
	}
	ExpectWithOffset(1, k8sClient.Create(ctx, config)).To(Succeed())
	return config
}

// createPromotion creates a VectorPromotion in the test namespace referencing a config.
func createPromotion(name, configRef string) *konfidence.VectorPromotion {
	promotion := &konfidence.VectorPromotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: konfidence.VectorPromotionSpec{
			VectorPromotionConfigRef: configRef,
			Source:                   templateSource("some-template"),
			Target:                   stageTarget("some-stage"),
			Vector:                   testVector,
		},
	}
	ownPromotionIfConfigExists(promotion, configRef)
	ExpectWithOffset(1, k8sClient.Create(ctx, promotion)).To(Succeed())
	return promotion
}

// createConfigWithRetention creates a VectorPromotionConfig with a retention bound.
func createConfigWithRetention(name string, keep int32) *konfidence.VectorPromotionConfig {
	config := &konfidence.VectorPromotionConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: konfidence.VectorPromotionConfigSpec{
			Source:             templateSource("some-template"),
			Target:             stageTarget("some-stage"),
			KeepLastPromotions: &keep,
		},
	}
	ExpectWithOffset(1, k8sClient.Create(ctx, config)).To(Succeed())
	return config
}

// ownPromotionIfConfigExists mirrors the drift controller's owner reference so
// Owns() aggregation triggers work for test-created promotions too.
func ownPromotionIfConfigExists(promotion *konfidence.VectorPromotion, configRef string) {
	config := &konfidence.VectorPromotionConfig{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: configRef, Namespace: testNamespace}, config)
	if err != nil {
		return
	}
	ExpectWithOffset(2, controllerutil.SetControllerReference(config, promotion, scheme.Scheme)).To(Succeed())
}

// createPromotionTargeting creates a VectorPromotion carrying a real target
// snapshot, for specs that execute against an actual Stage.
func createPromotionTargeting(name, configRef string, target konfidence.PromotionTargetReference, requireApproval bool) *konfidence.VectorPromotion {
	promotion := &konfidence.VectorPromotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: konfidence.VectorPromotionSpec{
			VectorPromotionConfigRef: configRef,
			Source:                   templateSource("some-template"),
			Target:                   target,
			Vector:                   testVector,
			RequireApproval:          requireApproval,
		},
	}
	ownPromotionIfConfigExists(promotion, configRef)
	ExpectWithOffset(1, k8sClient.Create(ctx, promotion)).To(Succeed())
	return promotion
}

// createPromotionRequiringApproval creates a VectorPromotion with requireApproval set.
func createPromotionRequiringApproval(name, configRef string) *konfidence.VectorPromotion {
	promotion := &konfidence.VectorPromotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: konfidence.VectorPromotionSpec{
			VectorPromotionConfigRef: configRef,
			Source:                   templateSource("some-template"),
			Target:                   stageTarget("some-stage"),
			Vector:                   testVector,
			RequireApproval:          true,
		},
	}
	ownPromotionIfConfigExists(promotion, configRef)
	ExpectWithOffset(1, k8sClient.Create(ctx, promotion)).To(Succeed())
	return promotion
}

// approvePromotion sets the Approved condition, as the konfidence API would.
func approvePromotion(promotion *konfidence.VectorPromotion) {
	EventuallyWithOffset(1, func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: promotion.Name, Namespace: testNamespace,
		}, promotion)).To(Succeed())
		meta.SetStatusCondition(&promotion.Status.Conditions, metav1.Condition{
			Type:    konfidence.ConditionTypeApproved,
			Status:  metav1.ConditionTrue,
			Reason:  konfidence.ReasonPromotionManuallyApproved,
			Message: "approved by test",
		})
		g.Expect(k8sClient.Status().Update(ctx, promotion)).To(Succeed())
	}, timeout, interval).Should(Succeed())
}

// createPromotionWithTTL creates a VectorPromotion with TTLAfterFinished set.
func createPromotionWithTTL(name, configRef string, ttl time.Duration) *konfidence.VectorPromotion {
	promotion := &konfidence.VectorPromotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: konfidence.VectorPromotionSpec{
			VectorPromotionConfigRef: configRef,
			Source:                   templateSource("some-template"),
			Target:                   stageTarget("some-stage"),
			Vector:                   testVector,
			TTLAfterFinished:         &metav1.Duration{Duration: ttl},
		},
	}
	ExpectWithOffset(1, k8sClient.Create(ctx, promotion)).To(Succeed())
	return promotion
}

// setSucceededCondition writes the Succeeded condition on a promotion directly.
// The execution controller is not registered in this suite; tests drive
// promotion conditions manually to exercise the TTL and status propagation
// controllers.
func setSucceededCondition(
	promotion *konfidence.VectorPromotion,
	status metav1.ConditionStatus,
	reason string,
	transitionTime time.Time,
) {
	EventuallyWithOffset(1, func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: promotion.Name, Namespace: testNamespace,
		}, promotion)).To(Succeed())
		meta.SetStatusCondition(&promotion.Status.Conditions, metav1.Condition{
			Type:    konfidence.ConditionTypeSucceeded,
			Status:  status,
			Reason:  reason,
			Message: "set by test",
		})
		// meta.SetStatusCondition stamps its own transition time; force the test-controlled one.
		cond := meta.FindStatusCondition(promotion.Status.Conditions, konfidence.ConditionTypeSucceeded)
		cond.LastTransitionTime = metav1.NewTime(transitionTime.Truncate(time.Second))
		g.Expect(k8sClient.Status().Update(ctx, promotion)).To(Succeed())
	}, timeout, interval).Should(Succeed())
}
