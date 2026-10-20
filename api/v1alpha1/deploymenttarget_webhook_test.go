package v1alpha1

import (
	"context"

	controller "github.com/konfidence-project/konfidence/pkg/controller"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("DeploymentTargetValidator", func() {
	var (
		ctx    context.Context
		scheme *runtime.Scheme
	)

	landscapeNamespace := func(name string) *corev1.Namespace {
		return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				controller.ProjectTypeLabel:   "landscape",
				controller.LandscapeNameLabel: name,
			},
		}}
	}
	target := func(name, namespace, artifactType string) *DeploymentTarget {
		return &DeploymentTarget{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec:       DeploymentTargetSpec{ArtifactType: artifactType},
		}
	}

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(AddToScheme(scheme)).To(Succeed())
	})

	Describe("ValidateCreate", func() {
		Context("when the artifact type is unique in a landscape namespace", func() {
			It("should allow creation", func() {
				ns := landscapeNamespace("landscape")
				fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns).Build()
				validator := &DeploymentTargetValidator{Client: fakeClient}

				warnings, err := validator.ValidateCreate(ctx, target("helm", ns.Name, "helm.konfidence.cloud"))
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			})
		})

		Context("when the artifact type already exists in the same namespace", func() {
			It("should reject creation", func() {
				ns := landscapeNamespace("landscape")
				existing := target("existing", ns.Name, "helm.konfidence.cloud")
				fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns, existing).Build()
				validator := &DeploymentTargetValidator{Client: fakeClient}

				warnings, err := validator.ValidateCreate(ctx, target("duplicate", ns.Name, existing.Spec.ArtifactType))
				Expect(err).To(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			})
		})

		Context("when the artifact type exists in another landscape namespace", func() {
			It("should allow creation", func() {
				first := landscapeNamespace("first")
				second := landscapeNamespace("second")
				existing := target("existing", first.Name, "helm.konfidence.cloud")
				fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(first, second, existing).Build()
				validator := &DeploymentTargetValidator{Client: fakeClient}

				warnings, err := validator.ValidateCreate(ctx, target("other", second.Name, existing.Spec.ArtifactType))
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			})
		})

		Context("when the namespace is not a landscape namespace", func() {
			It("should reject creation", func() {
				ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ordinary"}}
				fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns).Build()
				validator := &DeploymentTargetValidator{Client: fakeClient}

				warnings, err := validator.ValidateCreate(ctx, target("target", ns.Name, "helm.konfidence.cloud"))
				Expect(err).To(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			})
		})
	})

	Describe("ValidateUpdate", func() {
		Context("when parameters change in a landscape namespace", func() {
			It("should allow the update", func() {
				ns := landscapeNamespace("landscape")
				oldTarget := target("target", ns.Name, "helm.konfidence.cloud")
				newTarget := oldTarget.DeepCopy()
				newTarget.Spec.Parameters = &Parameters{Kind: "Secret", Name: "updated"}
				fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns, oldTarget).Build()
				validator := &DeploymentTargetValidator{Client: fakeClient}

				warnings, err := validator.ValidateUpdate(ctx, oldTarget, newTarget)
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			})
		})
		Context("when an update creates a duplicate artifact type", func() {
			It("should reject the update", func() {
				ns := landscapeNamespace("landscape")
				oldTarget := target("target", ns.Name, "kustomize.konfidence.cloud")
				existing := target("existing", ns.Name, "helm.konfidence.cloud")
				newTarget := oldTarget.DeepCopy()
				newTarget.Spec.ArtifactType = existing.Spec.ArtifactType
				fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns, oldTarget, existing).Build()
				validator := &DeploymentTargetValidator{Client: fakeClient}

				warnings, err := validator.ValidateUpdate(ctx, oldTarget, newTarget)
				Expect(err).To(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			})
		})

		Context("when the namespace is not a landscape namespace", func() {
			It("should reject the update", func() {
				ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ordinary"}}
				oldTarget := target("target", ns.Name, "helm.konfidence.cloud")
				newTarget := oldTarget.DeepCopy()
				newTarget.Spec.Parameters = &Parameters{Kind: "Secret", Name: "kubeconfig"}
				fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns, oldTarget).Build()
				validator := &DeploymentTargetValidator{Client: fakeClient}

				warnings, err := validator.ValidateUpdate(ctx, oldTarget, newTarget)
				Expect(err).To(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			})
		})
	})
})
