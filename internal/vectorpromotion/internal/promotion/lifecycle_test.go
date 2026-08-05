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

			It("returns true when unknown", func() {
				p := &konfidence.VectorPromotion{
					Status: konfidence.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:   konfidence.ConditionTypeSucceeded,
							Status: metav1.ConditionUnknown,
							Reason: "SomeUnexpectedReason",
						}},
					},
				}

				Expect(IsTerminal(p)).To(BeTrue())
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
			Entry("superseded",
				promotionWith(false, succeededCondition(metav1.ConditionFalse, konfidence.ReasonPromotionSuperseded)),
				konfidence.PromotionStateSuperseded),
			Entry("execution pending stub",
				promotionWith(false, succeededCondition(metav1.ConditionUnknown, konfidence.ReasonPromotionExecutionPending)),
				konfidence.PromotionStatePending),
			Entry("failed",
				promotionWith(false, succeededCondition(metav1.ConditionFalse, konfidence.ReasonPromotionFailed)),
				konfidence.PromotionStateFailed),
			Entry("status unknown",
				promotionWith(false, succeededCondition(metav1.ConditionUnknown, "SomeUnexpectedReason")),
				konfidence.PromotionStateFailed),
		)
	})

	Describe("TTLStatus", func() {
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
