package vectorpromotion

import (
	"context"
	"fmt"

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

var _ = Describe("Approve", func() {
	var (
		ctx context.Context
		key types.NamespacedName
	)

	newPromotion := func(conditions ...metav1.Condition) *konfidence.VectorPromotion {
		return &konfidence.VectorPromotion{
			ObjectMeta: metav1.ObjectMeta{Name: "promotion", Namespace: "default"},
			Spec: konfidence.VectorPromotionSpec{
				VectorPromotionConfigRef: "config",
				Vector:                   "registry.example//konfidence.io/promo/app:1.0.0",
				RequireApproval:          true,
			},
			Status: konfidence.VectorPromotionStatus{Conditions: conditions},
		}
	}

	newClient := func(objects ...client.Object) client.Client {
		testScheme := runtime.NewScheme()
		Expect(scheme.AddToScheme(testScheme)).To(Succeed())
		Expect(konfidence.AddToScheme(testScheme)).To(Succeed())
		return fake.NewClientBuilder().
			WithScheme(testScheme).
			WithStatusSubresource(&konfidence.VectorPromotion{}).
			WithObjects(objects...).
			Build()
	}

	BeforeEach(func() {
		ctx = context.Background()
		key = types.NamespacedName{Name: "promotion", Namespace: "default"}
	})

	It("approves a promotion waiting for approval", func() {
		c := newClient(newPromotion(metav1.Condition{
			Type:   konfidence.ConditionTypeApproved,
			Status: metav1.ConditionFalse,
			Reason: konfidence.ReasonPromotionWaitingForApproval,
		}))

		Expect(Approve(ctx, c, key, "alice@example.com")).To(Succeed())

		approved := &konfidence.VectorPromotion{}
		Expect(c.Get(ctx, key, approved)).To(Succeed())
		cond := meta.FindStatusCondition(approved.Status.Conditions, konfidence.ConditionTypeApproved)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		Expect(cond.Reason).To(Equal(konfidence.ReasonPromotionManuallyApproved))
		Expect(cond.Message).To(Equal("approved by alice@example.com"))
		Expect(approved.Status.State).To(Equal(konfidence.PromotionStateApproved))
		Expect(approved.Status.Approvals).To(HaveLen(1))
		Expect(approved.Status.Approvals[0].ApprovedBy).To(Equal("alice@example.com"))
		Expect(approved.Status.Approvals[0].ApprovedAt.IsZero()).To(BeFalse())
	})

	It("rejects an already approved promotion", func() {
		c := newClient(newPromotion(metav1.Condition{
			Type:   konfidence.ConditionTypeApproved,
			Status: metav1.ConditionTrue,
			Reason: konfidence.ReasonPromotionManuallyApproved,
		}))

		Expect(Approve(ctx, c, key, "alice@example.com")).To(MatchError(ErrAlreadyApproved))
	})

	DescribeTable("rejects terminal promotions",
		func(reason string, status metav1.ConditionStatus) {
			c := newClient(newPromotion(metav1.Condition{
				Type:   konfidence.ConditionTypeSucceeded,
				Status: status,
				Reason: reason,
			}))

			err := Approve(ctx, c, key, "alice@example.com")
			Expect(err).To(MatchError(ErrPromotionFinished))
		},
		Entry("succeeded", konfidence.ReasonPromotionSucceeded, metav1.ConditionTrue),
		Entry("superseded", konfidence.ReasonPromotionSuperseded, metav1.ConditionFalse),
		Entry("failed", konfidence.ReasonPromotionFailed, metav1.ConditionFalse),
	)

	It("wraps a fetch failure", func() {
		c := newClient()

		err := Approve(ctx, c, key, "alice@example.com")
		Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("failed to fetch VectorPromotion %q", key.Name))))
	})
})
