package vectorpromotion

import (
	"context"
	"strings"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/konfidence-project/konfidence/pkg/promotiongate"
)

var _ = Describe("Repository", func() {
	var (
		ctx context.Context
		key types.NamespacedName
	)

	newPromotion := func(conditions ...metav1.Condition) *konfidence.VectorPromotion {
		return &konfidence.VectorPromotion{
			ObjectMeta: metav1.ObjectMeta{Name: "promotion", Namespace: "default"},
			Spec: konfidence.VectorPromotionSpec{
				VectorPromotionConfigName: "config",
				Vector:                    "registry.example//konfidence.io/promo/app:1.0.0",
				Gates:                     []konfidence.PromotionGate{{Type: konfidence.GateTypeExplicitApproval}},
			},
			Status: konfidence.VectorPromotionStatus{Conditions: conditions},
		}
	}

	newRepositoryWithInterceptor := func(funcs interceptor.Funcs, objects ...client.Object) Repository {
		testScheme := runtime.NewScheme()
		Expect(scheme.AddToScheme(testScheme)).To(Succeed())
		Expect(konfidence.AddToScheme(testScheme)).To(Succeed())
		c := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithStatusSubresource(&konfidence.VectorPromotion{}).
			WithIndex(&konfidence.VectorPromotion{}, PromotionConfigNameField, PromotionConfigNameIndexFunc).
			WithObjects(objects...).
			WithInterceptorFuncs(funcs).
			Build()
		return NewRepository(c)
	}
	newRepository := func(objects ...client.Object) Repository {
		return newRepositoryWithInterceptor(interceptor.Funcs{}, objects...)
	}

	BeforeEach(func() {
		ctx = context.Background()
		key = types.NamespacedName{Name: "promotion", Namespace: "default"}
	})

	Describe("Get", func() {
		It("returns the promotion for a namespace and name", func() {
			repo := newRepository(newPromotion())

			got, err := repo.Get(ctx, key.Namespace, key.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal(key.Name))
			Expect(got.Namespace).To(Equal(key.Namespace))
		})

		It("reports a missing promotion as not found", func() {
			repo := newRepository()

			_, err := repo.Get(ctx, key.Namespace, "missing")
			Expect(err).To(MatchError(ErrVectorPromotionNotFound))
		})

		It("does not return a promotion that lives in another namespace", func() {
			repo := newRepository(newPromotion())

			_, err := repo.Get(ctx, "other-namespace", key.Name)
			Expect(err).To(MatchError(ErrVectorPromotionNotFound))
		})

		It("scopes the lookup to the requested namespace", func() {
			inA := newPromotion()
			inA.Namespace = "project-a"
			inB := newPromotion()
			inB.Namespace = "project-b"
			repo := newRepository(inA, inB)

			got, err := repo.Get(ctx, "project-a", key.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Namespace).To(Equal("project-a"))
		})
	})

	Describe("Approve", func() {
		It("satisfies the explicit-approval gate with provenance", func() {
			waiting := newPromotion()
			waiting.Spec.Gates = append(waiting.Spec.Gates, konfidence.PromotionGate{Type: "e2e-tests.example.com"})
			waiting.Status.Gates = []konfidence.PromotionGateStatus{
				{Type: konfidence.GateTypeExplicitApproval, State: konfidence.PromotionGateStatePending},
				{Type: "e2e-tests.example.com", State: konfidence.PromotionGateStatePending},
			}
			repo := newRepository(waiting)

			Expect(repo.Approve(ctx, key.Namespace, key.Name, "alice@example.com")).To(Succeed())

			approved, err := repo.Get(ctx, key.Namespace, key.Name)
			Expect(err).NotTo(HaveOccurred())
			gate := promotiongate.Find(approved, konfidence.GateTypeExplicitApproval)
			Expect(gate).NotTo(BeNil())
			Expect(gate.State).To(Equal(konfidence.PromotionGateStateSatisfied))
			Expect(gate.Reason).To(Equal(konfidence.ReasonGateApprovedByUser))
			Expect(gate.Message).To(Equal("approved by alice@example.com"))
			Expect(gate.SetBy).To(Equal("alice@example.com"))
			Expect(gate.SetAt).NotTo(BeNil())
			Expect(promotiongate.StateOf(approved, "e2e-tests.example.com")).To(Equal(konfidence.PromotionGateStatePending))
		})

		It("approves a promotion whose gates were not seeded yet", func() {
			repo := newRepository(newPromotion())

			Expect(repo.Approve(ctx, key.Namespace, key.Name, "alice@example.com")).To(Succeed())

			approved, err := repo.Get(ctx, key.Namespace, key.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(promotiongate.StateOf(approved, konfidence.GateTypeExplicitApproval)).
				To(Equal(konfidence.PromotionGateStateSatisfied))
		})

		It("treats approving an approved promotion as a no-op", func() {
			approved := newPromotion()
			approved.Status.Gates = []konfidence.PromotionGateStatus{
				{Type: konfidence.GateTypeExplicitApproval, State: konfidence.PromotionGateStateSatisfied, SetBy: "bob"},
			}
			repo := newRepository(approved)

			Expect(repo.Approve(ctx, key.Namespace, key.Name, "alice@example.com")).To(Succeed())

			got, err := repo.Get(ctx, key.Namespace, key.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(promotiongate.Find(got, konfidence.GateTypeExplicitApproval).SetBy).To(Equal("bob"))
		})

		It("treats approving an approved promotion as a no-op even after it finished", func() {
			finished := newPromotion(metav1.Condition{
				Type: konfidence.ConditionTypeSucceeded, Status: metav1.ConditionTrue, Reason: konfidence.ReasonPromotionSucceeded,
			})
			finished.Status.Gates = []konfidence.PromotionGateStatus{
				{Type: konfidence.GateTypeExplicitApproval, State: konfidence.PromotionGateStateSatisfied, SetBy: "bob"},
			}
			repo := newRepository(finished)

			Expect(repo.Approve(ctx, key.Namespace, key.Name, "alice@example.com")).To(Succeed())
		})

		It("rejects a promotion whose gates are already satisfied", func() {
			repo := newRepository(newPromotion(metav1.Condition{
				Type:   konfidence.ConditionTypeGatesSatisfied,
				Status: metav1.ConditionTrue,
				Reason: konfidence.ReasonAllGatesSatisfied,
			}))

			Expect(repo.Approve(ctx, key.Namespace, key.Name, "alice@example.com")).To(MatchError(ErrPromotionExecuting))
		})

		DescribeTable("rejects finished promotions and forwards the outcome",
			func(reason string, status metav1.ConditionStatus) {
				repo := newRepository(newPromotion(metav1.Condition{
					Type:    konfidence.ConditionTypeSucceeded,
					Status:  status,
					Reason:  reason,
					Message: "set by test",
				}))

				err := repo.Approve(ctx, key.Namespace, key.Name, "alice@example.com")
				Expect(err).To(MatchError(ErrPromotionFinished))
				Expect(err).To(MatchError(ContainSubstring(reason + ": set by test")))
			},
			Entry("succeeded", konfidence.ReasonPromotionSucceeded, metav1.ConditionTrue),
			Entry("superseded by a newer promotion", konfidence.ReasonPromotionSuperseded, metav1.ConditionFalse),
			Entry("outstanding limit reached", konfidence.ReasonPromotionOutstandingLimitReached, metav1.ConditionFalse),
		)

		It("rejects an empty approver", func() {
			repo := newRepository(newPromotion())

			Expect(repo.Approve(ctx, key.Namespace, key.Name, "")).To(MatchError(ErrActorMissing))
		})

		It("rejects a promotion that has no approval gate", func() {
			gateless := newPromotion()
			gateless.Spec.Gates = []konfidence.PromotionGate{{Type: "e2e-tests.example.com"}}
			repo := newRepository(gateless)

			Expect(repo.Approve(ctx, key.Namespace, key.Name, "alice@example.com")).To(MatchError(ErrApprovalNotRequired))
		})

		It("reports a missing promotion as not found", func() {
			repo := newRepository()

			err := repo.Approve(ctx, key.Namespace, key.Name, "alice@example.com")
			Expect(err).To(MatchError(ErrVectorPromotionNotFound))
		})

		It("refuses a promotion an operator already overrode", func() {
			overridden := newPromotion()
			overridden.Status.Override = &konfidence.PromotionOverride{
				Type: konfidence.PromotionOverrideTypeReject, SetBy: "bob", Reason: "known bad",
			}
			repo := newRepository(overridden)

			err := repo.Approve(ctx, key.Namespace, key.Name, "alice@example.com")
			Expect(err).To(MatchError(ErrPromotionOverridden))
			Expect(err).To(MatchError(ContainSubstring("Reject by bob: known bad")))
		})
	})

	Describe("Reject and BypassGates", func() {
		type overrideFunc func(repo Repository, actor, reason string) error
		reject := func(repo Repository, actor, reason string) error {
			return repo.Reject(ctx, key.Namespace, key.Name, actor, reason)
		}
		bypassGates := func(repo Repository, actor, reason string) error {
			return repo.BypassGates(ctx, key.Namespace, key.Name, actor, reason)
		}

		DescribeTable("records the decision with provenance and leaves the gates untouched",
			func(apply overrideFunc, expectedType konfidence.PromotionOverrideType) {
				waiting := newPromotion()
				waiting.Status.Gates = []konfidence.PromotionGateStatus{
					{Type: konfidence.GateTypeExplicitApproval, State: konfidence.PromotionGateStateError, SetBy: "writer"},
				}
				repo := newRepository(waiting)

				Expect(apply(repo, "alice@example.com", "known bad vector")).To(Succeed())

				got, err := repo.Get(ctx, key.Namespace, key.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(got.Status.Override).NotTo(BeNil())
				Expect(got.Status.Override.Type).To(Equal(expectedType))
				Expect(got.Status.Override.SetBy).To(Equal("alice@example.com"))
				Expect(got.Status.Override.Reason).To(Equal("known bad vector"))
				Expect(got.Status.Override.SetAt.IsZero()).To(BeFalse())
				Expect(got.Status.Gates).To(Equal(waiting.Status.Gates))
			},
			Entry("reject", overrideFunc(reject), konfidence.PromotionOverrideTypeReject),
			Entry("bypass gates", overrideFunc(bypassGates), konfidence.PromotionOverrideTypeBypassGates),
		)

		DescribeTable("keeps the first decision",
			func(first konfidence.PromotionOverrideType, apply overrideFunc, expected error) {
				overridden := newPromotion()
				overridden.Status.Override = &konfidence.PromotionOverride{Type: first, SetBy: "bob", Reason: "first"}
				repo := newRepository(overridden)

				err := apply(repo, "alice@example.com", "second")
				if expected == nil {
					Expect(err).NotTo(HaveOccurred())
				} else {
					Expect(err).To(MatchError(expected))
				}

				got, err := repo.Get(ctx, key.Namespace, key.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(got.Status.Override.SetBy).To(Equal("bob"))
			},
			Entry("reject twice", konfidence.PromotionOverrideTypeReject, overrideFunc(reject), nil),
			Entry("bypass twice", konfidence.PromotionOverrideTypeBypassGates, overrideFunc(bypassGates), nil),
			Entry("reject after bypass", konfidence.PromotionOverrideTypeBypassGates, overrideFunc(reject), ErrPromotionOverridden),
			Entry("bypass after reject", konfidence.PromotionOverrideTypeReject, overrideFunc(bypassGates), ErrPromotionOverridden),
		)

		DescribeTable("refuses promotions that no longer wait on their gates",
			func(apply overrideFunc, condition metav1.Condition, expected error) {
				repo := newRepository(newPromotion(condition))
				Expect(apply(repo, "alice@example.com", "reason")).To(MatchError(expected))
			},
			Entry("reject a finished promotion", overrideFunc(reject),
				metav1.Condition{Type: konfidence.ConditionTypeSucceeded, Status: metav1.ConditionTrue, Reason: konfidence.ReasonPromotionSucceeded},
				ErrPromotionFinished),
			Entry("bypass a superseded promotion", overrideFunc(bypassGates),
				metav1.Condition{Type: konfidence.ConditionTypeSucceeded, Status: metav1.ConditionFalse, Reason: konfidence.ReasonPromotionSuperseded},
				ErrPromotionFinished),
			Entry("reject a promotion whose gates are satisfied", overrideFunc(reject),
				metav1.Condition{Type: konfidence.ConditionTypeGatesSatisfied, Status: metav1.ConditionTrue, Reason: konfidence.ReasonAllGatesSatisfied},
				ErrPromotionExecuting),
			Entry("bypass a promotion whose gates are satisfied", overrideFunc(bypassGates),
				metav1.Condition{Type: konfidence.ConditionTypeGatesSatisfied, Status: metav1.ConditionTrue, Reason: konfidence.ReasonAllGatesSatisfied},
				ErrPromotionExecuting),
		)

		DescribeTable("validates the provenance",
			func(actor, reason string, expected error) {
				repo := newRepository(newPromotion())
				Expect(repo.Reject(ctx, key.Namespace, key.Name, actor, reason)).To(MatchError(expected))
				Expect(repo.BypassGates(ctx, key.Namespace, key.Name, actor, reason)).To(MatchError(expected))
			},
			Entry("missing actor", "", "reason", ErrActorMissing),
			Entry("missing reason", "alice@example.com", "", ErrReasonInvalid),
			Entry("reason too long", "alice@example.com", strings.Repeat("x", MaxOverrideReasonLength+1), ErrReasonInvalid),
		)

		It("reports a concurrent write as a conflict instead of overwriting it", func() {
			repo := newRepositoryWithInterceptor(interceptor.Funcs{
				SubResourcePatch: func(context.Context, client.Client, string, client.Object, client.Patch, ...client.SubResourcePatchOption) error {
					return apierrors.NewConflict(schema.GroupResource{Resource: "vectorpromotions"}, key.Name, nil)
				},
			}, newPromotion())

			Expect(repo.Reject(ctx, key.Namespace, key.Name, "alice", "reason")).To(MatchError(ErrConflict))
		})

		It("reports a missing promotion as not found", func() {
			repo := newRepository()
			Expect(repo.Reject(ctx, key.Namespace, key.Name, "alice", "reason")).To(MatchError(ErrVectorPromotionNotFound))
			Expect(repo.BypassGates(ctx, key.Namespace, key.Name, "alice", "reason")).To(MatchError(ErrVectorPromotionNotFound))
		})
	})

	Describe("RevokeApproval", func() {
		approvedPromotion := func(conditions ...metav1.Condition) *konfidence.VectorPromotion {
			approved := newPromotion(conditions...)
			approved.Status.Gates = []konfidence.PromotionGateStatus{{
				Type:  konfidence.GateTypeExplicitApproval,
				State: konfidence.PromotionGateStateSatisfied,
				SetBy: "alice@example.com",
			}}
			return approved
		}

		It("sets the explicit-approval gate back to Pending with provenance", func() {
			approved := approvedPromotion()
			approved.Spec.Gates = append(approved.Spec.Gates, konfidence.PromotionGate{Type: "e2e-tests.example.com"})
			approved.Status.Gates = append(approved.Status.Gates, konfidence.PromotionGateStatus{
				Type: "e2e-tests.example.com", State: konfidence.PromotionGateStatePending,
			})
			repo := newRepository(approved)

			Expect(repo.RevokeApproval(ctx, key.Namespace, key.Name, "bob@example.com")).To(Succeed())

			revoked, err := repo.Get(ctx, key.Namespace, key.Name)
			Expect(err).NotTo(HaveOccurred())
			gate := promotiongate.Find(revoked, konfidence.GateTypeExplicitApproval)
			Expect(gate).NotTo(BeNil())
			Expect(gate.State).To(Equal(konfidence.PromotionGateStatePending))
			Expect(gate.Reason).To(Equal(konfidence.ReasonGateApprovalRevokedByUser))
			Expect(gate.Message).To(Equal("approval revoked by bob@example.com"))
			Expect(gate.SetBy).To(Equal("bob@example.com"))
			Expect(gate.SetAt).NotTo(BeNil())
			Expect(promotiongate.StateOf(revoked, "e2e-tests.example.com")).
				To(Equal(konfidence.PromotionGateStatePending))
		})

		It("is idempotent after revocation and preserves the original provenance", func() {
			repo := newRepository(approvedPromotion())
			Expect(repo.RevokeApproval(ctx, key.Namespace, key.Name, "bob@example.com")).To(Succeed())
			before, err := repo.Get(ctx, key.Namespace, key.Name)
			Expect(err).NotTo(HaveOccurred())

			Expect(repo.RevokeApproval(ctx, key.Namespace, key.Name, "carol@example.com")).To(Succeed())
			after, err := repo.Get(ctx, key.Namespace, key.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(after).To(Equal(before))

			By("allowing a fresh revocation after re-approval")
			Expect(repo.Approve(ctx, key.Namespace, key.Name, "alice@example.com")).To(Succeed())
			Expect(repo.RevokeApproval(ctx, key.Namespace, key.Name, "carol@example.com")).To(Succeed())
			after, err = repo.Get(ctx, key.Namespace, key.Name)
			Expect(err).NotTo(HaveOccurred())
			gate := promotiongate.Find(after, konfidence.GateTypeExplicitApproval)
			Expect(gate.State).To(Equal(konfidence.PromotionGateStatePending))
			Expect(gate.Reason).To(Equal(konfidence.ReasonGateApprovalRevokedByUser))
			Expect(gate.SetBy).To(Equal("carol@example.com"))
		})

		DescribeTable("rejects a gate that is not approved without changing the promotion",
			func(state konfidence.PromotionGateState) {
				pending := newPromotion()
				if state != "" {
					pending.Status.Gates = []konfidence.PromotionGateStatus{{
						Type: konfidence.GateTypeExplicitApproval, State: state,
					}}
				}
				repo := newRepository(pending)
				before, err := repo.Get(ctx, key.Namespace, key.Name)
				Expect(err).NotTo(HaveOccurred())

				Expect(repo.RevokeApproval(ctx, key.Namespace, key.Name, "bob@example.com")).
					To(MatchError(ErrNotApproved))
				after, err := repo.Get(ctx, key.Namespace, key.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(after).To(Equal(before))
			},
			Entry("missing status", konfidence.PromotionGateState("")),
			Entry("pending", konfidence.PromotionGateStatePending),
			Entry("error", konfidence.PromotionGateStateError),
		)

		It("rejects a promotion whose gates are already satisfied", func() {
			repo := newRepository(approvedPromotion(metav1.Condition{
				Type:   konfidence.ConditionTypeGatesSatisfied,
				Status: metav1.ConditionTrue,
				Reason: konfidence.ReasonAllGatesSatisfied,
			}))

			Expect(repo.RevokeApproval(ctx, key.Namespace, key.Name, "bob@example.com")).
				To(MatchError(ErrPromotionExecuting))
		})

		DescribeTable("rejects finished promotions and forwards the outcome",
			func(reason string, status metav1.ConditionStatus) {
				repo := newRepository(approvedPromotion(metav1.Condition{
					Type:    konfidence.ConditionTypeSucceeded,
					Status:  status,
					Reason:  reason,
					Message: "set by test",
				}))

				err := repo.RevokeApproval(ctx, key.Namespace, key.Name, "bob@example.com")
				Expect(err).To(MatchError(ErrPromotionFinished))
				Expect(err).To(MatchError(ContainSubstring(reason + ": set by test")))
			},
			Entry("succeeded", konfidence.ReasonPromotionSucceeded, metav1.ConditionTrue),
			Entry("superseded", konfidence.ReasonPromotionSuperseded, metav1.ConditionFalse),
		)

		It("rejects a promotion that has no approval gate", func() {
			gateless := approvedPromotion()
			gateless.Spec.Gates = []konfidence.PromotionGate{{Type: "e2e-tests.example.com"}}
			repo := newRepository(gateless)

			Expect(repo.RevokeApproval(ctx, key.Namespace, key.Name, "bob@example.com")).
				To(MatchError(ErrApprovalNotRequired))
		})

		It("rejects an empty revoker", func() {
			repo := newRepository(approvedPromotion())

			Expect(repo.RevokeApproval(ctx, key.Namespace, key.Name, "")).To(MatchError(ErrRevokerMissing))
		})

		It("reports a missing promotion as not found", func() {
			repo := newRepository()

			Expect(repo.RevokeApproval(ctx, key.Namespace, key.Name, "bob@example.com")).
				To(MatchError(ErrVectorPromotionNotFound))
		})
	})

	Describe("ListForConfig", func() {
		newPromotionForConfig := func(name, namespace, configName string, sequence int64) *konfidence.VectorPromotion {
			return &konfidence.VectorPromotion{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Spec: konfidence.VectorPromotionSpec{
					VectorPromotionConfigName: configName,
					Vector:                    "registry.example//konfidence.io/promo/app:1.0.0",
					Sequence:                  sequence,
				},
			}
		}

		It("returns promotions belonging to the given config, sorted by sequence descending (newest first)", func() {
			p1 := newPromotionForConfig("promo-2", "default", "config", 2)
			p2 := newPromotionForConfig("promo-1", "default", "config", 1)
			repo := newRepository(p1, p2)

			got, err := repo.ListForConfig(ctx, "default", "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(HaveLen(2))
			Expect(got[0].Name).To(Equal("promo-2"))
			Expect(got[1].Name).To(Equal("promo-1"))
		})

		It("excludes promotions from a different config in the same namespace", func() {
			target := newPromotionForConfig("target-promo", "default", "config", 1)
			other := newPromotionForConfig("other-promo", "default", "other-config", 1)
			repo := newRepository(target, other)

			got, err := repo.ListForConfig(ctx, "default", "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(HaveLen(1))
			Expect(got[0].Name).To(Equal("target-promo"))
		})

		It("is namespace-scoped: excludes promotions in another namespace", func() {
			inNamespace := newPromotionForConfig("promo-a", "project-a", "config", 1)
			otherNamespace := newPromotionForConfig("promo-b", "project-b", "config", 1)
			repo := newRepository(inNamespace, otherNamespace)

			got, err := repo.ListForConfig(ctx, "project-a", "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(HaveLen(1))
			Expect(got[0].Name).To(Equal("promo-a"))
		})

		It("returns empty when the config has no promotions", func() {
			repo := newRepository()

			got, err := repo.ListForConfig(ctx, "default", "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(BeEmpty())
		})
	})
})
