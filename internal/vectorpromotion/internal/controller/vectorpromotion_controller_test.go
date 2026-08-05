package controller

import (
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
)

// The stub reconciler is exercised directly instead of through the shared
// manager: registering it there would race the manually driven conditions the
// TTL and status propagation tests rely on.
var _ = Describe("VectorPromotion execution stub", Ordered, Serial, func() {

	BeforeEach(func() { cleanupPromotions() })

	reconcile := func(name string) {
		r := &VectorPromotionReconciler{Client: k8sClient, Recorder: events.NewFakeRecorder(10)}
		_, err := r.Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Name: name, Namespace: testNamespace},
		})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}

	It("marks a pending promotion as execution pending", func() {
		promotion := createPromotion("stub-pending-promotion", "stub-config")

		reconcile(promotion.Name)

		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: promotion.Name, Namespace: testNamespace,
		}, promotion)).To(Succeed())
		cond := meta.FindStatusCondition(promotion.Status.Conditions, konfidence.ConditionTypeSucceeded)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionUnknown))
		Expect(cond.Reason).To(Equal(konfidence.ReasonPromotionExecutionPending))
		Expect(promotion.Status.State).To(Equal(konfidence.PromotionStatePending))
	})

	It("does not reconcile a promotion twice", func() {
		promotion := createPromotion("stub-idempotent-promotion", "stub-config")

		reconcile(promotion.Name)
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: promotion.Name, Namespace: testNamespace,
		}, promotion)).To(Succeed())
		resourceVersion := promotion.ResourceVersion

		reconcile(promotion.Name)
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: promotion.Name, Namespace: testNamespace,
		}, promotion)).To(Succeed())
		Expect(promotion.ResourceVersion).To(Equal(resourceVersion))
		Expect(promotion.Status.Conditions).To(HaveLen(1))
	})

	It("does not touch a promotion that already has a Succeeded condition", func() {
		promotion := createPromotion("stub-terminal-promotion", "stub-config")
		setSucceededCondition(promotion, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, time.Now())
		resourceVersion := promotion.ResourceVersion

		reconcile(promotion.Name)

		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: promotion.Name, Namespace: testNamespace,
		}, promotion)).To(Succeed())
		Expect(promotion.ResourceVersion).To(Equal(resourceVersion))
		cond := meta.FindStatusCondition(promotion.Status.Conditions, konfidence.ConditionTypeSucceeded)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Reason).To(Equal(konfidence.ReasonPromotionSucceeded))
	})
})
