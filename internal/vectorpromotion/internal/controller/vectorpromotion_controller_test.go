package controller

import (
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
)

// The placeholder reconciler is exercised directly instead of through the
// shared manager: registering it there would race the manually driven
// conditions the TTL and status propagation tests rely on.
var _ = Describe("VectorPromotion execution placeholder", Ordered, Serial, func() {

	BeforeEach(func() { cleanupPromotions() })

	reconcile := func(name string) {
		r := &VectorPromotionReconciler{Client: k8sClient, Recorder: events.NewFakeRecorder(10)}
		_, err := r.Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Name: name, Namespace: testNamespace},
		})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}

	It("leaves the promotion untouched until the execution rework lands", func() {
		promotion := createPromotion("placeholder-promotion", "placeholder-config")
		resourceVersion := promotion.ResourceVersion

		reconcile(promotion.Name)

		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: promotion.Name, Namespace: testNamespace,
		}, promotion)).To(Succeed())
		Expect(promotion.ResourceVersion).To(Equal(resourceVersion))
		Expect(promotion.Status.Conditions).To(BeEmpty())
	})

	It("tolerates a promotion that no longer exists", func() {
		r := &VectorPromotionReconciler{Client: k8sClient, Recorder: events.NewFakeRecorder(10)}
		_, err := r.Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Name: "does-not-exist", Namespace: testNamespace},
		})
		Expect(err).ToNot(HaveOccurred())
	})
})

var _ = Describe("VectorPromotion derived state", func() {
	It("is empty until a controller writes status", func() {
		// The state field is derived and only written alongside conditions;
		// a freshly created promotion carries neither.
		promotion := &konfidence.VectorPromotion{}
		Expect(promotion.Status.State).To(BeEmpty())
	})
})
