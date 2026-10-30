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

	cond := func(conditionType string, status metav1.ConditionStatus) metav1.Condition {
		return metav1.Condition{Type: conditionType, Status: status, Reason: "SetByTest"}
	}
	gates := func(status metav1.ConditionStatus) metav1.Condition {
		return cond(konfidence.ConditionTypeGatesSatisfied, status)
	}
	succeeded := func(status metav1.ConditionStatus) metav1.Condition {
		return cond(konfidence.ConditionTypeSucceeded, status)
	}

	type facts struct {
		conditions []metav1.Condition
		started    bool
	}
	type predicates struct {
		finished, succeeded, allGatesSatisfied, waiting, cleared, started bool
	}

	DescribeTable("derives every predicate from condition statuses and timestamps",
		func(f facts, expected predicates) {
			p := withConditions(f.conditions)
			if f.started {
				now := metav1.Now()
				p.Status.StartTime = &now
			}
			Expect(predicates{
				finished:          IsTerminal(p),
				succeeded:         IsSucceeded(p),
				allGatesSatisfied: AllGatesSatisfied(p),
				waiting:           IsWaiting(p),
				cleared:           IsCleared(p),
				started:           IsRunning(p),
			}).To(Equal(expected))
		},
		Entry("never reconciled", facts{},
			predicates{waiting: true}),
		Entry("a gate not satisfied yet", facts{conditions: []metav1.Condition{gates(metav1.ConditionFalse), succeeded(metav1.ConditionUnknown)}},
			predicates{waiting: true}),
		Entry("gates satisfied", facts{conditions: []metav1.Condition{gates(metav1.ConditionTrue), succeeded(metav1.ConditionUnknown)}},
			predicates{allGatesSatisfied: true, cleared: true}),
		Entry("started", facts{conditions: []metav1.Condition{gates(metav1.ConditionTrue), succeeded(metav1.ConditionUnknown)}, started: true},
			predicates{allGatesSatisfied: true, cleared: true, started: true}),
		Entry("succeeded", facts{conditions: []metav1.Condition{gates(metav1.ConditionTrue), succeeded(metav1.ConditionTrue)}, started: true},
			predicates{finished: true, succeeded: true, allGatesSatisfied: true}),
		Entry("superseded while waiting", facts{conditions: []metav1.Condition{gates(metav1.ConditionFalse), succeeded(metav1.ConditionFalse)}},
			predicates{finished: true}),
	)

	It("never reads a condition reason", func() {
		p := withConditions(phase(metav1.ConditionUnknown, "SomeUnexpectedReason"))
		Expect(IsTerminal(p)).To(BeFalse())
		p = withConditions(phase(metav1.ConditionFalse, "SomeUnexpectedReason"))
		Expect(IsTerminal(p)).To(BeTrue())
	})

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
		promo := func(name string, sequence int64, conditions ...metav1.Condition) konfidence.VectorPromotion {
			return konfidence.VectorPromotion{
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec:       konfidence.VectorPromotionSpec{Sequence: sequence},
				Status:     konfidence.VectorPromotionStatus{Conditions: conditions},
			}
		}
		ready := func(name string, sequence int64) konfidence.VectorPromotion {
			return promo(name, sequence, gates(metav1.ConditionTrue), succeeded(metav1.ConditionUnknown))
		}
		waiting := func(name string, sequence int64) konfidence.VectorPromotion {
			return promo(name, sequence, gates(metav1.ConditionFalse), succeeded(metav1.ConditionUnknown))
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

		It("skips finished promotions", func() {
			result := NewestCleared([]konfidence.VectorPromotion{
				promo("finished", 2, gates(metav1.ConditionTrue), succeeded(metav1.ConditionTrue)),
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

	Describe("TTLStatus completion time", func() {
		It("measures from completionTime when set", func() {
			completed := metav1.NewTime(time.Now().Add(-2 * time.Hour))
			p := withConditions([]metav1.Condition{{
				Type: konfidence.ConditionTypeSucceeded, Status: metav1.ConditionTrue, Reason: "SetByTest",
				LastTransitionTime: metav1.Now(),
			}})
			p.Spec.TTLAfterFinished = &metav1.Duration{Duration: time.Hour}
			p.Status.CompletionTime = &completed
			shouldDelete, _ := TTLStatus(p)
			Expect(shouldDelete).To(BeTrue())
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
							Reason:             konfidence.ReasonPromotionSuperseded,
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
