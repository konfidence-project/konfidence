package promotiongate_test

import (
	"context"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/konfidence-project/konfidence/pkg/promotiongate"
)

var _ = Describe("Set", func() {
	var (
		ctx    context.Context
		scheme *runtime.Scheme
		key    = types.NamespacedName{Namespace: "ns", Name: "promo-1"}
		update = promotiongate.Update{
			Type: approval, State: konfidence.PromotionGateStateSatisfied, Reason: "ApprovedByUser", SetBy: "alice",
		}
	)

	newPromotion := func() *konfidence.VectorPromotion {
		p := promotionWithGates(approval, e2e)
		p.ObjectMeta = metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name}
		return p
	}

	newClient := func(funcs interceptor.Funcs, objs ...client.Object) client.Client {
		return fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(objs...).
			WithStatusSubresource(&konfidence.VectorPromotion{}).
			WithInterceptorFuncs(funcs).
			Build()
	}

	get := func(c client.Client) *konfidence.VectorPromotion {
		p := &konfidence.VectorPromotion{}
		Expect(c.Get(ctx, key, p)).To(Succeed())
		return p
	}

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
		Expect(konfidence.AddToScheme(scheme)).To(Succeed())
	})

	It("upserts the gate with its state and provenance, stamping setAt", func() {
		c := newClient(interceptor.Funcs{}, newPromotion())
		Expect(promotiongate.Set(ctx, c, key, update)).To(Succeed())

		entry := promotiongate.Find(get(c), approval)
		Expect(entry).NotTo(BeNil())
		Expect(entry.State).To(Equal(konfidence.PromotionGateStateSatisfied))
		Expect(entry.Reason).To(Equal("ApprovedByUser"))
		Expect(entry.SetBy).To(Equal("alice"))
		Expect(entry.SetAt).NotTo(BeNil())
	})

	It("keeps the original provenance when the state does not change", func() {
		c := newClient(interceptor.Funcs{}, newPromotion())
		Expect(promotiongate.Set(ctx, c, key, update)).To(Succeed())
		version := get(c).ResourceVersion

		again := update
		again.SetBy = "bob"
		Expect(promotiongate.Set(ctx, c, key, again)).To(Succeed())

		Expect(promotiongate.Find(get(c), approval).SetBy).To(Equal("alice"))
		Expect(get(c).ResourceVersion).To(Equal(version), "a no-op must not patch")
	})

	It("overwrites the provenance on a state change", func() {
		c := newClient(interceptor.Funcs{}, newPromotion())
		Expect(promotiongate.Set(ctx, c, key, update)).To(Succeed())

		Expect(promotiongate.Set(ctx, c, key, promotiongate.Update{
			Type: approval, State: konfidence.PromotionGateStateError, Reason: "RevokedByUser", SetBy: "bob",
		})).To(Succeed())

		entry := promotiongate.Find(get(c), approval)
		Expect(entry.State).To(Equal(konfidence.PromotionGateStateError))
		Expect(entry.Reason).To(Equal("RevokedByUser"))
		Expect(entry.SetBy).To(Equal("bob"))
	})

	It("returns ErrNotFound for a missing promotion", func() {
		c := newClient(interceptor.Funcs{})
		Expect(promotiongate.Set(ctx, c, key, update)).To(MatchError(promotiongate.ErrNotFound))
	})

	DescribeTable("rejects invalid updates",
		func(u promotiongate.Update, expected error) {
			c := newClient(interceptor.Funcs{}, newPromotion())
			Expect(promotiongate.Set(ctx, c, key, u)).To(MatchError(expected))
		},
		Entry("undeclared gate",
			promotiongate.Update{Type: "other.example.com", State: konfidence.PromotionGateStateSatisfied, SetBy: "x"},
			promotiongate.ErrGateNotDeclared),
		Entry("missing setBy",
			promotiongate.Update{Type: approval, State: konfidence.PromotionGateStateSatisfied},
			promotiongate.ErrSetByMissing),
		Entry("missing type",
			promotiongate.Update{State: konfidence.PromotionGateStateSatisfied, SetBy: "x"},
			promotiongate.ErrTypeMissing),
		Entry("invalid state",
			promotiongate.Update{Type: approval, State: "Open", SetBy: "x"},
			promotiongate.ErrInvalidState),
	)

	DescribeTable("writes a gate only while the promotion is mutable",
		func(conds []metav1.Condition, mutable bool) {
			p := newPromotion()
			p.Status.Conditions = conds
			c := newClient(interceptor.Funcs{}, p)

			err := promotiongate.Set(ctx, c, key, update)
			if mutable {
				Expect(err).To(Succeed())
			} else {
				Expect(err).To(MatchError(promotiongate.ErrPromotionNotMutable))
			}
		},
		Entry("never reconciled", nil, true),
		Entry("a gate not satisfied yet", setConditions(konfidence.ConditionTypeGatesSatisfied, metav1.ConditionFalse), true),
		Entry("gates satisfied", setConditions(konfidence.ConditionTypeGatesSatisfied, metav1.ConditionTrue), false),
		Entry("active", setConditions(konfidence.ConditionTypeSucceeded, metav1.ConditionUnknown), true),
		Entry("superseded while waiting", setConditions(konfidence.ConditionTypeSucceeded, metav1.ConditionFalse), false),
		Entry("succeeded", setConditions(konfidence.ConditionTypeSucceeded, metav1.ConditionTrue), false),
	)

	It("returns ErrConflict when a concurrent write lands first", func() {
		c := newClient(interceptor.Funcs{
			SubResourcePatch: func(ctx context.Context, c client.Client, sub string, obj client.Object,
				patch client.Patch, opts ...client.SubResourcePatchOption) error {
				// Another writer (e.g. the promotion controller seeding) lands
				// between our read and our patch, bumping the resourceVersion so
				// our optimistic-locked patch conflicts.
				other := &konfidence.VectorPromotion{}
				Expect(c.Get(ctx, key, other)).To(Succeed())
				other.Status.Gates = append(other.Status.Gates, konfidence.PromotionGateStatus{
					Type: e2e, State: konfidence.PromotionGateStatePending,
				})
				Expect(c.Status().Update(ctx, other)).To(Succeed())
				return c.Status().Patch(ctx, obj, patch, opts...)
			},
		}, newPromotion())

		Expect(promotiongate.Set(ctx, c, key, update)).To(MatchError(promotiongate.ErrConflict))
	})
})

func setConditions(conditionType string, status metav1.ConditionStatus) []metav1.Condition {
	return []metav1.Condition{{Type: conditionType, Status: status, Reason: "SetByTest"}}
}
