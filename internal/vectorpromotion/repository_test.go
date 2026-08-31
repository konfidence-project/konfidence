package vectorpromotion

import (
	"context"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
				RequireApproval:           true,
			},
			Status: konfidence.VectorPromotionStatus{Conditions: conditions},
		}
	}

	newRepository := func(objects ...client.Object) Repository {
		testScheme := runtime.NewScheme()
		Expect(scheme.AddToScheme(testScheme)).To(Succeed())
		Expect(konfidence.AddToScheme(testScheme)).To(Succeed())
		c := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithStatusSubresource(&konfidence.VectorPromotion{}).
			WithIndex(&konfidence.VectorPromotion{}, PromotionConfigNameField, PromotionConfigNameIndexFunc).
			WithObjects(objects...).
			Build()
		return NewRepository(c)
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
		It("approves a promotion waiting for approval", func() {
			repo := newRepository(newPromotion(metav1.Condition{
				Type:   konfidence.ConditionTypeApproved,
				Status: metav1.ConditionFalse,
				Reason: konfidence.ReasonPromotionWaitingForApproval,
			}))

			Expect(repo.Approve(ctx, key.Namespace, key.Name, "alice@example.com")).To(Succeed())

			approved, err := repo.Get(ctx, key.Namespace, key.Name)
			Expect(err).NotTo(HaveOccurred())
			cond := meta.FindStatusCondition(approved.Status.Conditions, konfidence.ConditionTypeApproved)
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			Expect(cond.Reason).To(Equal(konfidence.ReasonPromotionManuallyApproved))
			Expect(cond.Message).To(Equal(
				`promotion of vector "registry.example//konfidence.io/promo/app:1.0.0" approved by alice@example.com`))
			Expect(approved.Status.State).To(Equal(konfidence.PromotionStateReady))
			Expect(approved.Status.Approval).NotTo(BeNil())
			Expect(approved.Status.Approval.ApprovedBy).To(Equal("alice@example.com"))
			Expect(approved.Status.Approval.ApprovedAt.IsZero()).To(BeFalse())
		})

		It("rejects an already approved promotion", func() {
			repo := newRepository(newPromotion(metav1.Condition{
				Type:   konfidence.ConditionTypeApproved,
				Status: metav1.ConditionTrue,
				Reason: konfidence.ReasonPromotionManuallyApproved,
			}))

			Expect(repo.Approve(ctx, key.Namespace, key.Name, "alice@example.com")).To(MatchError(ErrAlreadyApproved))
		})

		DescribeTable("rejects terminal promotions",
			func(reason string, status metav1.ConditionStatus) {
				repo := newRepository(newPromotion(metav1.Condition{
					Type:   konfidence.ConditionTypeSucceeded,
					Status: status,
					Reason: reason,
				}))

				err := repo.Approve(ctx, key.Namespace, key.Name, "alice@example.com")
				Expect(err).To(MatchError(ErrPromotionFinished))
			},
			Entry("succeeded", konfidence.ReasonPromotionSucceeded, metav1.ConditionTrue),
			Entry("failed", konfidence.ReasonPromotionFailed, metav1.ConditionFalse),
		)

		It("rejects an empty approver", func() {
			repo := newRepository(newPromotion())

			Expect(repo.Approve(ctx, key.Namespace, key.Name, "")).To(MatchError(ErrApproverMissing))
		})

		It("rejects a promotion that has no approval gate", func() {
			gateless := newPromotion()
			gateless.Spec.RequireApproval = false
			repo := newRepository(gateless)

			Expect(repo.Approve(ctx, key.Namespace, key.Name, "alice@example.com")).To(MatchError(ErrApprovalNotRequired))
		})

		It("rejects a superseded promotion as locked", func() {
			repo := newRepository(newPromotion(metav1.Condition{
				Type:   konfidence.ConditionTypeSucceeded,
				Status: metav1.ConditionFalse,
				Reason: konfidence.ReasonPromotionSuperseded,
			}))

			Expect(repo.Approve(ctx, key.Namespace, key.Name, "alice@example.com")).To(MatchError(ErrPromotionSuperseded))
		})

		It("reports a missing promotion as not found", func() {
			repo := newRepository()

			err := repo.Approve(ctx, key.Namespace, key.Name, "alice@example.com")
			Expect(err).To(MatchError(ErrVectorPromotionNotFound))
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

		It("returns promotions belonging to the given config, sorted by sequence", func() {
			p1 := newPromotionForConfig("promo-2", "default", "config", 2)
			p2 := newPromotionForConfig("promo-1", "default", "config", 1)
			repo := newRepository(p1, p2)

			got, err := repo.ListForConfig(ctx, "default", "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(HaveLen(2))
			Expect(got[0].Name).To(Equal("promo-1"))
			Expect(got[1].Name).To(Equal("promo-2"))
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
