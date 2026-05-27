package promotion

import (
	"time"

	global "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Lifecycle", func() {
	Describe("IsPending", func() {
		It("returns true when no conditions exist", func() {
			p := &global.VectorPromotion{}

			Expect(IsPending(p)).To(BeTrue())
		})

		It("returns false when Succeeded condition exists", func() {
			p := &global.VectorPromotion{
				Status: global.VectorPromotionStatus{
					Conditions: []metav1.Condition{{
						Type:   global.ConditionTypeSucceeded,
						Status: metav1.ConditionTrue,
					}},
				},
			}

			Expect(IsPending(p)).To(BeFalse())
		})
	})

	Describe("IsRunning", func() {
		It("returns false when no conditions exist", func() {
			p := &global.VectorPromotion{}

			Expect(IsRunning(p)).To(BeFalse())
		})

		It("returns true when reason is PromotionRunning", func() {
			p := &global.VectorPromotion{
				Status: global.VectorPromotionStatus{
					Conditions: []metav1.Condition{{
						Type:   global.ConditionTypeSucceeded,
						Status: metav1.ConditionFalse,
						Reason: global.ReasonPromotionRunning,
					}},
				},
			}

			Expect(IsRunning(p)).To(BeTrue())
		})

		It("returns false when succeeded", func() {
			p := &global.VectorPromotion{
				Status: global.VectorPromotionStatus{
					Conditions: []metav1.Condition{{
						Type:   global.ConditionTypeSucceeded,
						Status: metav1.ConditionTrue,
						Reason: global.ReasonPromotionSucceeded,
					}},
				},
			}

			Expect(IsRunning(p)).To(BeFalse())
		})

		It("returns false when failed", func() {
			p := &global.VectorPromotion{
				Status: global.VectorPromotionStatus{
					Conditions: []metav1.Condition{{
						Type:   global.ConditionTypeSucceeded,
						Status: metav1.ConditionFalse,
						Reason: global.ReasonPromotionFailed,
					}},
				},
			}

			Expect(IsRunning(p)).To(BeFalse())
		})
	})

	Describe("IsTerminal", func() {
		Context("non-terminal states", func() {
			It("returns false when no conditions exist", func() {
				p := &global.VectorPromotion{}

				Expect(IsTerminal(p)).To(BeFalse())
			})

			It("returns false when running", func() {
				p := &global.VectorPromotion{
					Status: global.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:   global.ConditionTypeSucceeded,
							Status: metav1.ConditionFalse,
							Reason: global.ReasonPromotionRunning,
						}},
					},
				}

				Expect(IsTerminal(p)).To(BeFalse())
			})
		})

		Context("terminal states", func() {
			It("returns true when succeeded", func() {
				p := &global.VectorPromotion{
					Status: global.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:   global.ConditionTypeSucceeded,
							Status: metav1.ConditionTrue,
							Reason: global.ReasonPromotionSucceeded,
						}},
					},
				}

				Expect(IsTerminal(p)).To(BeTrue())
			})

			It("returns true when failed", func() {
				p := &global.VectorPromotion{
					Status: global.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:   global.ConditionTypeSucceeded,
							Status: metav1.ConditionFalse,
							Reason: global.ReasonPromotionFailed,
						}},
					},
				}

				Expect(IsTerminal(p)).To(BeTrue())
			})

			It("returns true when unknown", func() {
				p := &global.VectorPromotion{
					Status: global.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:   global.ConditionTypeSucceeded,
							Status: metav1.ConditionUnknown,
							Reason: global.ReasonPromotionStatusUnknown,
						}},
					},
				}

				Expect(IsTerminal(p)).To(BeTrue())
			})

			It("returns true when source not found", func() {
				p := &global.VectorPromotion{
					Status: global.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:   global.ConditionTypeSucceeded,
							Status: metav1.ConditionFalse,
							Reason: global.ReasonPromotionSourceNotFound,
						}},
					},
				}

				Expect(IsTerminal(p)).To(BeTrue())
			})

			It("returns true when config not found", func() {
				p := &global.VectorPromotion{
					Status: global.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:   global.ConditionTypeSucceeded,
							Status: metav1.ConditionFalse,
							Reason: global.ReasonPromotionConfigurationNotFound,
						}},
					},
				}

				Expect(IsTerminal(p)).To(BeTrue())
			})
		})
	})

	Describe("TTLStatus", func() {
		Context("when TTL is not configured", func() {
			It("returns shouldDelete=false", func() {
				p := &global.VectorPromotion{}

				shouldDelete, remaining := TTLStatus(p)

				Expect(shouldDelete).To(BeFalse())
				Expect(remaining).To(Equal(time.Duration(0)))
			})
		})

		Context("when promotion is not terminal", func() {
			It("returns shouldDelete=false even with TTL configured", func() {
				p := &global.VectorPromotion{
					Spec: global.VectorPromotionSpec{
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
				p := &global.VectorPromotion{
					Spec: global.VectorPromotionSpec{
						TTLAfterFinished: &metav1.Duration{Duration: time.Hour},
					},
					Status: global.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:               global.ConditionTypeSucceeded,
							Status:             metav1.ConditionTrue,
							Reason:             global.ReasonPromotionSucceeded,
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
				p := &global.VectorPromotion{
					Spec: global.VectorPromotionSpec{
						TTLAfterFinished: &metav1.Duration{Duration: time.Hour},
					},
					Status: global.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:               global.ConditionTypeSucceeded,
							Status:             metav1.ConditionTrue,
							Reason:             global.ReasonPromotionSucceeded,
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
				p := &global.VectorPromotion{
					Spec: global.VectorPromotionSpec{
						TTLAfterFinished: &metav1.Duration{Duration: time.Hour},
					},
					Status: global.VectorPromotionStatus{
						Conditions: []metav1.Condition{{
							Type:               global.ConditionTypeSucceeded,
							Status:             metav1.ConditionFalse,
							Reason:             global.ReasonPromotionFailed,
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
