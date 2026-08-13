package promotion

import (
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Lifecycle", func() {
	Describe("IsSuperseded", func() {
		DescribeTable("derives from the Succeeded condition",
			func(conditions []metav1.Condition, expected bool) {
				p := &konfidence.VectorPromotion{
					Status: konfidence.VectorPromotionStatus{Conditions: conditions},
				}

				Expect(IsSuperseded(p)).To(Equal(expected))
			},
			Entry("no conditions", nil, false),
			Entry("superseded", []metav1.Condition{{
				Type:   konfidence.ConditionTypeSucceeded,
				Status: metav1.ConditionFalse,
				Reason: konfidence.ReasonPromotionSuperseded,
			}}, true),
			Entry("failed", []metav1.Condition{{
				Type:   konfidence.ConditionTypeSucceeded,
				Status: metav1.ConditionFalse,
				Reason: konfidence.ReasonPromotionFailed,
			}}, false),
			Entry("succeeded", []metav1.Condition{{
				Type:   konfidence.ConditionTypeSucceeded,
				Status: metav1.ConditionTrue,
				Reason: konfidence.ReasonPromotionSucceeded,
			}}, false),
		)
	})

	Describe("IsTerminal", func() {
		Context("non-terminal states", func() {
			It("returns false when no conditions exist", func() {
				p := &konfidence.VectorPromotion{}

				Expect(IsTerminal(p)).To(BeFalse())
			})

			It("returns false when running", func() {
				p := &konfidence.VectorPromotion{
					Status: konfidence.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:   konfidence.ConditionTypeSucceeded,
							Status: metav1.ConditionFalse,
							Reason: konfidence.ReasonPromotionRunning,
						}},
					},
				}

				Expect(IsTerminal(p)).To(BeFalse())
			})
		})

		Context("terminal states", func() {
			It("returns true when succeeded", func() {
				p := &konfidence.VectorPromotion{
					Status: konfidence.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:   konfidence.ConditionTypeSucceeded,
							Status: metav1.ConditionTrue,
							Reason: konfidence.ReasonPromotionSucceeded,
						}},
					},
				}

				Expect(IsTerminal(p)).To(BeTrue())
			})

			It("returns true when failed", func() {
				p := &konfidence.VectorPromotion{
					Status: konfidence.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:   konfidence.ConditionTypeSucceeded,
							Status: metav1.ConditionFalse,
							Reason: konfidence.ReasonPromotionFailed,
						}},
					},
				}

				Expect(IsTerminal(p)).To(BeTrue())
			})

			It("returns false when the status is unknown", func() {
				p := &konfidence.VectorPromotion{
					Status: konfidence.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:   konfidence.ConditionTypeSucceeded,
							Status: metav1.ConditionUnknown,
							Reason: "SomeUnexpectedReason",
						}},
					},
				}

				Expect(IsTerminal(p)).To(BeFalse())
			})

			It("returns false when the target is unresolved", func() {
				p := &konfidence.VectorPromotion{
					Status: konfidence.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:   konfidence.ConditionTypeSucceeded,
							Status: metav1.ConditionFalse,
							Reason: konfidence.ReasonPromotionTargetUnresolved,
						}},
					},
				}

				Expect(IsTerminal(p)).To(BeFalse())
			})

			It("returns true when timed out", func() {
				p := &konfidence.VectorPromotion{
					Status: konfidence.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:   konfidence.ConditionTypeSucceeded,
							Status: metav1.ConditionFalse,
							Reason: konfidence.ReasonPromotionTimedOut,
						}},
					},
				}

				Expect(IsTerminal(p)).To(BeTrue())
			})
		})
	})

	Describe("DeriveState", func() {
		promotionWith := func(requireApproval bool, conditions ...metav1.Condition) *konfidence.VectorPromotion {
			return &konfidence.VectorPromotion{
				Spec:   konfidence.VectorPromotionSpec{RequireApproval: requireApproval},
				Status: konfidence.VectorPromotionStatus{Conditions: conditions},
			}
		}
		succeededCondition := func(status metav1.ConditionStatus, reason string) metav1.Condition {
			return metav1.Condition{Type: konfidence.ConditionTypeSucceeded, Status: status, Reason: reason}
		}

		DescribeTable("derives the state from conditions",
			func(p *konfidence.VectorPromotion, expected konfidence.VectorPromotionState) {
				Expect(DeriveState(p)).To(Equal(expected))
			},
			Entry("no conditions, no approval required",
				promotionWith(false), konfidence.PromotionStateReady),
			Entry("no conditions, approval required",
				promotionWith(true), konfidence.PromotionStateWaiting),
			Entry("approved, approval required, not started",
				promotionWith(true, metav1.Condition{
					Type:   konfidence.ConditionTypeApproved,
					Status: metav1.ConditionTrue,
					Reason: konfidence.ReasonPromotionManuallyApproved,
				}), konfidence.PromotionStateReady),
			Entry("approval condition false, approval required",
				promotionWith(true, metav1.Condition{
					Type:   konfidence.ConditionTypeApproved,
					Status: metav1.ConditionFalse,
					Reason: konfidence.ReasonPromotionWaitingForApproval,
				}), konfidence.PromotionStateWaiting),
			Entry("running",
				promotionWith(false, succeededCondition(metav1.ConditionFalse, konfidence.ReasonPromotionRunning)),
				konfidence.PromotionStateInProgress),
			Entry("succeeded",
				promotionWith(false, succeededCondition(metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded)),
				konfidence.PromotionStateSucceeded),
			Entry("blocked on unresolved target",
				promotionWith(false, succeededCondition(metav1.ConditionFalse, konfidence.ReasonPromotionTargetUnresolved)),
				konfidence.PromotionStateBlocked),
			Entry("superseded",
				promotionWith(false, succeededCondition(metav1.ConditionFalse, konfidence.ReasonPromotionSuperseded)),
				konfidence.PromotionStateSuperseded),
			Entry("failed",
				promotionWith(false, succeededCondition(metav1.ConditionFalse, konfidence.ReasonPromotionFailed)),
				konfidence.PromotionStateFailed),
			Entry("status unknown",
				promotionWith(false, succeededCondition(metav1.ConditionUnknown, "SomeUnexpectedReason")),
				konfidence.PromotionStateFailed),
		)
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

	Describe("InProgressLongerThan", func() {
		It("returns false for a promotion that is not in progress", func() {
			Expect(InProgressLongerThan(&konfidence.VectorPromotion{}, time.Minute)).To(BeFalse())
		})

		It("reports a promotion running longer than the given duration", func() {
			p := &konfidence.VectorPromotion{
				Status: konfidence.VectorPromotionStatus{
					Conditions: []metav1.Condition{{
						Type:               konfidence.ConditionTypeSucceeded,
						Status:             metav1.ConditionFalse,
						Reason:             konfidence.ReasonPromotionRunning,
						LastTransitionTime: metav1.NewTime(time.Now().Add(-10 * time.Minute)),
					}},
				},
			}
			Expect(InProgressLongerThan(p, 5*time.Minute)).To(BeTrue())
			Expect(InProgressLongerThan(p, 15*time.Minute)).To(BeFalse())
		})
	})

	Describe("Cleared", func() {
		It("treats a gateless promotion as cleared without any condition", func() {
			p := &konfidence.VectorPromotion{
				Spec: konfidence.VectorPromotionSpec{RequireApproval: false},
			}

			Expect(Cleared(p)).To(BeTrue())
		})

		It("treats an unapproved gated promotion as not cleared", func() {
			p := &konfidence.VectorPromotion{
				Spec: konfidence.VectorPromotionSpec{RequireApproval: true},
			}

			Expect(Cleared(p)).To(BeFalse())
		})
	})

	Describe("NewestCleared", func() {
		approvedAt := func(name string, sequence int64, conditions ...metav1.Condition) konfidence.VectorPromotion {
			all := append([]metav1.Condition{{
				Type:   konfidence.ConditionTypeApproved,
				Status: metav1.ConditionTrue,
				Reason: konfidence.ReasonPromotionManuallyApproved,
			}}, conditions...)
			return konfidence.VectorPromotion{
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec:       konfidence.VectorPromotionSpec{Sequence: sequence},
				Status:     konfidence.VectorPromotionStatus{Conditions: all},
			}
		}
		waiting := func(name string, sequence int64) konfidence.VectorPromotion {
			return konfidence.VectorPromotion{
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec:       konfidence.VectorPromotionSpec{Sequence: sequence, RequireApproval: true},
			}
		}
		gateless := func(name string, sequence int64) konfidence.VectorPromotion {
			return konfidence.VectorPromotion{
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec:       konfidence.VectorPromotionSpec{Sequence: sequence},
			}
		}

		It("returns nil when nothing is cleared", func() {
			Expect(NewestCleared([]konfidence.VectorPromotion{waiting("a", 1)})).To(BeNil())
		})

		It("returns the cleared promotion with the highest sequence", func() {
			result := NewestCleared([]konfidence.VectorPromotion{
				approvedAt("old", 1),
				approvedAt("new", 2),
				waiting("newest-but-waiting", 3),
			})
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("new"))
		})

		It("counts a gateless promotion as cleared without any condition", func() {
			result := NewestCleared([]konfidence.VectorPromotion{
				approvedAt("approved", 1),
				gateless("gateless", 2),
			})
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("gateless"))
		})

		It("skips terminal promotions", func() {
			result := NewestCleared([]konfidence.VectorPromotion{
				approvedAt("terminal", 2, metav1.Condition{
					Type:   konfidence.ConditionTypeSucceeded,
					Status: metav1.ConditionTrue,
					Reason: konfidence.ReasonPromotionSucceeded,
				}),
				approvedAt("live", 1),
			})
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("live"))
		})

		It("keeps the first listed promotion on equal sequences", func() {
			result := NewestCleared([]konfidence.VectorPromotion{
				approvedAt("promo-a", 1),
				approvedAt("promo-b", 1),
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
							Status:             metav1.ConditionFalse,
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
