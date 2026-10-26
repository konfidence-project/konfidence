package promotion

import (
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Lifecycle", func() {
	// phase builds the Succeeded condition that carries the promotion phase.
	phase := func(status metav1.ConditionStatus, reason string) []metav1.Condition {
		return []metav1.Condition{{Type: konfidence.ConditionTypeSucceeded, Status: status, Reason: reason}}
	}
	withConditions := func(conditions []metav1.Condition) *konfidence.VectorPromotion {
		return &konfidence.VectorPromotion{Status: konfidence.VectorPromotionStatus{Conditions: conditions}}
	}

	DescribeTable("pairs every reason with its condition status and state",
		func(reason string, status metav1.ConditionStatus, state konfidence.VectorPromotionState) {
			gotStatus, ok := ConditionStatusFor(reason)
			Expect(ok).To(BeTrue())
			Expect(gotStatus).To(Equal(status))
			Expect(DeriveState(withConditions(phase(status, reason)))).To(Equal(state))
		},
		Entry("gates pending", konfidence.ReasonGatesPending, metav1.ConditionUnknown, konfidence.PromotionStateWaiting),
		Entry("gate error", konfidence.ReasonGateError, metav1.ConditionUnknown, konfidence.PromotionStateWaiting),
		Entry("ready", konfidence.ReasonPromotionReady, metav1.ConditionUnknown, konfidence.PromotionStateReady),
		Entry("running", konfidence.ReasonPromotionRunning, metav1.ConditionUnknown, konfidence.PromotionStateInProgress),
		Entry("target unresolved", konfidence.ReasonPromotionTargetUnresolved, metav1.ConditionUnknown,
			konfidence.PromotionStateBlocked),
		Entry("succeeded", konfidence.ReasonPromotionSucceeded, metav1.ConditionTrue, konfidence.PromotionStateSucceeded),
		Entry("superseded", konfidence.ReasonPromotionSuperseded, metav1.ConditionFalse, konfidence.PromotionStateSuperseded),
		Entry("outstanding limit reached", konfidence.ReasonPromotionOutstandingLimitReached, metav1.ConditionFalse,
			konfidence.PromotionStateSuperseded),
		Entry("failed", konfidence.ReasonPromotionFailed, metav1.ConditionFalse, konfidence.PromotionStateFailed),
	)

	Describe("DeriveState", func() {
		It("is Waiting before the first reconcile", func() {
			Expect(DeriveState(withConditions(nil))).To(Equal(konfidence.PromotionStateWaiting))
		})

		It("has no state for an unrecognized reason", func() {
			p := withConditions(phase(metav1.ConditionUnknown, "SomeUnexpectedReason"))
			Expect(DeriveState(p)).To(BeEmpty())
			Expect(DeriveState(p).Terminal()).To(BeFalse())
			Expect(DeriveState(p).Cleared()).To(BeFalse())
			_, ok := ConditionStatusFor("SomeUnexpectedReason")
			Expect(ok).To(BeFalse())
		})

		It("ignores the gates: only the recorded phase counts", func() {
			p := withConditions(phase(metav1.ConditionUnknown, konfidence.ReasonPromotionReady))
			p.Spec.Gates = []konfidence.PromotionGate{{Type: konfidence.GateTypeExplicitApproval}}
			p.Status.Gates = []konfidence.PromotionGateStatus{{
				Type: konfidence.GateTypeExplicitApproval, State: konfidence.PromotionGateStatePending,
			}}
			Expect(DeriveState(p)).To(Equal(konfidence.PromotionStateReady))
		})
	})

	DescribeTable("classifies states as terminal or cleared",
		func(state konfidence.VectorPromotionState, terminal, cleared bool) {
			Expect(state.Terminal()).To(Equal(terminal))
			Expect(state.Cleared()).To(Equal(cleared))
		},
		Entry("waiting", konfidence.PromotionStateWaiting, false, false),
		Entry("ready", konfidence.PromotionStateReady, false, true),
		Entry("in progress", konfidence.PromotionStateInProgress, false, true),
		Entry("blocked", konfidence.PromotionStateBlocked, false, true),
		Entry("succeeded", konfidence.PromotionStateSucceeded, true, false),
		Entry("failed", konfidence.PromotionStateFailed, true, false),
		Entry("superseded", konfidence.PromotionStateSuperseded, true, false),
	)

	Describe("Newer", func() {
		promo := func(sequence int64) *konfidence.VectorPromotion {
			return &konfidence.VectorPromotion{
				Spec: konfidence.VectorPromotionSpec{Sequence: sequence},
			}
		}

		It("orders by the creator-assigned sequence", func() {
			Expect(Newer(promo(2), promo(1))).To(BeTrue())
			Expect(Newer(promo(1), promo(2))).To(BeFalse())
		})

		It("treats equal sequences as not newer", func() {
			Expect(Newer(promo(1), promo(1))).To(BeFalse())
		})
	})

	Describe("NewestCleared", func() {
		promo := func(name string, sequence int64, status metav1.ConditionStatus, reason string) konfidence.VectorPromotion {
			return konfidence.VectorPromotion{
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec:       konfidence.VectorPromotionSpec{Sequence: sequence},
				Status:     konfidence.VectorPromotionStatus{Conditions: phase(status, reason)},
			}
		}
		ready := func(name string, sequence int64) konfidence.VectorPromotion {
			return promo(name, sequence, metav1.ConditionUnknown, konfidence.ReasonPromotionReady)
		}
		waiting := func(name string, sequence int64) konfidence.VectorPromotion {
			return promo(name, sequence, metav1.ConditionUnknown, konfidence.ReasonGatesPending)
		}

		It("returns nil when nothing is cleared", func() {
			Expect(NewestCleared([]konfidence.VectorPromotion{waiting("a", 1)})).To(BeNil())
		})

		It("returns the cleared promotion with the highest sequence", func() {
			result := NewestCleared([]konfidence.VectorPromotion{
				ready("old", 1),
				ready("new", 2),
				waiting("newest-but-waiting", 3),
			})
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("new"))
		})

		It("counts in-progress and blocked promotions as cleared", func() {
			result := NewestCleared([]konfidence.VectorPromotion{
				ready("ready", 1),
				promo("blocked", 2, metav1.ConditionUnknown, konfidence.ReasonPromotionTargetUnresolved),
			})
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("blocked"))
		})

		It("skips terminal promotions", func() {
			result := NewestCleared([]konfidence.VectorPromotion{
				promo("terminal", 2, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded),
				ready("live", 1),
			})
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("live"))
		})

		It("keeps the first listed promotion on equal sequences", func() {
			result := NewestCleared([]konfidence.VectorPromotion{
				ready("promo-a", 1),
				ready("promo-b", 1),
			})
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("promo-a"))
		})
	})

	Describe("TTLStatus", func() {
		Context("when the promotion is running", func() {
			It("returns shouldDelete=false even with an expired TTL", func() {
				p := &konfidence.VectorPromotion{
					Spec: konfidence.VectorPromotionSpec{
						TTLAfterFinished: &metav1.Duration{Duration: time.Hour},
					},
					Status: konfidence.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:               konfidence.ConditionTypeSucceeded,
							Status:             metav1.ConditionUnknown,
							Reason:             konfidence.ReasonPromotionRunning,
							LastTransitionTime: metav1.NewTime(time.Now().Add(-2 * time.Hour)),
						}},
					},
				}

				shouldDelete, _ := TTLStatus(p)

				Expect(shouldDelete).To(BeFalse())
			})
		})

		Context("when TTL is not configured", func() {
			It("returns shouldDelete=false", func() {
				p := &konfidence.VectorPromotion{}

				shouldDelete, remaining := TTLStatus(p)

				Expect(shouldDelete).To(BeFalse())
				Expect(remaining).To(Equal(time.Duration(0)))
			})
		})

		Context("when promotion is not terminal", func() {
			It("returns shouldDelete=false even with TTL configured", func() {
				p := &konfidence.VectorPromotion{
					Spec: konfidence.VectorPromotionSpec{
						TTLAfterFinished: &metav1.Duration{Duration: time.Hour},
					},
				}

				shouldDelete, remaining := TTLStatus(p)

				Expect(shouldDelete).To(BeFalse())
				Expect(remaining).To(Equal(time.Duration(0)))
			})
		})

		Context("when TTL has expired", func() {
			It("returns shouldDelete=true", func() {
				pastTime := metav1.NewTime(time.Now().Add(-2 * time.Hour))
				p := &konfidence.VectorPromotion{
					Spec: konfidence.VectorPromotionSpec{
						TTLAfterFinished: &metav1.Duration{Duration: time.Hour},
					},
					Status: konfidence.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:               konfidence.ConditionTypeSucceeded,
							Status:             metav1.ConditionTrue,
							Reason:             konfidence.ReasonPromotionSucceeded,
							LastTransitionTime: pastTime,
						}},
					},
				}

				shouldDelete, remaining := TTLStatus(p)

				Expect(shouldDelete).To(BeTrue())
				Expect(remaining).To(BeNumerically("<=", 0))
			})
		})

		Context("when TTL has not expired", func() {
			It("returns shouldDelete=false with positive remaining time", func() {
				recentTime := metav1.NewTime(time.Now().Add(-30 * time.Minute))
				p := &konfidence.VectorPromotion{
					Spec: konfidence.VectorPromotionSpec{
						TTLAfterFinished: &metav1.Duration{Duration: time.Hour},
					},
					Status: konfidence.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:               konfidence.ConditionTypeSucceeded,
							Status:             metav1.ConditionTrue,
							Reason:             konfidence.ReasonPromotionSucceeded,
							LastTransitionTime: recentTime,
						}},
					},
				}

				shouldDelete, remaining := TTLStatus(p)

				Expect(shouldDelete).To(BeFalse())
				Expect(remaining).To(BeNumerically(">", 0))
				Expect(remaining).To(BeNumerically("<=", 30*time.Minute))
			})
		})

		Context("when promotion failed with TTL", func() {
			It("still respects TTL for failed promotions", func() {
				pastTime := metav1.NewTime(time.Now().Add(-2 * time.Hour))
				p := &konfidence.VectorPromotion{
					Spec: konfidence.VectorPromotionSpec{
						TTLAfterFinished: &metav1.Duration{Duration: time.Hour},
					},
					Status: konfidence.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:               konfidence.ConditionTypeSucceeded,
							Status:             metav1.ConditionFalse,
							Reason:             konfidence.ReasonPromotionFailed,
							LastTransitionTime: pastTime,
						}},
					},
				}

				shouldDelete, _ := TTLStatus(p)

				Expect(shouldDelete).To(BeTrue())
			})
		})
	})
})
