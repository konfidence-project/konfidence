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

	"github.com/konfidence-project/konfidence/pkg/promotiongate"
)

var _ = Describe("VectorPromotion operator override", Ordered, Serial, func() {

	BeforeEach(func() { cleanupPromotions() })

	expectCondition := func(promotion *konfidence.VectorPromotion, conditionType string, status metav1.ConditionStatus, reason string) {
		GinkgoHelper()
		cond := meta.FindStatusCondition(promotion.Status.Conditions, conditionType)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(status))
		Expect(cond.Reason).To(Equal(reason))
	}
	reconcileWithEvents := func(name string) *events.FakeRecorder {
		GinkgoHelper()
		reconciler := newExecutionReconciler()
		_, err := reconciler.Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Name: name, Namespace: testNamespace},
		})
		Expect(err).NotTo(HaveOccurred())
		recorder, ok := reconciler.Recorder.(*events.FakeRecorder)
		Expect(ok).To(BeTrue())
		return recorder
	}

	Describe("Reject", func() {
		It("terminally rejects a waiting promotion and leaves its gates untouched", func() {
			promotion := createGatedPromotion("override-reject", "some-config", konfidence.PromotionGate{Type: e2eGate})
			reconcilePromotion(promotion.Name)
			setGate(promotion, e2eGate, konfidence.PromotionGateStateError, "Misconfigured")
			gates := promotion.DeepCopy().Status.Gates

			setOverride(promotion, konfidence.PromotionOverrideTypeReject)
			recorder := reconcileWithEvents(promotion.Name)
			refreshPromotion(promotion)

			Expect(observedState(promotion)).To(Equal("Rejected"))
			expectCondition(promotion, konfidence.ConditionTypeSucceeded, metav1.ConditionFalse, konfidence.ReasonPromotionRejected)
			Expect(meta.FindStatusCondition(promotion.Status.Conditions, konfidence.ConditionTypeSucceeded).Message).
				To(And(ContainSubstring("alice"), ContainSubstring("set by test")))
			Expect(promotion.Status.CompletionTime).NotTo(BeNil())
			Expect(promotion.Status.StartTime).To(BeNil())
			Expect(promotion.Status.Gates).To(Equal(gates))
			Expect(recorder.Events).To(Receive(ContainSubstring("Normal PromotionRejected")))

			By("the rejection is final")
			resourceVersion := promotion.ResourceVersion
			reconcilePromotion(promotion.Name)
			refreshPromotion(promotion)
			Expect(promotion.ResourceVersion).To(Equal(resourceVersion))
		})

		It("takes precedence over gates satisfied before the controller observed them", func() {
			promotion := createPromotionRequiringApproval("override-reject-precedence", "some-config")
			approvePromotion(promotion)
			setOverride(promotion, konfidence.PromotionOverrideTypeReject)

			reconcilePromotion(promotion.Name)
			refreshPromotion(promotion)

			Expect(observedState(promotion)).To(Equal("Rejected"))
			Expect(promotion.Status.StartTime).To(BeNil())
			Expect(meta.FindStatusCondition(promotion.Status.Conditions, konfidence.ConditionTypeGatesSatisfied)).To(BeNil())
		})

		It("is reaped by ttlAfterFinished like any finished promotion", func() {
			promotion := createPromotionWithTTL("override-reject-ttl", "some-config", time.Second)
			setOverride(promotion, konfidence.PromotionOverrideTypeReject)
			reconcilePromotion(promotion.Name)

			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: promotion.Name, Namespace: testNamespace}, promotion)
				g.Expect(err).To(HaveOccurred())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("BypassGates", func() {
		It("executes despite unsatisfied gates and leaves the gate entries untouched", func() {
			createLandscapeWithNamespace("override-bypass-landscape", "kden-l-override-bypass")
			stage := createStage("kden-l-override-bypass", "bypass-stage", "registry.example//konfidence.io/promo/app:0.9.0")
			promotion := createPromotionTargeting("override-bypass", "override-bypass-config",
				stageTargetInLandscape("bypass-stage", "override-bypass-landscape"), true)
			reconcilePromotion(promotion.Name)
			refreshPromotion(promotion)
			Expect(observedState(promotion)).To(Equal("Waiting"))
			gates := promotion.DeepCopy().Status.Gates

			setOverride(promotion, konfidence.PromotionOverrideTypeBypassGates)
			recorder := reconcileWithEvents(promotion.Name)
			refreshPromotion(promotion)

			Expect(observedState(promotion)).To(Equal("Succeeded"))
			expectCondition(promotion, konfidence.ConditionTypeGatesSatisfied, metav1.ConditionFalse, konfidence.ReasonGatesBypassed)
			expectCondition(promotion, konfidence.ConditionTypeSucceeded, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded)
			Expect(promotion.Status.Gates).To(Equal(gates))
			Expect(promotiongate.StateOf(promotion, konfidence.GateTypeExplicitApproval)).
				To(Equal(konfidence.PromotionGateStatePending))
			Expect(recorder.Events).To(Receive(ContainSubstring("Normal PromotionExecutionStarted")))

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: stage.Name, Namespace: stage.Namespace}, stage)).To(Succeed())
			Expect(stage.Spec.Vector).To(Equal(testVector))
		})

		It("reports genuinely satisfied gates as satisfied", func() {
			promotion := createPromotionRequiringApproval("override-bypass-satisfied", "some-config")
			approvePromotion(promotion)
			setOverride(promotion, konfidence.PromotionOverrideTypeBypassGates)

			Expect(newExecutionReconciler().reconcileGates(ctx, promotion)).To(Succeed())
			refreshPromotion(promotion)
			expectCondition(promotion, konfidence.ConditionTypeGatesSatisfied, metav1.ConditionTrue, konfidence.ReasonAllGatesSatisfied)
		})
	})
})
