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

	It("upserts the gate entry without a prior seed", func() {
		c := newClient(interceptor.Funcs{}, newPromotion())
		Expect(promotiongate.Set(ctx, c, key, update)).To(Succeed())

		entry := promotiongate.Find(get(c), approval)
		Expect(entry).NotTo(BeNil())
		Expect(entry.State).To(Equal(konfidence.PromotionGateStateSatisfied))
		Expect(entry.SetBy).To(Equal("alice"))
		Expect(entry.SetAt).NotTo(BeNil())
	})

	It("returns ErrNotFound for a missing promotion", func() {
		c := newClient(interceptor.Funcs{})
		Expect(promotiongate.Set(ctx, c, key, update)).To(MatchError(promotiongate.ErrNotFound))
	})

	It("returns ErrGateNotDeclared for an undeclared gate", func() {
		c := newClient(interceptor.Funcs{}, newPromotion())
		u := update
		u.Type = "other.example.com"
		Expect(promotiongate.Set(ctx, c, key, u)).To(MatchError(promotiongate.ErrGateNotDeclared))
	})

	It("retries on conflict and keeps a concurrent writer's entry", func() {
		concurrentWrites := 0
		c := newClient(interceptor.Funcs{
			SubResourcePatch: func(ctx context.Context, c client.Client, sub string, obj client.Object,
				patch client.Patch, opts ...client.SubResourcePatchOption) error {
				if concurrentWrites == 0 {
					concurrentWrites++
					// Another writer (e.g. the promotion controller seeding)
					// lands between our read and our patch.
					other := &konfidence.VectorPromotion{}
					Expect(c.Get(ctx, key, other)).To(Succeed())
					promotiongate.Seed(other)
					Expect(c.Status().Update(ctx, other)).To(Succeed())
				}
				return c.Status().Patch(ctx, obj, patch, opts...)
			},
		}, newPromotion())

		Expect(promotiongate.Set(ctx, c, key, update)).To(Succeed())
		Expect(concurrentWrites).To(Equal(1))

		p := get(c)
		Expect(p.Status.Gates).To(HaveLen(2))
		Expect(promotiongate.StateOf(p, approval)).To(Equal(konfidence.PromotionGateStateSatisfied))
		Expect(promotiongate.StateOf(p, e2e)).To(Equal(konfidence.PromotionGateStatePending))
	})

	It("reads through the provided reader", func() {
		reads := 0
		reader := newClient(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, k client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				reads++
				return c.Get(ctx, k, obj, opts...)
			},
		}, newPromotion())
		Expect(promotiongate.Set(ctx, reader, key, update, promotiongate.WithReader(reader))).To(Succeed())
		Expect(reads).To(BeNumerically(">=", 1))
	})
})
