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

// newFakeReconciler builds a ProjectReconciler backed by a fake client seeded
// with the given objects. Unlike envtest, the fake client has no
// namespace-termination semantics, so a deleted namespace disappears
// immediately — letting these specs observe the full delete cycle
// (namespace gone → finalizer released) that envtest cannot reproduce.
func newFakeReconciler(objects ...client.Object) (*ProjectReconciler, client.Client) {
	scheme := runtime.NewScheme()
	Expect(konfidence.AddToScheme(scheme)).To(Succeed())
	Expect(corev1.AddToScheme(scheme)).To(Succeed())

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		WithStatusSubresource(&konfidence.Project{}).
		Build()
	return &ProjectReconciler{Client: c, Scheme: scheme, Recorder: noopRecorder{}}, c
}

// withFinalizer marks a Project as owning the controller finalizer and a UID,
// mirroring a Project that has already been reconciled once.
func withFinalizer(p *konfidence.Project) {
	p.UID = types.UID("uid-" + p.Name)
	p.Finalizers = []string{projectFinalizer}
}

// newManagedNamespace builds a namespace owned by the given Project, as the
// controller would create it (managed labels + controller owner reference).
func newManagedNamespace(name string, project *konfidence.Project) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				pkgctrl.ManagedByLabel:   projectControllerName,
				pkgctrl.ProjectTypeLabel: projectNamespaceTypeValue,
				pkgctrl.ProjectNameLabel: project.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(project, konfidence.GroupVersion.WithKind(konfidence.ProjectKind)),
			},
		},
	}
}

func namespaceMissing(ctx context.Context, c client.Client, name string) bool {
	err := c.Get(ctx, types.NamespacedName{Name: name}, &corev1.Namespace{})
	return apierrors.IsNotFound(err)
}

func projectFinalizers(ctx context.Context, c client.Client, name string) []string {
	project := &konfidence.Project{}
	Expect(c.Get(ctx, types.NamespacedName{Name: name}, project)).To(Succeed())
	return project.Finalizers
}

var _ = Describe("Project reconcileDelete", func() {
	ctx := context.Background()

	It("deletes the managed namespace, then releases the finalizer once it is gone", func() {
		project := NewProject("alpha", withFinalizer)
		ns := newManagedNamespace(konfidence.ProjectNamespacePrefix+"alpha", project)
		r, c := newFakeReconciler(project, ns)

		By("deleting the namespace and requeueing while the deletion is pending")
		result, err := r.reconcileDelete(ctx, project)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(Equal(deletionRequeueInterval))
		Expect(namespaceMissing(ctx, c, ns.Name)).To(BeTrue())
		Expect(projectFinalizers(ctx, c, "alpha")).To(ContainElement(projectFinalizer))

		By("releasing the finalizer on the next pass, once the namespace is gone")
		result, err = r.reconcileDelete(ctx, project)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsZero()).To(BeTrue())
		Expect(projectFinalizers(ctx, c, "alpha")).To(BeEmpty())
	})

	It("prefers the namespace recorded in the status over the recomputed name", func() {
		project := NewProject("beta", withFinalizer, func(p *konfidence.Project) {
			p.Status.Namespace = "beta-custom-ns"
		})
		ns := newManagedNamespace("beta-custom-ns", project)
		r, c := newFakeReconciler(project, ns)

		_, err := r.reconcileDelete(ctx, project)
		Expect(err).NotTo(HaveOccurred())
		Expect(namespaceMissing(ctx, c, "beta-custom-ns")).To(BeTrue())
	})

	It("releases the finalizer immediately when the namespace is absent", func() {
		project := NewProject("gamma", withFinalizer)
		r, c := newFakeReconciler(project)

		result, err := r.reconcileDelete(ctx, project)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsZero()).To(BeTrue())
		Expect(projectFinalizers(ctx, c, "gamma")).To(BeEmpty())
	})

	It("releases the finalizer without deleting an unmanaged same-name namespace", func() {
		project := NewProject("delta", withFinalizer)
		unmanaged := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: konfidence.ProjectNamespacePrefix + "delta"},
		}
		r, c := newFakeReconciler(project, unmanaged)

		result, err := r.reconcileDelete(ctx, project)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsZero()).To(BeTrue())
		Expect(namespaceMissing(ctx, c, unmanaged.Name)).To(BeFalse())
		Expect(projectFinalizers(ctx, c, "delta")).To(BeEmpty())
	})

	It("does nothing for a Project without the controller finalizer", func() {
		project := NewProject("epsilon")
		ns := newManagedNamespace(konfidence.ProjectNamespacePrefix+"epsilon", project)
		r, c := newFakeReconciler(project, ns)

		result, err := r.reconcileDelete(ctx, project)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsZero()).To(BeTrue())
		Expect(namespaceMissing(ctx, c, ns.Name)).To(BeFalse())
	})
})
