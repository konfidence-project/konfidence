//nolint:staticcheck // ST1001: allow dot-import for test specs using Ginkgo/Gomega
package controller

import (
	"context"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// noopRecorder is a no-op events.EventRecorder for the fake-client specs.
type noopRecorder struct{}

func (noopRecorder) Eventf(_ runtime.Object, _ runtime.Object, _, _, _, _ string, _ ...interface{}) {}

// newFakeReconciler builds a LandscapeReconciler backed by a fake client seeded
// with the given objects. Unlike envtest, the fake client has no
// namespace-termination semantics, so a deleted namespace disappears
// immediately — letting these specs observe the full delete cycle
// (namespace gone → finalizer released) that envtest cannot reproduce.
func newFakeReconciler(objects ...client.Object) (*LandscapeReconciler, client.Client) {
	scheme := runtime.NewScheme()
	Expect(konfidence.AddToScheme(scheme)).To(Succeed())
	Expect(corev1.AddToScheme(scheme)).To(Succeed())

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		WithStatusSubresource(&konfidence.Landscape{}).
		Build()
	return &LandscapeReconciler{Client: c, Scheme: scheme, Recorder: noopRecorder{}}, c
}

// withFinalizer marks a Landscape as owning the controller finalizer and a UID,
// mirroring a Landscape that has already been reconciled once.
func withFinalizer(l *konfidence.Landscape) {
	l.UID = types.UID("uid-" + l.Name)
	l.Finalizers = []string{landscapeFinalizer}
}

// newManagedNamespace builds a namespace managed by the given Landscape, as the
// controller would create it (managed labels, no owner reference because
// Landscape is namespace-scoped and cannot own a cluster-scoped resource).
func newManagedNamespace(name, projectName string, landscape *konfidence.Landscape) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				pkgctrl.ManagedByLabel:     landscapeControllerName,
				pkgctrl.ProjectTypeLabel:   landscapeNamespaceTypeValue,
				pkgctrl.ProjectNameLabel:   projectName,
				pkgctrl.LandscapeNameLabel: landscape.Name,
			},
		},
	}
}

func namespaceMissing(ctx context.Context, c client.Client, name string) bool {
	err := c.Get(ctx, types.NamespacedName{Name: name}, &corev1.Namespace{})
	return apierrors.IsNotFound(err)
}

func landscapeFinalizers(ctx context.Context, c client.Client, name, namespace string) []string {
	landscape := &konfidence.Landscape{}
	Expect(c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, landscape)).To(Succeed())
	return landscape.Finalizers
}

var _ = Describe("Landscape reconcileDelete", func() {
	ctx := context.Background()

	It("deletes the managed namespace, then releases the finalizer once it is gone", func() {
		const projectName = "proj1"
		landscape := NewLandscape("alpha", "kden-p-proj1", withFinalizer, func(l *konfidence.Landscape) {
			l.Status.ProjectName = projectName
			l.Status.Namespace = "kden-l-alpha-1ogisnw"
		})
		ns := newManagedNamespace("kden-l-alpha-1ogisnw", projectName, landscape)
		r, c := newFakeReconciler(landscape, ns)

		By("deleting the namespace and requeueing while the deletion is pending")
		result, err := r.reconcileDelete(ctx, landscape)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(Equal(deletionRequeueInterval))
		Expect(namespaceMissing(ctx, c, ns.Name)).To(BeTrue())
		Expect(landscapeFinalizers(ctx, c, "alpha", "kden-p-proj1")).To(ContainElement(landscapeFinalizer))

		By("releasing the finalizer on the next pass, once the namespace is gone")
		result, err = r.reconcileDelete(ctx, landscape)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsZero()).To(BeTrue())
		Expect(landscapeFinalizers(ctx, c, "alpha", "kden-p-proj1")).To(BeEmpty())
	})

	It("prefers the namespace recorded in the status over the recomputed name", func() {
		const projectName = "proj2"
		landscape := NewLandscape("beta", "kden-p-proj2", withFinalizer, func(l *konfidence.Landscape) {
			l.Status.ProjectName = projectName
			l.Status.Namespace = "beta-custom-ns"
		})
		ns := newManagedNamespace("beta-custom-ns", projectName, landscape)
		r, c := newFakeReconciler(landscape, ns)

		_, err := r.reconcileDelete(ctx, landscape)
		Expect(err).NotTo(HaveOccurred())
		Expect(namespaceMissing(ctx, c, "beta-custom-ns")).To(BeTrue())
	})

	It("releases the finalizer immediately when the namespace is absent", func() {
		landscape := NewLandscape("gamma", "kden-p-proj3", withFinalizer, func(l *konfidence.Landscape) {
			l.Status.ProjectName = "proj3"
			l.Status.Namespace = "kden-l-gamma-abcd123"
		})
		r, c := newFakeReconciler(landscape)

		result, err := r.reconcileDelete(ctx, landscape)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsZero()).To(BeTrue())
		Expect(landscapeFinalizers(ctx, c, "gamma", "kden-p-proj3")).To(BeEmpty())
	})

	It("releases the finalizer without deleting an unmanaged same-name namespace", func() {
		const projectName = "proj4"
		landscape := NewLandscape("delta", "kden-p-proj4", withFinalizer, func(l *konfidence.Landscape) {
			l.Status.ProjectName = projectName
			l.Status.Namespace = "kden-l-delta-xyz789"
		})
		unmanaged := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "kden-l-delta-xyz789"},
		}
		r, c := newFakeReconciler(landscape, unmanaged)

		result, err := r.reconcileDelete(ctx, landscape)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsZero()).To(BeTrue())
		Expect(namespaceMissing(ctx, c, unmanaged.Name)).To(BeFalse())
		Expect(landscapeFinalizers(ctx, c, "delta", "kden-p-proj4")).To(BeEmpty())
	})

	It("does nothing for a Landscape without the controller finalizer", func() {
		const projectName = "proj5"
		landscape := NewLandscape("epsilon", "kden-p-proj5")
		ns := newManagedNamespace("kden-l-epsilon-abc456", projectName, landscape)
		r, c := newFakeReconciler(landscape, ns)

		result, err := r.reconcileDelete(ctx, landscape)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsZero()).To(BeTrue())
		Expect(namespaceMissing(ctx, c, ns.Name)).To(BeFalse())
	})
})
