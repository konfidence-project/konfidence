package controller

import (
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/konfidence-project/konfidence/pkg/promotiongate"
)

const e2eGate = "e2e-tests.example.com"

var _ = Describe("VectorPromotion gates phase", Ordered, Serial, func() {

	BeforeEach(func() { cleanupPromotions() })

	conditionOf := func(promotion *konfidence.VectorPromotion, conditionType string) *metav1.Condition {
		return meta.FindStatusCondition(promotion.Status.Conditions, conditionType)
	}
	expectCondition := func(promotion *konfidence.VectorPromotion, conditionType string, status metav1.ConditionStatus, reason string) {
		GinkgoHelper()
		cond := conditionOf(promotion, conditionType)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(status))
		Expect(cond.Reason).To(Equal(reason))
	}

	It("waits for approval and promotes once approved", func() {
		createLandscapeWithNamespace("exec-manual-landscape", "kden-l-exec-manual")
		stage := createStage("kden-l-exec-manual", "manual-stage", "registry.example//konfidence.io/promo/app:0.9.0")
		config := createConfig("exec-manual-config",
			stageSource("other-stage"), stageTargetInLandscape("manual-stage", "exec-manual-landscape"))
		promotion := createPromotionTargeting("exec-manual-promotion", config.Name,
			stageTargetInLandscape("manual-stage", "exec-manual-landscape"), true)

		By("first reconcile seeds the gate, records the promotion active and its gates pending")
		reconcilePromotion(promotion.Name)
		refreshPromotion(promotion)
		Expect(observedState(promotion)).To(Equal("Waiting"))
		Expect(promotion.Status.Gates).To(ConsistOf(konfidence.PromotionGateStatus{
			Type:  konfidence.GateTypeExplicitApproval,
			State: konfidence.PromotionGateStatePending,
		}))
		expectCondition(promotion, konfidence.ConditionTypeGatesSatisfied, metav1.ConditionFalse, konfidence.ReasonGatesPending)
		expectCondition(promotion, konfidence.ConditionTypeSucceeded, metav1.ConditionUnknown, konfidence.ReasonPromotionPending)

		By("reconciling again without approval does not execute")
		reconcilePromotion(promotion.Name)
		refreshPromotion(promotion)
		expectCondition(promotion, konfidence.ConditionTypeGatesSatisfied, metav1.ConditionFalse, konfidence.ReasonGatesPending)
		Expect(promotion.Status.StartTime).To(BeNil())

		By("approving clears the gates and executes in the same reconcile")
		approvePromotion(promotion)
		Expect(observedState(promotion)).To(Equal("Waiting"))
		reconcilePromotion(promotion.Name)
		refreshPromotion(promotion)
		Expect(observedState(promotion)).To(Equal("Succeeded"))
		expectCondition(promotion, konfidence.ConditionTypeGatesSatisfied, metav1.ConditionTrue, konfidence.ReasonAllGatesSatisfied)
		expectCondition(promotion, konfidence.ConditionTypeTargetResolved, metav1.ConditionTrue, konfidence.ReasonTargetResolved)
		expectCondition(promotion, konfidence.ConditionTypeSucceeded, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded)
		Expect(promotion.Status.StartTime).NotTo(BeNil())
		Expect(promotion.Status.CompletionTime).NotTo(BeNil())

		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: stage.Name, Namespace: stage.Namespace,
		}, stage)).To(Succeed())
		Expect(stage.Spec.Vector).To(Equal(testVector))
	})

	It("freezes the gates once GatesSatisfied is True", func() {
		promotion := createPromotionRequiringApproval("gates-freeze", "some-config")
		approvePromotion(promotion)

		By("the gate phase sets GatesSatisfied")
		Expect(newExecutionReconciler().reconcileGates(ctx, promotion)).To(Succeed())
		refreshPromotion(promotion)
		expectCondition(promotion, konfidence.ConditionTypeGatesSatisfied, metav1.ConditionTrue, konfidence.ReasonAllGatesSatisfied)
		Expect(observedState(promotion)).To(Equal("Ready"))

		By("a gate writer can no longer reopen the gate")
		key := types.NamespacedName{Name: promotion.Name, Namespace: testNamespace}
		Expect(promotiongate.Set(ctx, k8sClient, key, promotiongate.Update{
			Type: konfidence.GateTypeExplicitApproval, State: konfidence.PromotionGateStatePending, SetBy: "test",
		})).To(MatchError(promotiongate.ErrPromotionNotMutable))
	})

	It("keeps a gate that was satisfied before the first reconcile", func() {
		promotion := createGatedPromotion("gates-early-writer", "some-config",
			konfidence.PromotionGate{Type: konfidence.GateTypeExplicitApproval},
			konfidence.PromotionGate{Type: e2eGate})

		By("a gate writer acts before the promotion controller ever reconciled")
		setGate(promotion, e2eGate, konfidence.PromotionGateStateSatisfied, "TestsPassed")
		early := promotiongate.Find(promotion, e2eGate).DeepCopy()

		By("seeding adds the missing entry and leaves the written one untouched")
		reconcilePromotion(promotion.Name)
		refreshPromotion(promotion)
		Expect(observedState(promotion)).To(Equal("Waiting"))
		Expect(promotion.Status.Gates).To(HaveLen(2))
		Expect(promotiongate.Find(promotion, e2eGate)).To(Equal(early))
		Expect(promotiongate.StateOf(promotion, konfidence.GateTypeExplicitApproval)).
			To(Equal(konfidence.PromotionGateStatePending))
	})

	It("rejects a seed patch built from a stale read instead of overwriting a gate", func() {
		promotion := createGatedPromotion("gates-stale-seed", "some-config",
			konfidence.PromotionGate{Type: konfidence.GateTypeExplicitApproval},
			konfidence.PromotionGate{Type: e2eGate})
		stale := promotion.DeepCopy()

		By("a gate writer lands between the controller's read and its patch")
		setGate(promotion, e2eGate, konfidence.PromotionGateStateSatisfied, "TestsPassed")

		By("the controller's seed patch conflicts")
		err := newExecutionReconciler().reconcileGates(ctx, stale)
		Expect(apierrors.IsConflict(err)).To(BeTrue(), "expected a conflict, got %v", err)

		refreshPromotion(promotion)
		Expect(promotiongate.StateOf(promotion, e2eGate)).To(Equal(konfidence.PromotionGateStateSatisfied))

		By("the retried reconcile on fresh state seeds without losing the gate")
		reconcilePromotion(promotion.Name)
		refreshPromotion(promotion)
		Expect(promotiongate.StateOf(promotion, e2eGate)).To(Equal(konfidence.PromotionGateStateSatisfied))
		Expect(promotiongate.StateOf(promotion, konfidence.GateTypeExplicitApproval)).
			To(Equal(konfidence.PromotionGateStatePending))
	})

	It("surfaces a gate in Error without failing the promotion", func() {
		promotion := createGatedPromotion("gates-error", "some-config",
			konfidence.PromotionGate{Type: e2eGate})
		setGate(promotion, e2eGate, konfidence.PromotionGateStateError, "Misconfigured")

		reconciler := newExecutionReconciler()
		_, err := reconciler.Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Name: promotion.Name, Namespace: testNamespace},
		})
		Expect(err).NotTo(HaveOccurred())
		refreshPromotion(promotion)

		Expect(observedState(promotion)).To(Equal("Waiting"))
		expectCondition(promotion, konfidence.ConditionTypeGatesSatisfied, metav1.ConditionFalse, konfidence.ReasonGateError)
		expectCondition(promotion, konfidence.ConditionTypeSucceeded, metav1.ConditionUnknown, konfidence.ReasonPromotionPending)

		recorder, ok := reconciler.Recorder.(*events.FakeRecorder)
		Expect(ok).To(BeTrue())
		Expect(recorder.Events).To(Receive(ContainSubstring("Warning PromotionGateError")))

		By("the writer can still recover the gate")
		setGate(promotion, e2eGate, konfidence.PromotionGateStateSatisfied, "TestsPassed")
		Expect(newExecutionReconciler().reconcileGates(ctx, promotion)).To(Succeed())
		refreshPromotion(promotion)
		expectCondition(promotion, konfidence.ConditionTypeGatesSatisfied, metav1.ConditionTrue, konfidence.ReasonAllGatesSatisfied)
	})

	It("never rewrites a satisfied GatesSatisfied condition", func() {
		promotion := createPromotionRequiringApproval("gates-satisfied", "some-config")
		approvePromotion(promotion)
		Expect(newExecutionReconciler().reconcileGates(ctx, promotion)).To(Succeed())
		refreshPromotion(promotion)
		satisfied := meta.FindStatusCondition(promotion.Status.Conditions, konfidence.ConditionTypeGatesSatisfied).DeepCopy()

		By("a reconcile after it is set skips the gate phase and goes to execution")
		_, err := newExecutionReconciler().Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Name: promotion.Name, Namespace: testNamespace},
		})
		Expect(err).To(MatchError(ContainSubstring("does not exist")), "execution ran against the unresolved target")
		refreshPromotion(promotion)
		expectCondition(promotion, konfidence.ConditionTypeTargetResolved, metav1.ConditionFalse,
			konfidence.VectorPromotionConfigLandscapeNotFoundReason)
		Expect(*meta.FindStatusCondition(promotion.Status.Conditions, konfidence.ConditionTypeGatesSatisfied)).To(Equal(*satisfied))
	})

	It("does not touch a terminal promotion", func() {
		promotion := createPromotion("exec-terminal-promotion", "some-config")
		setSucceededCondition(promotion, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, metav1.Now().Time)
		refreshPromotion(promotion)
		resourceVersion := promotion.ResourceVersion

		reconcilePromotion(promotion.Name)
		refreshPromotion(promotion)
		Expect(promotion.ResourceVersion).To(Equal(resourceVersion))
	})
})
