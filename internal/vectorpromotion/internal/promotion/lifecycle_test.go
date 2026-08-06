package promotion

import (
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Lifecycle", func() {
	Describe("IsPending", func() {
		It("returns true when no conditions exist", func() {
			p := &konfidence.VectorPromotion{}

			Expect(IsPending(p)).To(BeTrue())
		})

		It("returns false when Succeeded condition exists", func() {
			p := &konfidence.VectorPromotion{
				Status: konfidence.VectorPromotionStatus{
					Conditions: []metav1.Condition{{
						Type:   konfidence.ConditionTypeSucceeded,
						Status: metav1.ConditionTrue,
					}},
				},
			}

			Expect(IsPending(p)).To(BeFalse())
		})
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

			It("returns true when source not found", func() {
				p := &konfidence.VectorPromotion{
					Status: konfidence.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:   konfidence.ConditionTypeSucceeded,
							Status: metav1.ConditionFalse,
							Reason: konfidence.ReasonPromotionSourceNotFound,
						}},
					},
				}

				Expect(IsTerminal(p)).To(BeTrue())
			})

			It("returns true when config not found", func() {
				p := &konfidence.VectorPromotion{
					Status: konfidence.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:   konfidence.ConditionTypeSucceeded,
							Status: metav1.ConditionFalse,
							Reason: konfidence.ReasonPromotionConfigurationNotFound,
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
				promotionWith(false), konfidence.PromotionStatePending),
			Entry("no conditions, approval required",
				promotionWith(true), konfidence.PromotionStateWaitingForApproval),
			Entry("approved, approval required, not started",
				promotionWith(true, metav1.Condition{
					Type:   konfidence.ConditionTypeApproved,
					Status: metav1.ConditionTrue,
					Reason: konfidence.ReasonPromotionManuallyApproved,
				}), konfidence.PromotionStateApproved),
			Entry("approval condition false, approval required",
				promotionWith(true, metav1.Condition{
					Type:   konfidence.ConditionTypeApproved,
					Status: metav1.ConditionFalse,
					Reason: konfidence.ReasonPromotionWaitingForApproval,
				}), konfidence.PromotionStateWaitingForApproval),
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
		now := metav1.NewTime(time.Now().Truncate(time.Second))
		promo := func(name string, created metav1.Time, sequence int64) *konfidence.VectorPromotion {
			return &konfidence.VectorPromotion{
				ObjectMeta: metav1.ObjectMeta{Name: name, CreationTimestamp: created},
				Spec:       konfidence.VectorPromotionSpec{Sequence: sequence},
			}
		}

		It("prefers the creator-assigned sequence over timestamps", func() {
			older := promo("older", metav1.NewTime(now.Add(time.Minute)), 1)
			newer := promo("newer", now, 2)
			Expect(Newer(newer, older)).To(BeTrue())
			Expect(Newer(older, newer)).To(BeFalse())
		})

		It("falls back to creation timestamps without sequences", func() {
			older := promo("older", now, 0)
			newer := promo("newer", metav1.NewTime(now.Add(time.Second)), 0)
			Expect(Newer(newer, older)).To(BeTrue())
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

	Describe("NewestApproved", func() {
		approvedAt := func(name string, created time.Time, conditions ...metav1.Condition) konfidence.VectorPromotion {
			all := append([]metav1.Condition{{
				Type:   konfidence.ConditionTypeApproved,
				Status: metav1.ConditionTrue,
				Reason: konfidence.ReasonPromotionManuallyApproved,
			}}, conditions...)
			return konfidence.VectorPromotion{
				ObjectMeta: metav1.ObjectMeta{Name: name, CreationTimestamp: metav1.NewTime(created)},
				Status:     konfidence.VectorPromotionStatus{Conditions: all},
			}
		}
		unapproved := func(name string, created time.Time) konfidence.VectorPromotion {
			return konfidence.VectorPromotion{
				ObjectMeta: metav1.ObjectMeta{Name: name, CreationTimestamp: metav1.NewTime(created)},
			}
		}
		now := time.Now().Truncate(time.Second)

		It("returns nil when nothing is approved", func() {
			Expect(NewestApproved([]konfidence.VectorPromotion{unapproved("a", now)})).To(BeNil())
		})

		It("returns the most recently created approved promotion", func() {
			result := NewestApproved([]konfidence.VectorPromotion{
				approvedAt("old", now.Add(-time.Minute)),
				approvedAt("new", now),
				unapproved("newest-but-unapproved", now.Add(time.Minute)),
			})
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("new"))
		})

		It("skips terminal promotions", func() {
			result := NewestApproved([]konfidence.VectorPromotion{
				approvedAt("terminal", now, metav1.Condition{
					Type:   konfidence.ConditionTypeSucceeded,
					Status: metav1.ConditionTrue,
					Reason: konfidence.ReasonPromotionSucceeded,
				}),
				approvedAt("live", now.Add(-time.Minute)),
			})
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("live"))
		})

		It("breaks creation timestamp ties by name", func() {
			result := NewestApproved([]konfidence.VectorPromotion{
				approvedAt("promo-a", now),
				approvedAt("promo-b", now),
			})
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("promo-b"))
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
